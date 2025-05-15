package labeler

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v68/github"
)

var (
	// commentRE strips HTML comments so example code isn't parsed.
	commentRE = regexp.MustCompile(`(?s)<!--.*?-->`)
	// kindRE captures /kind labels, case-insensitive.
	kindRE = regexp.MustCompile(`(?i)/kind\s+([a-z0-9_/-]+)`)
	// releaseNoteRE captures the first fenced code block with the word "release-note" in it.
	releaseNoteRE = regexp.MustCompile("(?s)```release-note\\s*(.*?)\\s*```")
)

// labeler handles PR labeling operations.
type labeler struct {
	client *github.Client
	owner  string
	repo   string
	prNum  int
}

// New creates a new Labeler instance.
func New(client *github.Client, owner, repo string, prNum int) *labeler {
	return &labeler{
		client: client,
		owner:  owner,
		repo:   repo,
		prNum:  prNum,
	}
}

// ProcessPR processes the PR body and updates labels accordingly.
func (l *labeler) ProcessPR(ctx context.Context, body string) error {
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

	// extract kinds and verify all kinds are supported
	kindRE := regexp.MustCompile(`(?m)^/kind\s+([\w/-]+)`)
	kinds := map[string]bool{}
	for _, match := range kindRE.FindAllStringSubmatch(sanitizedBody, -1) {
		kinds[match[1]] = true
	}

	// verify kinds
	for k := range kinds {
		if supportedKinds[k] {
			continue
		}
		if _, _, err := l.client.Issues.AddLabelsToIssue(ctx, l.owner, l.repo, l.prNum, []string{"do-not-merge/kind-invalid"}); err != nil {
			return fmt.Errorf("failed to add do-not-merge label: %w", err)
		}
		return fmt.Errorf("invalid /kind %q detected, labeling do-not-merge", k)
	}

	// fetch current labels
	current, _, err := l.client.Issues.ListLabelsByIssue(ctx, l.owner, l.repo, l.prNum, nil)
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
		_, _, err := l.client.Issues.AddLabelsToIssue(ctx, l.owner, l.repo, l.prNum, []string{kindLabel})
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
		_, err := l.client.Issues.RemoveLabelForIssue(ctx, l.owner, l.repo, l.prNum, label)
		if err != nil {
			return fmt.Errorf("failed to remove label %q: %w", label, err)
		}
	}

	// Process release notes
	match := releaseNoteRE.FindStringSubmatch(sanitizedBody)
	if len(match) < 2 || strings.TrimSpace(match[1]) == "" {
		// Missing or empty => invalid
		l.client.Issues.AddLabelsToIssue(ctx, l.owner, l.repo, l.prNum, []string{"do-not-merge/release-note-invalid"})
		return fmt.Errorf("missing or empty ```release-note``` block; please add your line or 'NONE'")
	}

	// Handle the special case "NONE" scenario
	entry := strings.TrimSpace(match[1])
	if strings.EqualFold(entry, "NONE") {
		l.client.Issues.AddLabelsToIssue(ctx, l.owner, l.repo, l.prNum, []string{"release-note-none"})
		l.client.Issues.RemoveLabelForIssue(ctx, l.owner, l.repo, l.prNum, "do-not-merge/release-note-invalid")
		return nil
	}

	// Valid entry
	l.client.Issues.AddLabelsToIssue(ctx, l.owner, l.repo, l.prNum, []string{"release-note"})
	l.client.Issues.RemoveLabelForIssue(ctx, l.owner, l.repo, l.prNum, "do-not-merge/release-note-invalid")

	return nil
}
