package cache

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/google/renameio/v2"

	"github.com/MDS-BD/aws-tunnel/internal/cloud"
)

// schemaVersion invalidates cached listings when their shape changes; usage history survives.
const schemaVersion = 2

// usageHalfLife controls how fast past picks fade: a pick made one half-life ago weighs half as much as one made now.
const usageHalfLife = 14 * 24 * time.Hour

// Entry is a cached AWS listing.
type Entry[T any] struct {
	FetchedAt time.Time `json:"fetchedAt"`
	Items     []T       `json:"items"`
}

// Usage is a frecency counter: an LFU score with exponential decay, so items used often *and* recently rank first.
type Usage struct {
	Count    int       `json:"count"`
	LastUsed time.Time `json:"lastUsed"`
	Score    float64   `json:"score"`
}

type data struct {
	Version   int                   `json:"version"`
	Domains   Entry[cloud.Domain]   `json:"domains"`
	Instances Entry[cloud.Instance] `json:"instances"`
	Usage     map[string]Usage      `json:"usage"`
}

// Store holds the cache of a single AWS account + region.
type Store struct {
	data data
	path string
}

func Open(accountID, region string) (*Store, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("user cache dir: %w", err)
	}

	s := &Store{
		data: data{Usage: map[string]Usage{}},
		path: filepath.Join(dir, "aws-tunnel", accountID+"-"+region+".json"),
	}

	raw, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read cache: %w", err)
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		// A corrupt cache is not fatal: start over, it gets rebuilt from AWS.
		return &Store{data: data{Usage: map[string]Usage{}}, path: s.path}, nil
	}
	if s.data.Usage == nil {
		s.data.Usage = map[string]Usage{}
	}
	if s.data.Version != schemaVersion {
		s.data = data{Usage: s.data.Usage}
	}
	return s, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Save() error {
	s.data.Version = schemaVersion
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if err := renameio.WriteFile(s.path, raw, 0o600); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	return nil
}

func (s *Store) Domains() *Entry[cloud.Domain]     { return &s.data.Domains }
func (s *Store) Instances() *Entry[cloud.Instance] { return &s.data.Instances }

// Get returns the cached items, fetching them again when forced or older than maxAge.
func Get[T any](ctx context.Context, e *Entry[T], maxAge time.Duration, force bool, fetch func(context.Context) ([]T, error)) ([]T, bool, error) {
	if !force && !e.FetchedAt.IsZero() && time.Since(e.FetchedAt) < maxAge {
		return e.Items, false, nil
	}
	items, err := fetch(ctx)
	if err != nil {
		return nil, false, err
	}
	e.FetchedAt = time.Now()
	e.Items = items
	return items, true, nil
}

// Touch records a use of key.
func (s *Store) Touch(key string) {
	now := time.Now()
	u := s.data.Usage[key]
	s.data.Usage[key] = Usage{Count: u.Count + 1, LastUsed: now, Score: u.decayed(now) + 1}
}

// Rank sorts items by descending usage score; unused items keep their order, sorted by label.
func Rank[T any](s *Store, items []T, key, label func(T) string) {
	now := time.Now()
	slices.SortStableFunc(items, func(a, b T) int {
		return cmp.Or(
			cmp.Compare(s.data.Usage[key(b)].decayed(now), s.data.Usage[key(a)].decayed(now)),
			cmp.Compare(label(a), label(b)),
		)
	})
}

func (u Usage) decayed(now time.Time) float64 {
	if u.Score == 0 {
		return 0
	}
	return u.Score * math.Exp2(-float64(now.Sub(u.LastUsed))/float64(usageHalfLife))
}
