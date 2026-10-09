package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from AtonixCorp",
	Run: func(cmd *cobra.Command, args []string) {
		os.Remove("session.jwt")
		fmt.Println("Logged out successfully.")
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
