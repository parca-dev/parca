package cmd

import (
	"github.com/spf13/cobra"
)

var (
	configPath string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config.file", "c", "parca.yaml", "Path to config file")
}

var rootCmd = &cobra.Command{
	Use:   "parca",
	Short: "parca is a continuous profiling server and storage engine",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute the command
func Execute() error {
	return rootCmd.Execute()
}
