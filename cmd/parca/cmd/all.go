package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(allCmd)
}

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all parca services from a single binary",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("hello world")
	},
}
