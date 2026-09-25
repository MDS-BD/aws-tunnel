package cache

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/marcoreni/aws-opensearch-proxy/internal/cloud"
)

type item struct{ ID string }

func itemKey(i item) string   { return i.ID }
func itemLabel(i item) string { return i.ID }

func newStore(t *testing.T) *Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	s, err := Open("123456789012", "eu-west-1")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	return s
}

func TestGet(t *testing.T) {
	calls := 0
	fetch := func(context.Context) ([]item, error) {
		calls++
		return []item{{ID: "a"}}, nil
	}
	e := &Entry[item]{}

	if _, fetched, _ := Get(t.Context(), e, time.Hour, false, fetch); !fetched || calls != 1 {
		t.Errorf("empty cache must fetch: fetched=%v calls=%d", fetched, calls)
	}
	if _, fetched, _ := Get(t.Context(), e, time.Hour, false, fetch); fetched || calls != 1 {
		t.Errorf("fresh cache must not fetch: fetched=%v calls=%d", fetched, calls)
	}
	if _, fetched, _ := Get(t.Context(), e, time.Hour, true, fetch); !fetched || calls != 2 {
		t.Errorf("force must fetch: fetched=%v calls=%d", fetched, calls)
	}

	e.FetchedAt = time.Now().Add(-2 * time.Hour)
	if _, fetched, _ := Get(t.Context(), e, time.Hour, false, fetch); !fetched || calls != 3 {
		t.Errorf("stale cache must fetch: fetched=%v calls=%d", fetched, calls)
	}
}

func TestGetKeepsCacheOnError(t *testing.T) {
	e := &Entry[item]{Items: []item{{ID: "old"}}}
	_, _, err := Get(t.Context(), e, time.Hour, true, func(context.Context) ([]item, error) {
		return nil, errors.New("boom")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !reflect.DeepEqual(e.Items, []item{{ID: "old"}}) {
		t.Errorf("cache overwritten on error: %v", e.Items)
	}
}

func TestRank(t *testing.T) {
	s := newStore(t)
	s.Touch("c")
	s.Touch("c")
	s.Touch("b")

	items := []item{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	Rank(s, items, itemKey, itemLabel)

	expected := []item{{ID: "c"}, {ID: "b"}, {ID: "a"}, {ID: "d"}}
	if !reflect.DeepEqual(items, expected) {
		t.Errorf("rank mismatch:\nGot: %v\nExpected: %v", items, expected)
	}
}

func TestRankDecay(t *testing.T) {
	s := newStore(t)
	// Used a lot, long ago...
	s.data.Usage["old"] = Usage{Count: 10, LastUsed: time.Now().Add(-10 * usageHalfLife), Score: 10}
	// ...loses to used a little, today.
	s.Touch("new")

	items := []item{{ID: "old"}, {ID: "new"}}
	Rank(s, items, itemKey, itemLabel)
	if items[0].ID != "new" {
		t.Errorf("recent use must outrank decayed use, got %v", items)
	}
}

func TestSaveOpen(t *testing.T) {
	s := newStore(t)
	s.Domains().Items = nil
	s.Instances().FetchedAt = time.Now().Truncate(time.Second)
	s.Touch("x")
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	reopened, err := Open("123456789012", "eu-west-1")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if reopened.data.Usage["x"].Count != 1 {
		t.Errorf("usage not persisted: %v", reopened.data.Usage)
	}
	if !reopened.Instances().FetchedAt.Equal(s.Instances().FetchedAt) {
		t.Errorf("fetchedAt mismatch: %v != %v", reopened.Instances().FetchedAt, s.Instances().FetchedAt)
	}

	other, err := Open("210987654321", "eu-west-1")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if len(other.data.Usage) != 0 {
		t.Errorf("accounts must not share a cache: %v", other.data.Usage)
	}
}

func TestOpenResetsOldSchema(t *testing.T) {
	s := newStore(t)
	s.Instances().Items = []cloud.Instance{{ID: "i-1"}}
	s.Instances().FetchedAt = time.Now()
	s.Touch("x")
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	raw, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	old := strings.Replace(string(raw), `"version": 2`, `"version": 1`, 1)
	if err := os.WriteFile(s.Path(), []byte(old), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	reopened, err := Open("123456789012", "eu-west-1")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if !reopened.Instances().FetchedAt.IsZero() || len(reopened.Instances().Items) != 0 {
		t.Errorf("old listing must be dropped: %+v", reopened.Instances())
	}
	if reopened.data.Usage["x"].Count != 1 {
		t.Errorf("usage must survive a schema change: %v", reopened.data.Usage)
	}
}
