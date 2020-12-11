package pipe

import (
	"context"
	"io"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/JulienBalestra/tcp-hub/pkg/conn"
	"go.uber.org/zap"
)

type Config struct {
	ByteBufferSize int
}

type Pipe struct {
	conf *Config

	hubConn, externalConn         *conn.Conn
	hubBuf, externalBuf           []byte
	hubBufIndex, externalBufIndex int
	noDeadline                    time.Time
}

func New(conf *Config) *Pipe {
	return &Pipe{
		conf:        conf,
		hubBuf:      make([]byte, conf.ByteBufferSize),
		externalBuf: make([]byte, conf.ByteBufferSize),
	}
}

func (p *Pipe) closeConnections() {
	_ = closeConn(p.hubConn)
	_ = closeConn(p.externalConn)
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
	var zctx *zap.Logger
	for {
		select {
		case <-ctx.Done():
			p.closeConnections()
			return nil

		case hubConn := <-hubConnCh:
			zctx = zap.L().With(
				zap.String("hubRemoteAddr", hubConn.RemoteAddr().String()),
				zap.String("hubLocalAddr", hubConn.LocalAddr().String()),
			)
			if p.hubConn != nil {
				zctx.Info("replacing connection")
				_ = closeConn(p.hubConn)
			}
			p.hubConn = hubConn
			if p.externalConn == nil {
				zctx.Debug("missing external connection")
				continue
			}
			zctx = zctx.With(
				zap.String("externalRemoteAddr", p.externalConn.RemoteAddr().String()),
				zap.String("externalLocalAddr", p.externalConn.LocalAddr().String()),
			)
			readFromExternal, err := connRead(p.externalConn.Conn, p.externalBuf)
			if err != nil {
				_ = closeConn(p.externalConn)
				p.externalConn = nil
				zctx.Warn("external connection is already closed", zap.Error(err))
				continue
			}
			p.externalBufIndex = readFromExternal

		case externalConn := <-externalConnCh:
			zctx = zap.L().With(
				zap.String("externalRemoteAddr", externalConn.RemoteAddr().String()),
				zap.String("externalLocalAddr", externalConn.LocalAddr().String()),
			)
			if p.externalConn != nil {
				zctx.Info("replacing connection")
				_ = closeConn(p.externalConn)
			}
			p.externalConn = externalConn
			if p.hubConn == nil {
				zctx.Debug("missing hub connection")
				continue
			}
			zctx = zctx.With(
				zap.String("hubRemoteAddr", p.hubConn.RemoteAddr().String()),
				zap.String("hubLocalAddr", p.hubConn.LocalAddr().String()),
			)
			readFromHub, err := connRead(p.hubConn.Conn, p.hubBuf)
			if err != nil {
				_ = closeConn(p.hubConn)
				p.hubConn = nil
				zctx.Warn("hub connection is already closed", zap.Error(err))
				continue
			}
			p.hubBufIndex = readFromHub
		}
		zctx.Info("start piping")
		once := &sync.Once{}
		go p.copyBuffer(p.hubConn, p.externalConn, p.externalBuf, p.externalBufIndex, once)
		go p.copyBuffer(p.externalConn, p.hubConn, p.hubBuf, p.hubBufIndex, once)
		<-p.hubConn.Context.Done()
		p.hubBufIndex = -1
		p.hubConn = nil
		<-p.externalConn.Context.Done()
		p.externalBufIndex = -1
		p.externalConn = nil
	}
}

func (p *Pipe) isClosed(c net.Conn, buf []byte) bool {
	_ = c.SetReadDeadline(time.Now())
	_, err := c.Read(buf[:1])
	return err != nil && err == io.EOF
}

func (p *Pipe) copyBuffer(dst, src net.Conn, buf []byte, pendingRead int, once *sync.Once) {
	defer once.Do(p.closeConnections)
	_ = src.SetReadDeadline(p.noDeadline)
	_ = dst.SetWriteDeadline(p.noDeadline)
	zctx := zap.L().With(
		zap.String("dstLocalAddr", dst.LocalAddr().String()),
		zap.String("dstRemoteAddr", dst.RemoteAddr().String()),
		zap.String("srcLocalAddr", src.LocalAddr().String()),
		zap.String("srcRemoteAddr", src.RemoteAddr().String()),
		zap.Int("pendingRead", pendingRead),
	)
	zctx.Debug("starting to copy data")
	if pendingRead > 0 {
		i, err := dst.Write(buf[:pendingRead])
		if err != nil {
			zctx.Error("failed to transfer pending data", zap.Error(err), zap.Int("pendingReadWrote", i))
			return
		}
		zctx.Info("transferred pending data", zap.Error(err), zap.Int("pendingReadWrote", i))
	}
	i, err := io.CopyBuffer(dst, src, buf)
	if err != nil && i == 0 {
		// TODO fixme this could still happen
		zctx.Error("failed to transfer data", zap.Error(err))
		return
	}
	zctx.Info("transferred data", zap.Int64("bytes", i))
}

func connRead(conn net.Conn, buf []byte) (int, error) {
	var n int
	rawConn, err := conn.(syscall.Conn).SyscallConn()
	if err != nil {
		return n, err
	}

	var sysErr error
	err = rawConn.Read(func(fd uintptr) bool {
		n, err = syscall.Read(int(fd), buf)
		sysErr = err
		switch {
		case n == 0 && err == nil:
			sysErr = io.EOF
		case err == syscall.EAGAIN:
			sysErr = nil
		case err == syscall.EWOULDBLOCK:
			sysErr = nil
		}
		if n == -1 {
			n = 0
		}
		return true
	})
	if err != nil {
		return n, err
	}
	return n, sysErr
}
