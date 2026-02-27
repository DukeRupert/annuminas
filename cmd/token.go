package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage Docker Hub personal access tokens",
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new personal access token",
	Long: `Create a new Docker Hub personal access token.

Valid scopes: repo:admin, repo:write, repo:read, repo:public_read
Higher scopes include lower ones (e.g. repo:write implies repo:read).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		label, _ := cmd.Flags().GetString("label")
		scopes, _ := cmd.Flags().GetStringSlice("scopes")

		token, err := client.CreateAccessToken(label, scopes)
		if err != nil {
			return err
		}

		fmt.Printf("Token created: %s\n", token.TokenLabel)
		fmt.Printf("UUID:          %s\n", token.UUID)
		fmt.Printf("Scopes:        %s\n", strings.Join(token.Scopes, ", "))
		fmt.Printf("Token:         %s\n", token.Token)
		fmt.Println("\nSave this token now — it cannot be retrieved later.")
		return nil
	},
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all personal access tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		tokens, err := client.ListAccessTokens()
		if err != nil {
			return err
		}

		if len(tokens) == 0 {
			fmt.Println("No personal access tokens found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "UUID\tLABEL\tSCOPES\tACTIVE\tCREATED\tLAST USED")
		fmt.Fprintln(w, "────\t─────\t──────\t──────\t───────\t─────────")
		for _, t := range tokens {
			lastUsed := t.LastUsed
			if lastUsed == "" {
				lastUsed = "never"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%s\t%s\n",
				t.UUID,
				t.TokenLabel,
				strings.Join(t.Scopes, ","),
				t.IsActive,
				t.CreatedAt,
				lastUsed,
			)
		}
		return w.Flush()
	},
}

var tokenDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a personal access token by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		uuid, _ := cmd.Flags().GetString("uuid")

		if err := client.DeleteAccessToken(uuid); err != nil {
			return err
		}

		fmt.Printf("Token deleted: %s\n", uuid)
		return nil
	},
}

func init() {
	tokenCreateCmd.Flags().String("label", "", "Friendly name for the token (required)")
	tokenCreateCmd.MarkFlagRequired("label")
	tokenCreateCmd.Flags().StringSlice("scopes", []string{"repo:read"}, "Comma-separated scopes (repo:admin, repo:write, repo:read, repo:public_read)")

	tokenDeleteCmd.Flags().String("uuid", "", "UUID of the token to delete (required)")
	tokenDeleteCmd.MarkFlagRequired("uuid")

	tokenCmd.AddCommand(tokenCreateCmd)
	tokenCmd.AddCommand(tokenListCmd)
	tokenCmd.AddCommand(tokenDeleteCmd)
	rootCmd.AddCommand(tokenCmd)
}
