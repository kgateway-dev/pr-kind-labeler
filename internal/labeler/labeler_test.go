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
	l := New(c, "foo", "bar", 42)
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
	l := New(c, "foo", "bar", 42)
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
	l := New(github.NewClient(httpClient), "foo", "bar", 45)
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
	l := New(github.NewClient(httpClient), "foo", "bar", 43)
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
	l := New(github.NewClient(httpClient), "foo", "bar", 44)
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
	l := New(github.NewClient(httpClient), "foo", "bar", 46)
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

	l := New(github.NewClient(httpClient), "foo", "bar", 47)
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

	l := New(github.NewClient(httpClient), "foo", "bar", 47)
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

			l := New(github.NewClient(httpClient), "owner", "repo", tc.prNum)
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

	l := New(github.NewClient(httpClient), "owner", "repo", prNum)
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
