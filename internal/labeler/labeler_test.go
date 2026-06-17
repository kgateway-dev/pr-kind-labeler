package labeler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-github/v68/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"

	"github.com/kgateway-dev/pr-kind-labeler/pkg/kinds"
	"github.com/kgateway-dev/pr-kind-labeler/pkg/labels"
)

func TestProcessPR_NoKindSupplied(t *testing.T) {
	expectedLabelsToAdd := []string{labels.InvalidKindLabel, labels.ReleaseNoteLabel}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		// Add a handler for delete, even if we expect no removals, to capture any unexpected ones
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 42, false)
	err := l.ProcessPR(context.Background(), "```release-note\nOK\n```", true)
	if err == nil || !strings.Contains(err.Error(), "no /kind") {
		t.Fatalf("expected an error when no kind is supplied, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_InvalidKind(t *testing.T) {
	expectedLabelsToAdd := []string{labels.InvalidKindLabel, labels.ReleaseNoteLabel}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)
	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 42, false)
	err := l.ProcessPR(context.Background(), "/kind banana\n```release-note\nOK\n```", true)
	if err == nil || !strings.Contains(err.Error(), "invalid /kind") {
		t.Fatalf("expected kind-invalid error, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_ValidKind_InvalidReleaseNote(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.InvalidReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			// No initial labels on the PR for this test case
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)
	l := New(github.NewClient(httpClient), "foo", "bar", 45, false)
	err := l.ProcessPR(context.Background(), "/kind fix\n```release-note\n\n```", true)
	if err == nil || !strings.Contains(err.Error(), "missing or empty") {
		t.Fatalf("expected missing release-note error, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_ValidKindAndReleaseNote(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Feature),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				// Respond with the labels that were "added" as github.Label pointers
				responseGithubLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseGithubLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseGithubLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)
	l := New(github.NewClient(httpClient), "foo", "bar", 43, false)
	err := l.ProcessPR(context.Background(), "/kind feature\n```release-note\nNew feature implemented\n```", true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_MultipleKinds(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Feature),
		fmt.Sprintf("kind/%s", kinds.Cleanup),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseGithubLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseGithubLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseGithubLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)
	l := New(github.NewClient(httpClient), "foo", "bar", 44, false)
	err := l.ProcessPR(context.Background(), "/kind feature\n/kind cleanup\n```release-note\nCleanup and feature\n```", true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_ReleaseNoteNone(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Cleanup),
		labels.ReleaseNoteNoneLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseGithubLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseGithubLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseGithubLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)
	l := New(github.NewClient(httpClient), "foo", "bar", 46, false)
	err := l.ProcessPR(context.Background(), "/kind cleanup\n```release-note\nNONE\n```", true)
	if err != nil {
		t.Fatalf("expected no error on NONE, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestValidateReleaseNoteQuality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		entry     string
		wantError string
	}{
		{
			name:  "valid plain release note",
			entry: "Fixed route status updates when backend services are recreated.",
		},
		{
			name:      "fix prefix rejected",
			entry:     "fix: update route status.",
			wantError: "conventional commit prefix",
		},
		{
			name:      "scoped breaking conventional prefix rejected",
			entry:     "feat(helm)!: add listener policy support.",
			wantError: "conventional commit prefix",
		},
		{
			name:      "breaking change prefix rejected",
			entry:     "BREAKING CHANGE: Route policy defaults now require explicit backend refs.",
			wantError: "BREAKING",
		},
		{
			name:      "emoji rejected",
			entry:     "Added listener policy support 🚀",
			wantError: "ASCII",
		},
		{
			name:      "bullet list rejected",
			entry:     "- Added listener policy support.",
			wantError: "markdown bullets",
		},
		{
			name:      "heading rejected",
			entry:     "## Added listener policy support.",
			wantError: "markdown headings",
		},
		{
			name:      "fenced code block rejected",
			entry:     "```go\nfmt.Println(\"listener policy\")\n```",
			wantError: "fenced code blocks",
		},
		{
			name:      "blank line rejected",
			entry:     "Added listener policy support.\n\nUpdated Helm values.",
			wantError: "blank lines",
		},
		{
			name:      "this PR rejected",
			entry:     "This PR adds listener policy support.",
			wantError: "this PR",
		},
		{
			name:      "max length rejected",
			entry:     strings.Repeat("a", maxReleaseNoteLength+1),
			wantError: "characters or fewer",
		},
		{
			name:  "NONE is handled before quality validation",
			entry: "NONE",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateReleaseNote(tc.entry)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
			}
			if !strings.Contains(err.Error(), "copied verbatim into public changelogs") {
				t.Fatalf("expected public changelog guidance, got %v", err)
			}
		})
	}
}

