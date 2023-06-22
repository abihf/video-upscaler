/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/abihf/video-upscaler/internal/queue"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
)

var addFlags struct {
	priority string
}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [-p priority] input-file.mkv output-file.mkv",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := asynq.NewClient(redisConn())
		in := args[0]
		if !fileExists(in) {
			return fmt.Errorf("input file %s not exist", in)
		}

		out := args[1]
		if fileExists(out) {
			return fmt.Errorf("output file %s already exist", out)
		}
		return queue.Add(cmd.Context(), client, in, out, addFlags.priority)
	},
	DisableFlagsInUseLine: true,
	ValidArgsFunction:     cobra.FixedCompletions([]string{"mkv"}, cobra.ShellCompDirectiveFilterFileExt),
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addFlags.priority, "priority", "p", "default", "Queue priority (default|critical|low)")
	addCmd.RegisterFlagCompletionFunc("priority", cobra.FixedCompletions([]string{"default", "critical", "low"}, cobra.ShellCompDirectiveDefault))
}
