package labeler

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/google/go-github/v68/github"
	"github.com/kgateway-dev/pr-kind-labeler/pkg/kinds"
	"github.com/kgateway-dev/pr-kind-labeler/pkg/labels"
)

var (
	// commentRE strips HTML comments so example code isn't parsed.
	commentRE = regexp.MustCompile(`(?s)<!--.*?-->`)
	// kindRE captures /kind labels, case-insensitive, matching start of line.
	kindRE = regexp.MustCompile(`(?im)^/kind\s+([a-z0-9_/-]+)`)
	// releaseNoteRE captures the first fenced code block with the word "release-note" in it.
	releaseNoteRE = regexp.MustCompile("(?s)```release-note\\s*(.*?)\\s*```")
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
func (l *labeler) ProcessPR(ctx context.Context, body string, syncLabels bool) error {
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
	if syncLabels {
		if err := l.syncLabels(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return joinErrs(errs...)
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
	parsedKinds := map[string]bool{}
	for _, match := range kindRE.FindAllStringSubmatch(body, -1) {
		kind := strings.ToLower(match[1])
		// temporary migration: if the kind is deprecated, use the new kind
		newKind, ok := kinds.DeprecatedKindMap[kind]
		if ok {
			parsedKinds[newKind] = true
			continue
		}
		parsedKinds[kind] = true
	}
	return parsedKinds
}

// verifyKinds checks if all extracted kinds are supported
func (l *labeler) verifyKinds(extractedKinds map[string]bool) error {
	if len(extractedKinds) == 0 {
		if !l.currentMap[labels.InvalidKindLabel] {
			l.labelsToAdd[labels.InvalidKindLabel] = true
		}
		return fmt.Errorf("no /kind labels found, labeling %q. supported kinds: %v", labels.InvalidKindLabel, slices.Collect(maps.Keys(kinds.SupportedKinds)))
	}
	for k := range extractedKinds {
		if kinds.SupportedKinds[k] {
			continue
		}
		if !l.currentMap[labels.InvalidKindLabel] {
			l.labelsToAdd[labels.InvalidKindLabel] = true
		}
		return fmt.Errorf("invalid /kind %q detected, labeling %q. supported kinds: %v", k, labels.InvalidKindLabel, slices.Collect(maps.Keys(kinds.SupportedKinds)))
	}
	if l.currentMap[labels.InvalidKindLabel] {
		l.labelsToRemove[labels.InvalidKindLabel] = true
	}
	return nil
}

// syncKindLabels synchronizes the PR labels with the extracted kinds
func (l *labeler) syncKindLabels(extractedKinds map[string]bool) error {
	// add missing labels
	for k := range extractedKinds {
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
		currentKindType := strings.TrimPrefix(label, "kind/")
		if newKindEquivalent, isDeprecated := kinds.DeprecatedKindMap[currentKindType]; isDeprecated {
			if extractedKinds[newKindEquivalent] {
				l.labelsToRemove[label] = true
				continue
			}
		}
		if !extractedKinds[currentKindType] {
			l.labelsToRemove[label] = true
		}
	}

	return nil
}

// processReleaseNotes handles the release note validation and labeling
func (l *labeler) processReleaseNotes(body string) error {
	// temporary migration: if the deprecated release-note-needed label exists, remove it
	// and let the logic below add the correct label.
	if l.currentMap[labels.DeprecatedReleaseNoteLabel] {
		l.labelsToRemove[labels.DeprecatedReleaseNoteLabel] = true
	}

	// validate the release note block is present
	match := releaseNoteRE.FindStringSubmatch(body)
	if len(match) < 2 {
		if !l.currentMap[labels.InvalidReleaseNoteLabel] {
			l.labelsToAdd[labels.InvalidReleaseNoteLabel] = true
		}
		if l.currentMap[labels.ReleaseNoteLabel] {
			l.labelsToRemove[labels.ReleaseNoteLabel] = true
		}
		if l.currentMap[labels.ReleaseNoteNoneLabel] {
			l.labelsToRemove[labels.ReleaseNoteNoneLabel] = true
		}
		return fmt.Errorf("missing or empty ```release-note``` block; please add your line. If no release notes, add:\n```release-note\nNONE\n```")
	}

	// process the release note block
	entry := strings.TrimSpace(match[1])
	switch {
	case entry == "":
		if !l.currentMap[labels.InvalidReleaseNoteLabel] {
			l.labelsToAdd[labels.InvalidReleaseNoteLabel] = true
		}
		if l.currentMap[labels.ReleaseNoteLabel] {
			l.labelsToRemove[labels.ReleaseNoteLabel] = true
		}
		if l.currentMap[labels.ReleaseNoteNoneLabel] {
			l.labelsToRemove[labels.ReleaseNoteNoneLabel] = true
		}
		return fmt.Errorf("missing or empty ```release-note``` block; please add your line or 'NONE'")
	case strings.EqualFold(entry, "NONE"):
		// handle special NONE case
		if !l.currentMap[labels.ReleaseNoteNoneLabel] {
			l.labelsToAdd[labels.ReleaseNoteNoneLabel] = true
		}
		if l.currentMap[labels.InvalidReleaseNoteLabel] {
			l.labelsToRemove[labels.InvalidReleaseNoteLabel] = true
		}
		if l.currentMap[labels.ReleaseNoteLabel] {
			l.labelsToRemove[labels.ReleaseNoteLabel] = true
		}
	default:
		// validate release note was found
		if !l.currentMap[labels.ReleaseNoteLabel] {
			l.labelsToAdd[labels.ReleaseNoteLabel] = true
		}
		if l.currentMap[labels.InvalidReleaseNoteLabel] {
			l.labelsToRemove[labels.InvalidReleaseNoteLabel] = true
		}
		if l.currentMap[labels.ReleaseNoteNoneLabel] {
			l.labelsToRemove[labels.ReleaseNoteNoneLabel] = true
		}
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

type joinError []error

// Error implements error.
func (j joinError) Error() string {
	if len(j) == 0 {
		return ""
	}
	if len(j) == 1 {
		return j[0].Error()
	}
	var sb strings.Builder
	for _, err := range j {
		sb.WriteString("\n")
		sb.WriteString("- " + err.Error())
	}
	return sb.String()
}

func joinErrs(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}
	return joinError(errs)
}