func TestInvalidReleaseNoteQualityLabelsPR(t *testing.T) {
	expectedLabelsToAdd := []string{
		labels.InvalidReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{
		labels.ReleaseNoteLabel,
		labels.ReleaseNoteNoneLabel,
	}
	sort.Strings(expectedLabelsToRemove)

	actualLabelsAdded, actualLabelsRemoved, err := processPRForTest(t,
		[]*github.Label{
			{Name: github.Ptr(fmt.Sprintf("kind/%s", kinds.Fix))},
			{Name: github.Ptr(labels.ReleaseNoteLabel)},
			{Name: github.Ptr(labels.ReleaseNoteNoneLabel)},
		},
		"/kind fix\n```release-note\nfix: repaired route status updates.\n```",
		true,
	)
	if err == nil || !strings.Contains(err.Error(), "copied verbatim into public changelogs") {
		t.Fatalf("expected release-note quality error, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestInvalidChangelogKindCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantAdd   []string
		wantError string
	}{
		{
			name:      "multiple changelog kinds rejected",
			body:      "/kind feature\n/kind fix\n```release-note\nImproved route status updates.\n```",
			wantAdd:   []string{labels.InvalidKindLabel, labels.ReleaseNoteLabel},
			wantError: "multiple changelog /kind labels",
		},
		{
			name:      "breaking change plus fix rejected",
			body:      "/kind breaking_change\n/kind fix\n```release-note\nChanged route policy defaults.\n```",
			wantAdd:   []string{labels.InvalidKindLabel, labels.ReleaseNoteLabel},
			wantError: "multiple changelog /kind labels",
		},
		{
			name:    "cleanup plus flake with NONE accepted",
			body:    "/kind cleanup\n/kind flake\n```release-note\nNONE\n```",
			wantAdd: []string{fmt.Sprintf("kind/%s", kinds.Cleanup), fmt.Sprintf("kind/%s", kinds.Flake), labels.ReleaseNoteNoneLabel},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actualLabelsAdded, _, err := processPRForTest(t, []*github.Label{}, tc.body, false, true)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
			}
			sort.Strings(tc.wantAdd)
			if !reflect.DeepEqual(actualLabelsAdded, tc.wantAdd) {
				t.Fatalf("Expected labels to be added %v, got %v", tc.wantAdd, actualLabelsAdded)
			}
		})
	}
}

