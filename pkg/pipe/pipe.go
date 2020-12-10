package pipe

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/JulienBalestra/tcp-hub/pkg/conn"
	"go.uber.org/zap"
)

type Config struct {
	ByteBufferSize int
}

type Pipe struct {
	hubConn, external   *conn.Conn
	hubBuf, externalBuf []byte
}

func New(conf *Config) *Pipe {
	return &Pipe{
		hubBuf:      make([]byte, conf.ByteBufferSize),
		externalBuf: make([]byte, conf.ByteBufferSize),
	}
}

func (p *Pipe) closeConnections() {
	_ = closeConn(p.hubConn)
	_ = closeConn(p.external)
}

func closeConn(c *conn.Conn) error {
	if c == nil {
		return nil
	}
	c.Cancel()
	err := c.Close()
	if err != nil {
		zap.L().Debug("error while closing connection", zap.Error(err))
	}
	return err
}

// Attach
func (p *Pipe) Attach(ctx context.Context, hubConnCh, externalConnCh chan *conn.Conn) error {
	for {
		select {
		case <-ctx.Done():
			p.closeConnections()
			return nil

		case hubConn := <-hubConnCh:
			zctx := zap.L().With(
				zap.String("connection", hubConn.LocalAddr().String()),
			)
			if p.hubConn != nil {
				zctx.Info("replacing connection")
				_ = closeConn(p.hubConn)
			}
			p.hubConn = hubConn
			if p.external == nil {
				zctx.Debug("missing connection")
				continue
			}

		case external := <-externalConnCh:
			zctx := zap.L().With(
				zap.String("connection", external.LocalAddr().String()),
			)
			if p.external != nil {
				zctx.Info("replacing connection")
				_ = closeConn(p.external)
			}
			p.external = external
			if p.hubConn == nil {
				zctx.Debug("missing connection")
				continue
			}
		}
		once := &sync.Once{}
		go p.copyBuffer(p.hubConn, p.external, p.hubBuf, once)
		go p.copyBuffer(p.external, p.hubConn, p.externalBuf, once)

		<-p.hubConn.Context.Done()
		p.hubConn = nil
		<-p.external.Context.Done()
		p.external = nil
	}
}

func (p *Pipe) copyBuffer(dst, src net.Conn, buf []byte, once *sync.Once) {
	zctx := zap.L().With(
		zap.String("dst", dst.LocalAddr().String()),
		zap.String("src", src.LocalAddr().String()),
	)
	zctx.Debug("starting to copy data")
	i, err := io.CopyBuffer(dst, src, buf)
	zctx.Info("copied data", zap.Int64("bytes", i))
	if err != nil {
		zctx.Debug("copied data with error", zap.Error(err))
	}
	once.Do(p.closeConnections)
}
