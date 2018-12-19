package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kinvolk/karydia"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(karydia.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
