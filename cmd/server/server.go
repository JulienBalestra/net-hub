package server

import (
	"context"
	"sync"

	"github.com/JulienBalestra/tcp-hub/cmd/signals"
	"github.com/JulienBalestra/tcp-hub/pkg/listener"
	"github.com/JulienBalestra/tcp-hub/pkg/pipe"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	c := &cobra.Command{
		Short:   "hub server",
		Long:    "server",
		Use:     "server",
		Aliases: []string{"h"},
	}
	hubListenerConfig := &listener.Config{}
	externalClientConfig := &listener.Config{}

	fs := &pflag.FlagSet{}
	fs.StringVar(&hubListenerConfig.ListenAddress, "server-listener", "0.0.0.0:9001", "hub server listener")
	fs.StringVar(&externalClientConfig.ListenAddress, "external-listener", "0.0.0.0:9000", "external client listener")
	c.Flags().AddFlagSet(fs)
	c.RunE = func(cmd *cobra.Command, args []string) error {

		ctx, cancel := context.WithCancel(context.TODO())
		waitGroup := sync.WaitGroup{}

		waitGroup.Add(1)
		go func() {
			signals.NotifySignals(ctx, cancel)
			waitGroup.Done()
		}()

		errorsChan := make(chan error)
		defer close(errorsChan)

		hubListener := listener.New(hubListenerConfig)
		externalClientListener := listener.New(externalClientConfig)

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
			err := externalClientListener.Run(ctx)
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
			err := p.Attach(ctx, hubListener.NewConnCh, externalClientListener.NewConnCh)
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
		// TODO fixme
		//waitGroup.Wait()
		return nil
	}
	return c
}
