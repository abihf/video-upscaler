/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"os"

	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
)

var rootFlags struct {
	redisAddr string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "video-upscaler",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	SilenceUsage: true,
}

func Execute(ctx context.Context) {
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootFlags.redisAddr, "redis-addr", "r", getEnv("REDIS_ADDR", "localhost:6379"), "redis address host:port")
	rootCmd.RegisterFlagCompletionFunc("redis-addr", cobra.FixedCompletions(nil, cobra.ShellCompDirectiveNoFileComp))
}

func redisConn() asynq.RedisConnOpt {
	return &asynq.RedisClientOpt{
		Addr: rootFlags.redisAddr,
	}
}
