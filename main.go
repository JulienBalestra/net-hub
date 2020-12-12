package main

import (
	"fmt"
	"github.com/JulienBalestra/tcp-hub/cmd/client"
	"github.com/JulienBalestra/tcp-hub/cmd/server"
	"github.com/JulienBalestra/tcp-hub/cmd/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"os"
	"time"
)

func main() {
	zapConfig := zap.NewProductionConfig()
	zapLevel := zapConfig.Level.String()

	root := &cobra.Command{
		Short: "net-hub",
		Long:  "net-hub",
		Use:   "net-hub",
	}
	root.AddCommand(version.NewCommand())
	fs := &pflag.FlagSet{}

	timezone := time.Local.String()

	fs.StringVar(&timezone, "timezone", timezone, "timezone")
	fs.StringVar(&zapLevel, "log-level", zapLevel, fmt.Sprintf("log level - %s %s %s %s %s %s %s", zap.DebugLevel, zap.InfoLevel, zap.WarnLevel, zap.ErrorLevel, zap.DPanicLevel, zap.PanicLevel, zap.FatalLevel))

	root.PersistentFlags().AddFlagSet(fs)
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logger, err := zapConfig.Build()
		if err != nil {
			return err
		}
		logger = logger.With(zap.Int("pid", os.Getpid()))
		zap.ReplaceGlobals(logger)
		zap.RedirectStdLog(logger)

		tz, err := time.LoadLocation(timezone)
		if err != nil {
			return err
		}
		time.Local = tz
		return nil
	}
	root.AddCommand(client.Command())
	root.AddCommand(server.Command())

	exitCode := 0
	err := root.Execute()
	if err != nil {
		exitCode = 1
		zap.L().Error("program exit", zap.Error(err), zap.Int("exitCode", exitCode))
	} else {
		zap.L().Info("program exit", zap.Int("exitCode", exitCode))
	}
	_ = zap.L().Sync()
	os.Exit(exitCode)
}
