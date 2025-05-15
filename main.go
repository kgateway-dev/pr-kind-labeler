package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/go-github/v68/github"
	"github.com/spf13/cobra"

	"github.com/kgateway-dev/pr-kind-labeler/internal/labeler"
)

func main() {
	cmd := cobra.Command{
		Use:   "pr-kind-labeler",
		Short: "Sync /kind commands in PR body to GitHub labels and enforce changelog notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// verify the token is set and create GH API client
			token := os.Args[1]
			if token == "" {
				return fmt.Errorf("input token is not set")
			}
			client := github.NewClient(nil).WithAuthToken(token)

			eventPath := os.Getenv("GITHUB_EVENT_PATH")
			payload, err := os.ReadFile(eventPath)
			if err != nil {
				return fmt.Errorf("failed to read event path: %w", err)
			}
			var prEvent github.PullRequestEvent
			if err := json.Unmarshal(payload, &prEvent); err != nil {
				return fmt.Errorf("failed to parse event JSON: %w", err)
			}

			owner := prEvent.GetRepo().GetOwner().GetLogin()
			repo := prEvent.GetRepo().GetName()
			prNum := prEvent.GetNumber()
			body := prEvent.GetPullRequest().GetBody()

			l := labeler.New(client, owner, repo, prNum)
			if err := l.ProcessPR(ctx, body); err != nil {
				return err
			}

			return nil
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
