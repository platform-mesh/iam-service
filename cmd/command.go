package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{ // nolint: gochecknoglobals
	Use:   "iam",
	Short: "IAM Service",
	Long:  `IAM Service binary to run the IAM Servive`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func init() { // nolint: gochecknoinits
	InitServeCmd(rootCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
