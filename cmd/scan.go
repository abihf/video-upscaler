/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/abihf/video-upscaler/internal/scanner"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan directory",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cli := asynq.NewClient(redisConn())
		s := scanner.Scanner{Root: args[0], AsynqClient: cli}
		return s.Scan(cmd.Context())
	},
	ValidArgsFunction: cobra.FixedCompletions(nil, cobra.ShellCompDirectiveFilterDirs),
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
