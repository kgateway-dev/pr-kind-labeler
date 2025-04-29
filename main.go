package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
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
			client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: token,
			})))

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

			// extract kinds
			kindRE := regexp.MustCompile(`(?m)^/kind\s+([\w/-]+)`)
			kinds := map[string]bool{}
			for _, match := range kindRE.FindAllStringSubmatch(body, -1) {
				kinds[match[1]] = true
			}

			// fetch current labels
			current, _, err := client.Issues.ListLabelsByIssue(ctx, owner, repo, prNum, nil)
			if err != nil {
				return fmt.Errorf("failed to list labels: %w", err)
			}
			currentMap := map[string]bool{}
			for _, L := range current {
				currentMap[L.GetName()] = true
			}

			// add missing and remove stale labels
			for k := range kinds {
				if currentMap[k] {
					continue
				}
				_, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, prNum, []string{k})
				if err != nil {
					return fmt.Errorf("failed to add label %q: %w", k, err)
				}
			}
			for label := range currentMap {
				if !kindRE.MatchString("/kind " + label) {
					continue
				}
				if kinds[label] {
					continue
				}
				_, err := client.Issues.RemoveLabelForIssue(ctx, owner, repo, prNum, label)
				if err != nil {
					return fmt.Errorf("failed to remove label %q: %w", label, err)
				}
			}

			// enforce changelog for required kinds
			changelogRE := regexp.MustCompile(`(?is)^###\s*Changelog:`)
			// list of kinds that require a changelog section
			requiresChangelog := map[string]bool{
				"new_feature":     true,
				"bug_fix":         true,
				"breaking_change": true,
			}
			for k := range kinds {
				if !requiresChangelog[k] {
					continue
				}
				if !changelogRE.MatchString(body) {
					return fmt.Errorf("PR is labeled %q but missing a \"### Changelog:\" section", k)
				}
			}

			return nil
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
