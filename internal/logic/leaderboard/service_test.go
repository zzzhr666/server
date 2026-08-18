package leaderboard

import (
	"context"
	"errors"
	"testing"
)

func TestServiceListNormalizesValidQueries(t *testing.T) {
	tests := []struct {
		name  string
		input ListInput
		want  ListInput
	}{
		{
			name:  "solo defaults limit",
			input: ListInput{Type: TypeSoloClearTime, MapVersion: "wave-v1"},
			want:  ListInput{Type: TypeSoloClearTime, MapVersion: "wave-v1", Limit: 20},
		},
		{
			name:  "duo preserves explicit limit",
			input: ListInput{Type: TypeDuoClearTime, MapVersion: "wave-v1", Limit: 50},
			want:  ListInput{Type: TypeDuoClearTime, MapVersion: "wave-v1", Limit: 50},
		},
		{
			name:  "total kills clears map version",
			input: ListInput{Type: TypeTotalKills, MapVersion: "ignored", Limit: 100},
			want:  ListInput{Type: TypeTotalKills, Limit: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantResult := &Result{Type: tt.want.Type, MapVersion: tt.want.MapVersion}
			repo := &fakeRepository{result: wantResult}
			result, err := NewService(repo).List(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("List returned error: %v", err)
			}
			if repo.input != tt.want {
				t.Fatalf("repository input = %+v, want %+v", repo.input, tt.want)
			}
			if result != wantResult {
				t.Fatalf("result = %+v, want %+v", result, wantResult)
			}
		})
	}
}

func TestServiceListRejectsInvalidQueries(t *testing.T) {
	tests := []struct {
		name  string
		input ListInput
	}{
		{name: "unknown type", input: ListInput{Type: "unknown", Limit: 10}},
		{name: "solo missing map", input: ListInput{Type: TypeSoloClearTime, Limit: 10}},
		{name: "duo missing map", input: ListInput{Type: TypeDuoClearTime, Limit: 10}},
		{name: "negative limit", input: ListInput{Type: TypeTotalKills, Limit: -1}},
		{name: "limit too large", input: ListInput{Type: TypeTotalKills, Limit: 101}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			_, err := NewService(repo).List(context.Background(), tt.input)
			if !errors.Is(err, ErrInvalidQuery) {
				t.Fatalf("List error = %v, want %v", err, ErrInvalidQuery)
			}
			if repo.called {
				t.Fatal("repository called for invalid query")
			}
		})
	}
}

func TestServiceListReturnsContextAndRepositoryErrors(t *testing.T) {
	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()
	repo := &fakeRepository{}
	_, err := NewService(repo).List(canceledContext, ListInput{Type: TypeTotalKills, Limit: 10})
	if !errors.Is(err, context.Canceled) || repo.called {
		t.Fatalf("canceled List error = %v, repository called = %v", err, repo.called)
	}

	wantErr := errors.New("state unavailable")
	repo = &fakeRepository{err: wantErr}
	_, err = NewService(repo).List(context.Background(), ListInput{Type: TypeTotalKills, Limit: 10})
	if !errors.Is(err, wantErr) {
		t.Fatalf("repository List error = %v, want %v", err, wantErr)
	}
}

type fakeRepository struct {
	called bool
	input  ListInput
	result *Result
	err    error
}

func (f *fakeRepository) List(_ context.Context, input ListInput) (*Result, error) {
	f.called = true
	f.input = input
	return f.result, f.err
}

var _ Repository = (*fakeRepository)(nil)
