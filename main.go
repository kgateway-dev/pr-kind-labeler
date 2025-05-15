package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v68/github"
	"github.com/spf13/cobra"
)

var (
	// commentRE strips HTML comments so example code isn't parsed.
	commentRE = regexp.MustCompile(`(?s)<!--.*?-->`)
	// kindRE captures /kind labels, case-insensitive.
	kindRE = regexp.MustCompile(`(?i)/kind\s+([a-z0-9_/-]+)`)
	// releaseNoteRE captures the first fenced code block with the word "release-note" in it.
	releaseNoteRE = regexp.MustCompile("(?s)```release-note\\s*(.*?)\\s*```")
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

			// get the PR description body and remove all HTML comments to make it easier to parse
			body := prEvent.GetPullRequest().GetBody()
			sanitizedBody := commentRE.ReplaceAllString(body, "")

			supportedKinds := map[string]bool{
				"design":          true,
				"deprecation":     true,
				"feature":         true,
				"fix":             true,
				"breaking_change": true,
				"documentation":   true,
				"cleanup":         true,
				"flake":           true,
			}

			// extract kinds and verify all kinds are supported. if not, label do-not-merge and exit.
			kindRE := regexp.MustCompile(`(?m)^/kind\s+([\w/-]+)`)
			kinds := map[string]bool{}
			for _, match := range kindRE.FindAllStringSubmatch(sanitizedBody, -1) {
				kinds[match[1]] = true
			}
			for k := range kinds {
				if supportedKinds[k] {
					continue
				}
				if _, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, prNum, []string{"do-not-merge/kind-invalid"}); err != nil {
					return fmt.Errorf("failed to add do-not-merge label: %w", err)
				}
				return fmt.Errorf("invalid /kind %q detected, labeling do-not-merge", k)
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
				kindLabel := "kind/" + k
				if currentMap[kindLabel] {
					continue
				}
				_, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, prNum, []string{kindLabel})
				if err != nil {
					return fmt.Errorf("failed to add label %q: %w", kindLabel, err)
				}
			}
			for label := range currentMap {
				if !strings.HasPrefix(label, "kind/") {
					continue
				}
				kindType := strings.TrimPrefix(label, "kind/")
				if kinds[kindType] {
					continue
				}
				_, err := client.Issues.RemoveLabelForIssue(ctx, owner, repo, prNum, label)
				if err != nil {
					return fmt.Errorf("failed to remove label %q: %w", label, err)
				}
			}

			// Enforce release-note block has been filled out.
			match := releaseNoteRE.FindStringSubmatch(sanitizedBody)
			if len(match) < 2 || strings.TrimSpace(match[1]) == "" {
				// Missing or empty => invalid
				client.Issues.AddLabelsToIssue(ctx, owner, repo, prNum, []string{"do-not-merge/release-note-invalid"})
				return fmt.Errorf("missing or empty ```release-note``` block; please add your line or 'NONE'")
			}
			// Handle the special case "NONE" scenario for changelog types that don't require release
			// notes. Remove any stale labels.
			entry := strings.TrimSpace(match[1])
			if strings.EqualFold(entry, "NONE") {
				client.Issues.AddLabelsToIssue(ctx, owner, repo, prNum, []string{"release-note-none"})
				client.Issues.RemoveLabelForIssue(ctx, owner, repo, prNum, "do-not-merge/release-note-invalid")
				return nil
			}
			// Else, valid entry. Remove invalid label and mark release-note so changelog generation automation
			// can query for this PR easily.
			client.Issues.AddLabelsToIssue(ctx, owner, repo, prNum, []string{"release-note"})
			client.Issues.RemoveLabelForIssue(ctx, owner, repo, prNum, "do-not-merge/release-note-invalid")

			return nil
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
