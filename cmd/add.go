/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/abihf/video-upscaler/internal/queue"
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
		in, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		if !fileExists(in) {
			return fmt.Errorf("input file %s not exist", in)
		}

		out, err := filepath.Abs(args[1])
		if err != nil {
			return err
		}
		if fileExists(out) {
			return fmt.Errorf("output file %s already exist", out)
		}
		return queue.Add(cmd.Context(), redisConn(), in, out, addFlags.priority, addFlags.force)
	},
}
