package gonet

import (
	"context"
	"errors"
	"net"
	"sync"
)

type Manager struct {
	used         sync.Map
	passiveConns sync.Map
}

var (
	m *Manager
)

func InitManager(passiveConns sync.Map) {
	m = &Manager{passiveConns: passiveConns}
}

func ListenTCP(port int) (net.Listener, error) {
	if _, ok := m.used.Load(port); !ok {
		m.used.Store(port, struct{}{})
		if v, ok := m.passiveConns.Load(port); ok {
			if conns, ok := v.(chan net.Conn); ok {
				return NewGoListener(context.Background(),
					port,
					func() { m.used.Delete(port) },
					conns), nil
			}
		}
	}

	return nil, errors.New("port is already in use")
}

var (
	ErrGoListenerClosed = errors.New("go listener is closed")
)

var _ net.Listener = (*GoListener)(nil)

type GoListener struct {
	ctx   context.Context
	conns <-chan net.Conn

	closed  chan struct{}
	closedf func()
	port    int
}

func NewGoListener(ctx context.Context,
	port int,
	closedf func(),
	conns <-chan net.Conn) net.Listener {
	return &GoListener{
		ctx:    ctx,
		port:   port,
		conns:  conns,
		closed: make(chan struct{}),
	}
}

func (g *GoListener) Accept() (net.Conn, error) {
	select {
	case _, ok := <-g.closed:
		if !ok {
			return nil, ErrGoListenerClosed
		}
	case <-g.ctx.Done():
		return nil, g.ctx.Err()
	case c := <-g.conns:
		return c, nil
	}
	return nil, nil
}

func (g *GoListener) Close() error {
	close(g.closed)
	if g.closedf != nil {
		g.closedf()
	}
	return nil
}

func (g *GoListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero, Port: g.port}
}
