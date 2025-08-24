/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/abihf/video-upscaler/internal/task"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan directory",
	Short: "A brief description of your command",

	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		return task.Scan(cmd.Context(), args[0])
	},
	ValidArgsFunction: cobra.FixedCompletions(nil, cobra.ShellCompDirectiveFilterDirs),
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
