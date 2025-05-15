package labeler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-github/v68/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
)

func TestProcessPR_NoKindSupplied(t *testing.T) {
	// note: no need to mock the labels, as the labeler will exit early if no
	// kind is supplied and no labels are added.
	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
	)

	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 42)
	err := l.ProcessPR(context.Background(), "```release-note\nOK\n```")
	if err == nil || !strings.Contains(err.Error(), "no /kind") {
		t.Fatalf("expected an error when no kind is supplied, got %v", err)
	}
}

func TestProcessPR_InvalidKind(t *testing.T) {
	// note: no need to mock the labels, as the labeler will exit early if the
	// kind is invalid and no labels are added.
	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
	)
	c := github.NewClient(httpClient)
	l := New(c, "foo", "bar", 42)
	err := l.ProcessPR(context.Background(), "/kind banana\n```release-note\nOK\n```")
	if err == nil || !strings.Contains(err.Error(), "invalid /kind") {
		t.Fatalf("expected kind-invalid error, got %v", err)
	}
}

func TestProcessPR_ValidKind_InvalidReleaseNote(t *testing.T) {
	// note: no need to mock the labels, as the labeler will exit early if the
	// release note is invalid and no labels are added.
	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{{Name: github.Ptr("kind/fix")}},
		),
	)
	l := New(github.NewClient(httpClient), "foo", "bar", 45)
	err := l.ProcessPR(context.Background(), "/kind fix\n```release-note\n\n```")
	if err == nil || !strings.Contains(err.Error(), "missing or empty") {
		t.Fatalf("expected missing release-note error, got %v", err)
	}
}

func TestProcessPR_ValidKindAndReleaseNote(t *testing.T) {
	expectedLabelsToAdd := []*github.Label{
		{Name: github.Ptr("kind/feature")},
		{Name: github.Ptr("release-note")},
	}
	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var labelsSentForAddition []string
				if err := json.NewDecoder(r.Body).Decode(&labelsSentForAddition); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				expectedLabelNames := make([]string, len(expectedLabelsToAdd))
				for i, label := range expectedLabelsToAdd {
					expectedLabelNames[i] = *label.Name
				}
				if len(labelsSentForAddition) != len(expectedLabelNames) {
					t.Fatalf("expected %d labels to be sent for addition, got %d. Expected: %v, Got: %v", len(expectedLabelNames), len(labelsSentForAddition), expectedLabelNames, labelsSentForAddition)
				}
				for _, expectedName := range expectedLabelNames {
					if slices.Contains(labelsSentForAddition, expectedName) {
						continue
					}
					t.Fatalf("expected label %s to be sent for addition, but it was not. Sent: %v", expectedName, labelsSentForAddition)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(expectedLabelsToAdd)
			}),
		),
	)
	l := New(github.NewClient(httpClient), "foo", "bar", 43)
	err := l.ProcessPR(context.Background(), "/kind feature\n```release-note\nNew feature implemented\n```")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessPR_MultipleKinds(t *testing.T) {
	expectedLabelsToAdd := []*github.Label{
		{Name: github.Ptr("kind/feature")},
		{Name: github.Ptr("kind/cleanup")},
		{Name: github.Ptr("release-note")},
	}
	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var labelsSentForAddition []string
				if err := json.NewDecoder(r.Body).Decode(&labelsSentForAddition); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				expectedLabelNames := make([]string, len(expectedLabelsToAdd))
				for i, label := range expectedLabelsToAdd {
					expectedLabelNames[i] = *label.Name
				}
				if len(labelsSentForAddition) != len(expectedLabelNames) {
					t.Fatalf("expected %d labels to be sent for addition, got %d. Expected: %v, Got: %v", len(expectedLabelNames), len(labelsSentForAddition), expectedLabelNames, labelsSentForAddition)
				}
				for _, expectedName := range expectedLabelNames {
					if slices.Contains(labelsSentForAddition, expectedName) {
						continue
					}
					t.Fatalf("expected label %s to be sent for addition, but it was not. Sent: %v", expectedName, labelsSentForAddition)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(expectedLabelsToAdd)
			}),
		),
	)
	l := New(github.NewClient(httpClient), "foo", "bar", 44)
	err := l.ProcessPR(context.Background(), "/kind feature\n/kind cleanup\n```release-note\nCleanup and feature\n```")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessPR_ReleaseNoteNone(t *testing.T) {
	expectedLabelToAdd := []*github.Label{
		{Name: github.Ptr("release-note-none")},
		{Name: github.Ptr("kind/cleanup")},
	}
	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var labelsSentForAddition []string
				if err := json.NewDecoder(r.Body).Decode(&labelsSentForAddition); err != nil {
					t.Fatalf("failed to decode request body for add: %v", err)
				}
				for _, label := range expectedLabelToAdd {
					if slices.Contains(labelsSentForAddition, *label.Name) {
						continue
					}
					t.Fatalf("expected label %s to be sent for addition, but it was not. Sent: %v", *label.Name, labelsSentForAddition)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(expectedLabelToAdd)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Println("DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName", r.Body)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)
	l := New(github.NewClient(httpClient), "foo", "bar", 46)
	err := l.ProcessPR(context.Background(), "/kind cleanup\n```release-note\nNONE\n```")
	if err != nil {
		t.Fatalf("expected no error on NONE, got %v", err)
	}
}

