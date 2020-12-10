package main

import (
	"context"
	"os"
	"sync"

	"github.com/JulienBalestra/tcp-hub/cmd/signals"
	"github.com/JulienBalestra/tcp-hub/pkg/listener"
	"github.com/JulienBalestra/tcp-hub/pkg/pipe"
	"go.uber.org/zap"
)

/*
	Hub: client registered

*/

func main() {
	zapConfig := zap.NewProductionConfig()
	logger, err := zapConfig.Build()
	if err != nil {
		panic(err)
	}
	logger = logger.With(zap.Int("pid", os.Getpid()))
	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)

	ctx, cancel := context.WithCancel(context.TODO())
	waitGroup := sync.WaitGroup{}

	waitGroup.Add(1)
	go func() {
		signals.NotifySignals(ctx, cancel)
		waitGroup.Done()
	}()

	errorsChan := make(chan error)
	defer close(errorsChan)

	hubListener := listener.New(&listener.Config{
		ListenAddress: ":9001",
	})
	clientListener := listener.New(&listener.Config{
		ListenAddress: ":9000",
	})

	waitGroup.Add(1)
	go func() {
		err := hubListener.Run(ctx)
		if err != nil {
			errorsChan <- err
		}
		waitGroup.Done()
		cancel()
	}()

	waitGroup.Add(1)
	go func() {
		err := clientListener.Run(ctx)
		if err != nil {
			errorsChan <- err
		}
		waitGroup.Done()
		cancel()
	}()

	p := pipe.New(&pipe.Config{
		ByteBufferSize: 65535,
	})
	waitGroup.Add(1)
	go func() {
		err := p.Attach(ctx, hubListener.NewConnCh, clientListener.NewConnCh)
		if err != nil {
			errorsChan <- err
		}
		waitGroup.Done()
		cancel()
	}()

	select {
	case <-ctx.Done():
	case err := <-errorsChan:
		zap.L().Error("failed to run listeners", zap.Error(err))
	}
	cancel()
	//waitGroup.Wait()
}
