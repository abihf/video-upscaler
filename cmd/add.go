/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/abihf/video-upscaler/internal/task"
	"github.com/spf13/cobra"
)

var addFlags struct {
	priority string
	force    bool
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVarP(&addFlags.force, "force", "f", false, "Remove old task when queue conflicts")
	addCmd.Flags().StringVarP(&addFlags.priority, "priority", "p", "default", "Queue priority (default|critical|low)")
	addCmd.RegisterFlagCompletionFunc("priority", cobra.FixedCompletions([]string{"default", "critical", "low"}, cobra.ShellCompDirectiveDefault))
}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [-p priority] [-f] input-file.mkv output-file.mkv",
	Short: "Add file to queue for upscale",
	Args:  cobra.ExactArgs(2),

	DisableFlagsInUseLine: true,
	ValidArgsFunction:     cobra.FixedCompletions([]string{"mkv"}, cobra.ShellCompDirectiveFilterFileExt),

	RunE: func(cmd *cobra.Command, args []string) error {

		return task.Add(cmd.Context(), args[0], args[1], addFlags.priority, addFlags.force)
	},
}