func TestProcessPR_EditedToInvalid(t *testing.T) {
	// note: no need to mock the labels, as the labeler will exit early if the
	// release note is invalid and no labels are added
	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{
				{Name: github.Ptr("kind/fix")},
				{Name: github.Ptr("release-note")},
			},
		),
	)

	l := New(github.NewClient(httpClient), "foo", "bar", 47)
	err := l.ProcessPR(context.Background(), "/kind fix\nNo release-note here")
	if err == nil || !strings.Contains(err.Error(), "missing or empty ```release-note``` block") {
		t.Fatalf("ProcessPR error expected to contain 'missing or empty ```release-note``` block', got: %v", err.Error())
	}
}

func TestProcessPR_EditedToValid(t *testing.T) {
	expectedLabelsToBeAdded := []string{"kind/fix", "release-note"}
	sort.Strings(expectedLabelsToBeAdded)

	httpClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposIssuesLabelsByOwnerByRepoByIssueNumber,
			[]*github.Label{{Name: github.Ptr("do-not-merge/release-note-invalid")}},
		),
		mock.WithRequestMatchHandler(
			mock.PostReposIssuesLabelsByOwnerByRepoByIssueNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var labelsBeingAdded []string
				if err := json.NewDecoder(r.Body).Decode(&labelsBeingAdded); err != nil {
					t.Fatalf("AddLabels Handler (expecting %v): failed to decode body: %v", expectedLabelsToBeAdded, err)
				}
				if !reflect.DeepEqual(labelsBeingAdded, expectedLabelsToBeAdded) {
					t.Fatalf("AddLabels Handler: expected %v, got %v", expectedLabelsToBeAdded, labelsBeingAdded)
				}
				responseLabels := make([]*github.Label, len(labelsBeingAdded))
				for i, name := range labelsBeingAdded {
					responseLabels[i] = &github.Label{Name: github.Ptr(name)}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseLabels)
			}),
		),
		mock.WithRequestMatchHandler(
			mock.DeleteReposIssuesLabelsByOwnerByRepoByIssueNumberByName,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "do-not-merge/release-note-invalid") {
					t.Fatalf("expected label to be removed, got %v", r.URL.Path)
				}
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	l := New(github.NewClient(httpClient), "foo", "bar", 47)
	err := l.ProcessPR(context.Background(), "/kind fix\n```release-note\nFixed it\n```")
	if err != nil {
		t.Fatalf("expected no error from ProcessPR, got %v", err)
	}
}
