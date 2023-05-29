package gonet

import (
	"context"
	"errors"
	"fmt"
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
	m.used.Range(func(key, value interface{}) bool {
		fmt.Println(key, value)
		return true
	})

	if _, ok := m.used.Load(port); !ok {
		m.used.Store(port, struct{}{})
		if v, ok := m.passiveConns.Load(port); ok {
			if conns, ok := v.(chan net.Conn); ok {
				return NewGoListener(context.Background(),
					port,
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

	closed chan struct{}
	port   int
}

func NewGoListener(ctx context.Context,
	port int,
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
	if m != nil {
		m.used.Delete(g.port)
	}

	close(g.closed)
	return nil
}

func (g *GoListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero, Port: g.port}
}