func TestStrictChangelogValidationDefaultsOff(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Feature),
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)

	actualLabelsAdded, _, err := processPRForTest(t,
		[]*github.Label{},
		"/kind feature\n/kind fix\n```release-note\nfix: repaired route status updates.\n```",
	)
	if err != nil {
		t.Fatalf("expected no error when strict changelog validation is not enabled, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
}

func TestReleaseNoteQualityFlagDoesNotEnforceKindExclusivity(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Feature),
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)

	actualLabelsAdded, _, err := processPRForTest(t,
		[]*github.Label{},
		"/kind feature\n/kind fix\n```release-note\nImproved route status updates.\n```",
		true,
		false,
	)
	if err != nil {
		t.Fatalf("expected no error when only release note quality validation is enabled, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
}

func TestKindExclusivityFlagDoesNotEnforceReleaseNoteQuality(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)

	actualLabelsAdded, _, err := processPRForTest(t,
		[]*github.Label{},
		"/kind fix\n```release-note\nfix: repaired route status updates.\n```",
		false,
		true,
	)
	if err != nil {
		t.Fatalf("expected no error when only changelog kind exclusivity is enabled, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
}

func TestProcessPR_EditedToInvalid(t *testing.T) {
	expectedLabelsToAdd := []string{
		labels.InvalidReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)

	expectedLabelsToRemove := []string{"release-note"}
	sort.Strings(expectedLabelsToRemove)

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{
				{Name: github.Ptr(fmt.Sprintf("kind/%s", kinds.Fix))},
				{Name: github.Ptr(labels.ReleaseNoteLabel)},
			},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pathPrefix := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/", "foo", "bar", 47)
				labelNameSegment := strings.TrimPrefix(r.URL.Path, pathPrefix)
				decodedLabelName, err := url.PathUnescape(labelNameSegment)
				if err != nil {
					t.Fatalf("Failed to unescape label name segment '%s': %v", labelNameSegment, err)
				}
				actualLabelsRemoved = append(actualLabelsRemoved, decodedLabelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	l := New(github.NewClient(httpClient), "foo", "bar", 47, false)
	err := l.ProcessPR(context.Background(), "/kind fix\nNo release-note here", true)
	if err == nil || !strings.Contains(err.Error(), "missing or empty ```release-note``` block") {
		t.Fatalf("ProcessPR error expected to contain 'missing or empty ```release-note``` block', got: %v", err.Error())
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_EditedToValid(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{
		labels.InvalidReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToRemove)

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{{Name: github.Ptr(labels.InvalidReleaseNoteLabel)}},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseGithubLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseGithubLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseGithubLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pathPrefix := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/", "foo", "bar", 47)
				labelNameSegment := strings.TrimPrefix(r.URL.Path, pathPrefix)
				decodedLabelName, err := url.PathUnescape(labelNameSegment)
				if err != nil {
					t.Fatalf("Failed to unescape label name segment '%s': %v", labelNameSegment, err)
				}
				actualLabelsRemoved = append(actualLabelsRemoved, decodedLabelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	l := New(github.NewClient(httpClient), "foo", "bar", 47, false)
	err := l.ProcessPR(context.Background(), "/kind fix\\n```release-note\\nFixed it\\n```", true)
	if err != nil {
		t.Fatalf("expected no error from ProcessPR, got %v", err)
	}

	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_LabelMigrationTableDriven(t *testing.T) {
	tt := []struct {
		name                   string
		prNum                  int
		initialLabels          []*github.Label
		prBody                 string
		expectedLabelsToAdd    []string
		expectedLabelsToRemove []string
	}{
		{
			name:  "Deprecated_Bug_Fix_To_Fix",
			prNum: 101,
			initialLabels: []*github.Label{
				{Name: github.Ptr("kind/bug_fix")},
				{Name: github.Ptr("release-note-needed")},
			},
			prBody: "/kind fix\\n```release-note\\nValid note\\n```",
			expectedLabelsToAdd: []string{
				fmt.Sprintf("kind/%s", kinds.Fix),
				labels.ReleaseNoteLabel,
			},
			expectedLabelsToRemove: []string{
				fmt.Sprintf("kind/%s", kinds.DeprecatedBugFix),
				labels.DeprecatedReleaseNoteLabel,
			},
		},
		{
			name:  "Deprecated_Feature_To_New_Feature",
			prNum: 106,
			initialLabels: []*github.Label{
				{Name: github.Ptr(fmt.Sprintf("kind/%s", kinds.DeprecatedNewFeature))},
				{Name: github.Ptr(labels.DeprecatedReleaseNoteLabel)},
			},
			prBody: "/kind new_feature\\n```release-note\\nValid note\\n```",
			expectedLabelsToAdd: []string{
				fmt.Sprintf("kind/%s", kinds.Feature),
				labels.ReleaseNoteLabel,
			},
			expectedLabelsToRemove: []string{
				fmt.Sprintf("kind/%s", kinds.DeprecatedNewFeature),
				labels.DeprecatedReleaseNoteLabel,
			},
		},
		{
			name:          "Install_Kind_Label",
			prNum:         107,
			initialLabels: []*github.Label{},
			prBody:        "/kind install\\n```release-note\\nUpdated Helm chart\\n```",
			expectedLabelsToAdd: []string{
				fmt.Sprintf("kind/%s", kinds.Install),
				labels.ReleaseNoteLabel,
			},
			expectedLabelsToRemove: []string{},
		},
		{
			name:          "Bump_Kind_Label",
			prNum:         108,
			initialLabels: []*github.Label{},
			prBody:        "/kind bump\\n```release-note\\nUpdated dependencies\\n```",
			expectedLabelsToAdd: []string{
				fmt.Sprintf("kind/%s", kinds.Bump),
				labels.ReleaseNoteLabel,
			},
			expectedLabelsToRemove: []string{},
		},
		{
			name:          "Test_Kind_Label",
			prNum:         109,
			initialLabels: []*github.Label{},
			prBody:        "/kind test\\n```release-note\\nAdded unit tests\\n```",
			expectedLabelsToAdd: []string{
				fmt.Sprintf("kind/%s", kinds.Test),
				labels.ReleaseNoteLabel,
			},
			expectedLabelsToRemove: []string{},
		},
	}

	for _, tc := range tt {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var actualLabelsAdded []string = make([]string, 0)
			var actualLabelsRemoved []string = make([]string, 0)

			httpClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
					tc.initialLabels,
				),
				mock.WithRequestMatchHandler(
					mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
							t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
						}
						sort.Strings(actualLabelsAdded)
						responseLabels := make([]*github.Label, len(actualLabelsAdded))
						for i, name := range actualLabelsAdded {
							responseLabels[i] = &github.Label{Name: github.Ptr(name)}
						}
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(responseLabels)
					}),
				),
				mock.WithRequestMatchHandler(
					mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						pathPrefix := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/", "owner", "repo", tc.prNum)
						labelNameSegment := strings.TrimPrefix(r.URL.Path, pathPrefix)
						decodedLabelName, err := url.PathUnescape(labelNameSegment)
						if err != nil {
							t.Fatalf("Failed to unescape label name segment '%s': %v", labelNameSegment, err)
						}
						actualLabelsRemoved = append(actualLabelsRemoved, decodedLabelName)
						sort.Strings(actualLabelsRemoved)
						w.WriteHeader(http.StatusNoContent)
					}),
				),
			)

			l := New(github.NewClient(httpClient), "owner", "repo", tc.prNum, false)
			err := l.ProcessPR(context.Background(), tc.prBody, true)
			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			sort.Strings(tc.expectedLabelsToAdd)
			if !reflect.DeepEqual(actualLabelsAdded, tc.expectedLabelsToAdd) {
				t.Errorf("Expected labels to add %v, got %v", tc.expectedLabelsToAdd, actualLabelsAdded)
			}

			sort.Strings(tc.expectedLabelsToRemove)
			if !reflect.DeepEqual(actualLabelsRemoved, tc.expectedLabelsToRemove) {
				t.Errorf("Expected labels to remove %v, got %v", tc.expectedLabelsToRemove, actualLabelsRemoved)
			}
		})
	}
}

