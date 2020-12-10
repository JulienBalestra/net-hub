package client

import (
	"context"
	"github.com/jpillora/backoff"
	"net"
	"time"

	"github.com/JulienBalestra/tcp-hub/pkg/conn"
	"go.uber.org/zap"
)

type Config struct {
	ServerAddress string
	BackOffMax    time.Duration
}

type Client struct {
	conf *Config

	NewConnCh chan *conn.Conn
	backoff   *backoff.Backoff
}

func NewClient(conf *Config) *Client {
	return &Client{
		conf:      conf,
		NewConnCh: make(chan *conn.Conn),
		backoff: &backoff.Backoff{
			Factor: 1.3,
			Jitter: true,
			Min:    time.Millisecond * 100,
			Max:    conf.BackOffMax,
		},
	}
}

func (c *Client) Run(ctx context.Context) error {
	zctx := zap.L().With(
		zap.String("serverAddress", c.conf.ServerAddress),
	)
	for {
		select {
		case <-ctx.Done():
			zctx.Info("context done", zap.String("reason", ctx.Err().Error()))
			return nil

		default:
			newConn, err := net.Dial("tcp4", c.conf.ServerAddress)
			if err != nil {
				sleepDuration := c.backoff.Duration()
				zap.L().Error("failed to dial", zap.Error(err), zap.String("backoff", sleepDuration.String()))
				time.Sleep(sleepDuration)
				continue
			}
			c.backoff.Reset()
			zctx.Info("new connection")
			nc := conn.NewConn(ctx, newConn)
			c.NewConnCh <- nc
			<-nc.Context.Done()
		}
	}
}
