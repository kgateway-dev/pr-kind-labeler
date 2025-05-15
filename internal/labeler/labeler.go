package labeler

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
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
	// supportedKinds is a map of supported kind labels.
	supportedKinds = map[string]bool{
		"design":          true,
		"deprecation":     true,
		"feature":         true,
		"fix":             true,
		"breaking_change": true,
		"documentation":   true,
		"cleanup":         true,
		"flake":           true,
	}
)

// labeler handles PR labeling operations.
type labeler struct {
	client         *github.Client
	owner          string
	repo           string
	prNum          int
	labelsToAdd    map[string]bool
	labelsToRemove map[string]bool
	currentMap     map[string]bool
}

// New creates a new Labeler instance.
func New(client *github.Client, owner, repo string, prNum int) *labeler {
	return &labeler{
		client:         client,
		owner:          owner,
		repo:           repo,
		prNum:          prNum,
		labelsToAdd:    map[string]bool{},
		labelsToRemove: map[string]bool{},
		currentMap:     map[string]bool{},
	}
}

// ProcessPR processes the PR body and updates labels accordingly.
func (l *labeler) ProcessPR(ctx context.Context, body string) error {
	// fetch current labels
	if err := l.fetchLabels(ctx); err != nil {
		return err
	}
	// strip HTML comments to make the body easier to parse.
	sanitizedBody := commentRE.ReplaceAllString(body, "")

	var errs []error
	if err := l.processKindLabels(sanitizedBody); err != nil {
		errs = append(errs, err)
	}
	if err := l.processReleaseNotes(sanitizedBody); err != nil {
		errs = append(errs, err)
	}
	if err := l.syncLabels(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// fetchLabels fetches the current labels for the PR
func (l *labeler) fetchLabels(ctx context.Context) error {
	current, _, err := l.client.Issues.ListLabelsByIssue(ctx, l.owner, l.repo, l.prNum, nil)
	if err != nil {
		return fmt.Errorf("failed to list labels: %w", err)
	}
	currentMap := map[string]bool{}
	for _, L := range current {
		currentMap[L.GetName()] = true
	}
	l.currentMap = currentMap
	return nil
}

// processKindLabels handles the extraction and validation of kind labels
func (l *labeler) processKindLabels(body string) error {
	kinds := l.extractKinds(body)
	if err := l.verifyKinds(kinds); err != nil {
		return err
	}
	return l.syncKindLabels(kinds)
}

// extractKinds extracts all /kind commands from the PR body
func (l *labeler) extractKinds(body string) map[string]bool {
	kindRE := regexp.MustCompile(`(?m)^/kind\s+([\w/-]+)`)
	kinds := map[string]bool{}
	for _, match := range kindRE.FindAllStringSubmatch(body, -1) {
		kinds[match[1]] = true
	}
	return kinds
}

// verifyKinds checks if all extracted kinds are supported
func (l *labeler) verifyKinds(kinds map[string]bool) error {
	if len(kinds) == 0 {
		l.labelsToAdd["do-not-merge/kind-invalid"] = true
		return fmt.Errorf("no /kind labels found, labeling do-not-merge/kind-invalid")
	}
	for k := range kinds {
		if supportedKinds[k] {
			continue
		}
		l.labelsToAdd["do-not-merge/kind-invalid"] = true
		return fmt.Errorf("invalid /kind %q detected, labeling do-not-merge/kind-invalid", k)
	}
	return nil
}

// syncKindLabels synchronizes the PR labels with the extracted kinds
func (l *labeler) syncKindLabels(kinds map[string]bool) error {
	// add missing labels
	for k := range kinds {
		kindLabel := "kind/" + k
		if l.currentMap[kindLabel] {
			continue
		}
		l.labelsToAdd[kindLabel] = true
	}

	// remove stale labels
	for label := range l.currentMap {
		if !strings.HasPrefix(label, "kind/") {
			continue
		}
		kindType := strings.TrimPrefix(label, "kind/")
		if kinds[kindType] {
			continue
		}
		l.labelsToRemove[label] = true
	}

	return nil
}

// processReleaseNotes handles the release note validation and labeling
func (l *labeler) processReleaseNotes(body string) error {
	match := releaseNoteRE.FindStringSubmatch(body)
	if len(match) < 2 || strings.TrimSpace(match[1]) == "" {
		l.labelsToAdd["do-not-merge/release-note-invalid"] = true
		return fmt.Errorf("missing or empty ```release-note``` block; please add your line or 'NONE'")
	}

	// trim the release note entry and check if it's the special "NONE" entry.
	entry := strings.TrimSpace(match[1])
	if strings.EqualFold(entry, "NONE") {
		l.labelsToAdd["release-note-none"] = true
		if l.currentMap["do-not-merge/release-note-invalid"] {
			l.labelsToRemove["do-not-merge/release-note-invalid"] = true
		}
		return nil
	}

	// Valid entry, add the release-note label and remove the invalid label if it exists.
	l.labelsToAdd["release-note"] = true
	if l.currentMap["do-not-merge/release-note-invalid"] {
		l.labelsToRemove["do-not-merge/release-note-invalid"] = true
	}
	return nil
}

func (l *labeler) syncLabels(ctx context.Context) error {
	var errs []error
	labelsToAdd := make([]string, 0, len(l.labelsToAdd))
	for k := range l.labelsToAdd {
		labelsToAdd = append(labelsToAdd, k)
	}
	sort.Strings(labelsToAdd)

	_, _, err := l.client.Issues.AddLabelsToIssue(ctx, l.owner, l.repo, l.prNum, labelsToAdd)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to add labels %q: %w", labelsToAdd, err))
	}

	labelsToRemove := make([]string, 0, len(l.labelsToRemove))
	for k := range l.labelsToRemove {
		labelsToRemove = append(labelsToRemove, k)
	}
	sort.Strings(labelsToRemove)

	for _, label := range labelsToRemove {
		_, err = l.client.Issues.RemoveLabelForIssue(ctx, l.owner, l.repo, l.prNum, label)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to remove label %q: %w", label, err))
		}
	}

	return errors.Join(errs...)
}