func TestProcessPR_RemovesKindInvalid_WhenValidKindProvided(t *testing.T) {
	t.Parallel()

	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Feature),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{
		labels.InvalidKindLabel,
		labels.ReleaseNoteNoneLabel,
	}
	sort.Strings(expectedLabelsToRemove)

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)
	prNum := 201

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{
				{Name: github.Ptr(labels.InvalidKindLabel)},
				{Name: github.Ptr(labels.ReleaseNoteNoneLabel)},
			},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pathPrefix := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/", "owner", "repo", prNum)
				labelNameSegment := strings.TrimPrefix(r.URL.Path, pathPrefix)
				decodedLabelName, err := url.PathUnescape(labelNameSegment)
				if err != nil {
					t.Fatalf("Failed to unescape label name segment '%s': %v", labelNameSegment, err)
				}
				actualLabelsRemoved = append(actualLabelsRemoved, decodedLabelName)
				sort.Strings(actualLabelsRemoved)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	l := New(github.NewClient(httpClient), "owner", "repo", prNum, false)
	err := l.ProcessPR(context.Background(), "/kind feature\\n```release-note\\nNONE\\n```", true)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Errorf("Expected labels to add %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Errorf("Expected labels to remove %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_MissingDescription(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
		labels.InvalidDescriptionLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 50, true)
	err := l.ProcessPR(context.Background(), "/kind fix\n```release-note\nFixed bug\n```", true)
	if err == nil || !strings.Contains(err.Error(), "missing # Description section") {
		t.Fatalf("expected missing Description error, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_EmptyDescription(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
		labels.InvalidDescriptionLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 51, true)
	prBody := "# Description\n\n# Change Type\n/kind fix\n\n```release-note\nFixed bug\n```"
	err := l.ProcessPR(context.Background(), prBody, true)
	if err == nil || !strings.Contains(err.Error(), "empty # Description section") {
		t.Fatalf("expected empty Description error, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_ValidDescription(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 52, true)
	prBody := "# Description\n\nThis PR fixes a critical bug in the authentication flow.\n\n# Change Type\n/kind fix\n\n```release-note\nFixed authentication bug\n```"
	err := l.ProcessPR(context.Background(), prBody, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_DescriptionValidationDisabled(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 53, false)
	// No description section, but validation is disabled
	err := l.ProcessPR(context.Background(), "/kind fix\n```release-note\nFixed bug\n```", true)
	if err != nil {
		t.Fatalf("expected no error when description validation disabled, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_RemovesInvalidDescription_WhenValidDescriptionProvided(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{
		labels.InvalidDescriptionLabel,
	}
	sort.Strings(expectedLabelsToRemove)

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{
				{Name: github.Ptr(labels.InvalidDescriptionLabel)},
			},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pathPrefix := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/", "foo", "bar", 54)
				labelNameSegment := strings.TrimPrefix(r.URL.Path, pathPrefix)
				decodedLabelName, err := url.PathUnescape(labelNameSegment)
				if err != nil {
					t.Fatalf("Failed to unescape label name segment '%s': %v", labelNameSegment, err)
				}
				actualLabelsRemoved = append(actualLabelsRemoved, decodedLabelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 54, true)
	prBody := "# Description\n\nThis PR fixes an important bug.\n\n# Change Type\n/kind fix\n\n```release-note\nFixed important bug\n```"
	err := l.ProcessPR(context.Background(), prBody, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func TestProcessPR_ValidDescriptionWithSubheadings(t *testing.T) {
	expectedLabelsToAdd := []string{
		fmt.Sprintf("kind/%s", kinds.Fix),
		labels.ReleaseNoteLabel,
	}
	sort.Strings(expectedLabelsToAdd)
	expectedLabelsToRemove := []string{}

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				labelName := parts[len(parts)-1]
				actualLabelsRemoved = append(actualLabelsRemoved, labelName)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 55, true)
	prBody := "# Description\n\n## Motivation\n\nThis fixes a bug.\n\n## Implementation\n\nUsed a different approach.\n\n# Change Type\n/kind fix\n\n```release-note\nFixed bug\n```"
	err := l.ProcessPR(context.Background(), prBody, true)
	if err != nil {
		t.Fatalf("expected no error with subheadings in description, got %v", err)
	}
	if !reflect.DeepEqual(actualLabelsAdded, expectedLabelsToAdd) {
		t.Fatalf("Expected labels to be added %v, got %v", expectedLabelsToAdd, actualLabelsAdded)
	}
	sort.Strings(actualLabelsRemoved)
	if !reflect.DeepEqual(actualLabelsRemoved, expectedLabelsToRemove) {
		t.Fatalf("Expected labels to be removed %v, got %v", expectedLabelsToRemove, actualLabelsRemoved)
	}
}

func processPRForTest(t *testing.T, initialLabels []*github.Label, prBody string, validationFlags ...bool) ([]string, []string, error) {
	t.Helper()

	var actualLabelsAdded []string = make([]string, 0)
	var actualLabelsRemoved []string = make([]string, 0)
	const prNum = 900
	enforceReleaseNoteQuality := false
	if len(validationFlags) > 0 {
		enforceReleaseNoteQuality = validationFlags[0]
	}
	enforceChangelogKindExclusivity := false
	if len(validationFlags) > 1 {
		enforceChangelogKindExclusivity = validationFlags[1]
	}

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			initialLabels,
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&actualLabelsAdded); err != nil {
					t.Fatalf("AddLabels Handler: failed to decode body: %v", err)
				}
				sort.Strings(actualLabelsAdded)
				responseLabels := make([]*github.Label, len(actualLabelsAdded))
				for i, name := range actualLabelsAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pathPrefix := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/", "owner", "repo", prNum)
				labelNameSegment := strings.TrimPrefix(r.URL.Path, pathPrefix)
				decodedLabelName, err := url.PathUnescape(labelNameSegment)
				if err != nil {
					t.Fatalf("Failed to unescape label name segment '%s': %v", labelNameSegment, err)
				}
				actualLabelsRemoved = append(actualLabelsRemoved, decodedLabelName)
				sort.Strings(actualLabelsRemoved)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	l := New(github.NewClient(httpClient), "owner", "repo", prNum, false, enforceReleaseNoteQuality, enforceChangelogKindExclusivity)
	err := l.ProcessPR(context.Background(), prBody, true)
	return actualLabelsAdded, actualLabelsRemoved, err
}
