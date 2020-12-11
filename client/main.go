package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/JulienBalestra/tcp-hub/cmd/signals"
	"github.com/JulienBalestra/tcp-hub/pkg/client"
	"github.com/JulienBalestra/tcp-hub/pkg/pipe"
	"go.uber.org/zap"
)

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

	applicationClient := client.NewClient(&client.Config{
		ServerAddress: "127.0.0.1:80",
		BackOffMax:    time.Second,
	})
	waitGroup.Add(1)
	go func() {
		err := applicationClient.Run(ctx)
		if err != nil {
			errorsChan <- err
		}
		waitGroup.Done()
		cancel()
	}()

	// TODO fixme
	hubClient := client.NewClient(&client.Config{
		ServerAddress: "127.0.0.1:9001",
		BackOffMax:    time.Minute * 5,
	})
	waitGroup.Add(1)
	go func() {
		err := hubClient.Run(ctx)
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
		err := p.Attach(ctx, hubClient.NewConnCh, applicationClient.NewConnCh)
		if err != nil {
			errorsChan <- err
		}
		waitGroup.Done()
		cancel()
	}()

	select {
	case <-ctx.Done():
	case err := <-errorsChan:
		zap.L().Error("failed to run clients", zap.Error(err))
	}
	cancel()
	waitGroup.Wait()
}
