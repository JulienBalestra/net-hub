package conn

import (
	"context"
	"net"
)

type Conn struct {
	net.Conn
	Context context.Context
	Cancel  func()
}

func NewConn(ctx context.Context, c net.Conn) *Conn {
	connContext, cancel := context.WithCancel(ctx)
	return &Conn{
		Conn:    c,
		Context: connContext,
		Cancel:  cancel,
	}
}
