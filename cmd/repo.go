package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage Docker Hub repositories",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories in the namespace",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		if namespace == "" {
			namespace = client.Username
		}

		repos, err := client.ListRepos(namespace)
		if err != nil {
			return err
		}

		if len(repos) == 0 {
			fmt.Println("No repositories found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tPRIVATE\tSTARS\tPULLS\tLAST UPDATED")
		fmt.Fprintln(w, "────\t───────\t─────\t─────\t────────────")
		for _, r := range repos {
			fmt.Fprintf(w, "%s\t%t\t%d\t%d\t%s\n",
				r.Name,
				r.IsPrivate,
				r.StarCount,
				r.PullCount,
				r.LastUpdated,
			)
		}
		return w.Flush()
	},
}

var repoGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get details for a specific repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		if namespace == "" {
			namespace = client.Username
		}

		repo, err := client.GetRepo(namespace, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Name:           %s\n", repo.Name)
		fmt.Printf("Namespace:      %s\n", repo.Namespace)
		fmt.Printf("Description:    %s\n", repo.Description)
		fmt.Printf("Private:        %t\n", repo.IsPrivate)
		fmt.Printf("Stars:          %d\n", repo.StarCount)
		fmt.Printf("Pulls:          %d\n", repo.PullCount)
		fmt.Printf("Last Updated:   %s\n", repo.LastUpdated)
		fmt.Printf("Date Registered: %s\n", repo.DateRegistered)
		return nil
	},
}

var repoCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		if namespace == "" {
			namespace = client.Username
		}
		name, _ := cmd.Flags().GetString("name")
		private, _ := cmd.Flags().GetBool("private")
		description, _ := cmd.Flags().GetString("description")

		repo, err := client.CreateRepo(namespace, name, description, private)
		if err != nil {
			return err
		}

		fmt.Printf("Repository created: %s/%s\n", repo.Namespace, repo.Name)
		return nil
	},
}

var repoDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a repository by name",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		if namespace == "" {
			namespace = client.Username
		}
		name, _ := cmd.Flags().GetString("name")

		if err := client.DeleteRepo(namespace, name); err != nil {
			return err
		}

		fmt.Printf("Repository deleted: %s/%s\n", namespace, name)
		return nil
	},
}

var repoEnsureCmd = &cobra.Command{
	Use:   "ensure",
	Short: "Create a repository if it doesn't exist (idempotent)",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		if namespace == "" {
			namespace = client.Username
		}
		name, _ := cmd.Flags().GetString("name")

		if err := client.EnsureRepo(namespace, name); err != nil {
			return err
		}

		fmt.Printf("Repository ensured: %s/%s\n", namespace, name)
		return nil
	},
}

func init() {
	// Persistent flag on parent — inherited by all subcommands
	repoCmd.PersistentFlags().String("namespace", "", "Namespace (user or org); defaults to DOCKERHUB_USERNAME")

	// repo create flags
	repoCreateCmd.Flags().String("name", "", "Repository name (required)")
	repoCreateCmd.MarkFlagRequired("name")
	repoCreateCmd.Flags().Bool("private", false, "Whether the repository is private")
	repoCreateCmd.Flags().String("description", "", "Short description for the repository")

	// repo delete flags
	repoDeleteCmd.Flags().String("name", "", "Repository name (required)")
	repoDeleteCmd.MarkFlagRequired("name")

	// repo ensure flags
	repoEnsureCmd.Flags().String("name", "", "Repository name (required)")
	repoEnsureCmd.MarkFlagRequired("name")

	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoGetCmd)
	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoDeleteCmd)
	repoCmd.AddCommand(repoEnsureCmd)
	rootCmd.AddCommand(repoCmd)
}
