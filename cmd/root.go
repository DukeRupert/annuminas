package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dukerupert/annuminas/pkg/dockerhub"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var client *dockerhub.Client

var rootCmd = &cobra.Command{
	Use:   "annuminas",
	Short: "A CLI tool for managing Docker Hub repositories",
	Long:  `Annuminas manages Docker Hub repositories via the Docker Hub API.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initClient()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initClient() error {
	// Try ~/.dotfiles/.env first, fall back to .env in current directory
	dotfilePath := filepath.Join(os.Getenv("HOME"), ".dotfiles", ".env")
	if _, err := os.Stat(dotfilePath); err == nil {
		_ = godotenv.Load(dotfilePath)
	} else {
		_ = godotenv.Load(".env")
	}

	username := os.Getenv("DOCKERHUB_USERNAME")
	if username == "" {
		return fmt.Errorf("DOCKERHUB_USERNAME must be set in ~/.dotfiles/.env or .env")
	}

	token := os.Getenv("DOCKERHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("DOCKERHUB_TOKEN must be set in ~/.dotfiles/.env or .env")
	}

	client = dockerhub.NewClient(username, token)
	return nil
}
