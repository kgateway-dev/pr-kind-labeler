package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v68/github"
	"github.com/spf13/cobra"

	"github.com/kgateway-dev/pr-kind-labeler/internal/labeler"
)

func main() {
	cmd := cobra.Command{
		Use:          "pr-kind-labeler",
		Short:        "Sync /kind commands in PR body to GitHub labels and enforce changelog notes",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// verify the token is set and create GH API client
			token := os.Args[1]
			if token == "" {
				return fmt.Errorf("input token is not set")
			}
			client := github.NewClient(nil).WithAuthToken(token)

			if ghprEnv := os.Getenv("GHPR"); ghprEnv != "" {
				// You can manually test, like so:
				// GHPR=kgateway-dev/kgateway/11221 go run . $GITHUB_API_TOKEN
				parts := strings.Split(ghprEnv, "/")
				if len(parts) != 3 {
					return fmt.Errorf("invalid PR format, expected owner/repo/PR")
				}
				owner := parts[0]
				repo := parts[1]
				prNum := parts[2]
				prNumInt, err := strconv.Atoi(prNum)
				if err != nil {
					return fmt.Errorf("invalid PR number: %w", err)
				}
				return manualTest(ctx, client, owner, repo, prNumInt)
			}

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
			if err := l.ProcessPR(ctx, body, true); err != nil {
				return err
			}

			return nil
		},
	}
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func manualTest(ctx context.Context, client *github.Client, owner, repo string, prNum int) error {

	prResp, _, err := client.PullRequests.Get(ctx, owner, repo, prNum)
	if err != nil {
		return fmt.Errorf("failed to get PR body: %w", err)
	}
	body := prResp.GetBody()

	l := labeler.New(client, owner, repo, prNum)
	return l.ProcessPR(ctx, body, false)
}
