package listener

import (
	"context"
	"net"

	"github.com/JulienBalestra/tcp-hub/pkg/conn"
	"go.uber.org/zap"
)

type Hub struct {
	conf      *Config
	NewConnCh chan *conn.Conn
}

type Config struct {
	ListenAddress string
}

func New(conf *Config) *Hub {
	return &Hub{
		conf:      conf,
		NewConnCh: make(chan *conn.Conn),
	}
}

func (h *Hub) Run(ctx context.Context) error {
	l, err := net.Listen("tcp4", h.conf.ListenAddress)
	if err != nil {
		zap.L().Error("failed to listen", zap.Error(err))
		return err
	}
	zctx := zap.L().With(
		zap.String("listener", l.Addr().String()),
	)
	zctx.Info("successfully started listener")
	for {
		select {
		case <-ctx.Done():
			zctx.Info("context done", zap.String("reason", ctx.Err().Error()))
			return l.Close()

		default:
			newConn, err := l.Accept()
			if err != nil {
				zctx.Error("failed to accept a new connection", zap.Error(err))
				continue
			}
			zctx.With(
				zap.String("localAddr", newConn.LocalAddr().String()),
				zap.String("remoteAddr", newConn.RemoteAddr().String()),
			).Info("new connection")
			nc := conn.NewConn(ctx, newConn)
			h.NewConnCh <- nc
		}
	}
}
