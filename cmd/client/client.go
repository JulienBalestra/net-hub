package client

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/JulienBalestra/tcp-hub/cmd/signals"
	"github.com/JulienBalestra/tcp-hub/pkg/client"
	"github.com/JulienBalestra/tcp-hub/pkg/pipe"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	c := &cobra.Command{
		Short:   "client to connect to a remote application and the hub server",
		Long:    "client",
		Use:     "client",
		Aliases: []string{"c"},
	}
	applicationConfig := &client.Config{}
	hubConfig := &client.Config{}

	fs := &pflag.FlagSet{}
	fs.StringVar(&applicationConfig.ServerAddress, "application-address", "", "application server address")
	fs.StringVar(&hubConfig.ServerAddress, "hub-address", "", "hub server address")
	fs.DurationVar(&applicationConfig.BackOffMax, "application-backoff", time.Second, "application server connection max backoff")
	fs.DurationVar(&hubConfig.BackOffMax, "hub-backoff", time.Minute*5, "hub server connection max backoff")
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

		applicationClient := client.NewClient(applicationConfig)
		waitGroup.Add(1)
		go func() {
			err := applicationClient.Run(ctx)
			if err != nil {
				errorsChan <- err
			}
			waitGroup.Done()
			cancel()
		}()

		hubClient := client.NewClient(hubConfig)
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
		return nil
	}
	return c
}
