package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
)

func main() {
	cmd := cobra.Command{
		Use:   "pr-kind-labeler",
		Short: "Sync /kind commands in PR body to GitHub labels and enforce changelog notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// token := args[0]
			eventPath := os.Getenv("GITHUB_EVENT_PATH")
			payload, err := os.ReadFile(eventPath)
			if err != nil {
				return fmt.Errorf("failed to read event path: %w", err)
			}

			var prEvent github.PullRequestEvent
			if err := json.Unmarshal(payload, &prEvent); err != nil {
				return fmt.Errorf("failed to parse event JSON: %w", err)
			}

			fmt.Println(prEvent.PullRequest.Body)
			return nil
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
