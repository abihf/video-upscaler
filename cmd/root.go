/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cobra.OnInitialize(initConfig)

	// getEnv("REDIS_ADDR", "localhost:6379")
	rootCmd.PersistentFlags().StringP("redis-addr", "r", "localhost:6379", "redis address host:port")
	rootCmd.RegisterFlagCompletionFunc("redis-addr", cobra.FixedCompletions(nil, cobra.ShellCompDirectiveNoFileComp))
	viper.BindPFlags(rootCmd.PersistentFlags())
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "video-upscaler [command]",
	Short:        "A brief description of your application",
	SilenceUsage: true,
}

func Execute(ctx context.Context) {
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$XDG_CONFIG_HOME/video-upscaler")
	viper.AddConfigPath("$HOME/.config/video-upscaler")

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	viper.ReadInConfig()
}

func redisConn() asynq.RedisConnOpt {
	return &asynq.RedisClientOpt{
		Addr: viper.GetString("redis-addr"),
	}
}
