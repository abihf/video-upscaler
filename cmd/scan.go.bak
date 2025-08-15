/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"path/filepath"

	"github.com/abihf/video-upscaler/internal/scanner"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan directory",
	Short: "A brief description of your command",

	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		s := scanner.Scanner{Root: root, Conn: redisConn()}
		return s.Scan(cmd.Context())
	},
	ValidArgsFunction: cobra.FixedCompletions(nil, cobra.ShellCompDirectiveFilterDirs),
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
