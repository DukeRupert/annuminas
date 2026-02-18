package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Verify credentials and connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.Ping(); err != nil {
			return fmt.Errorf("ping failed: %w", err)
		}
		fmt.Println("Credentials valid. Connection successful.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
