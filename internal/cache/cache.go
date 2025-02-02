// Cache for storing commits we've already diff-ed and parsed.
package cache

import (
	"fmt"
	"iter"
	"os"
	"slices"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/utils/iterutils"
)

func IsCachingEnabled() bool {
	if len(os.Getenv("GIT_WHO_DISABLE_CACHE")) > 0 {
		return false
	}

	return true
}

type Result struct {
	Revs    []string                     // All commit hashes in the sequence
	Commits iter.Seq2[git.Commit, error] // The sequence of commits
}

// If we use the zero-value for Result, the iterator will be nil. We instead
// want an interator over a zero-length sequence.
func EmptyResult() Result {
	return Result{
		Commits: iterutils.WithoutErrors(slices.Values([]git.Commit{})),
	}
}

func (r Result) AnyHits() bool {
	return len(r.Revs) > 0
}

type Backend interface {
	Name() string
	Get(revs []string) (Result, error)
	Add(commits []git.Commit) error
	Clear() error
}

type Cache struct {
	backend Backend
}

func NewCache(backend Backend) Cache {
	return Cache{
		backend: backend,
	}
}

func (c *Cache) Name() string {
	return c.backend.Name()
}

func (c *Cache) Get(revs []string) (_ Result, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to retrieve from cache: %w", err)
		}
	}()

	start := time.Now()

	result, err := c.backend.Get(revs)
	if err != nil {
		return result, err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug(
		"cache get",
		"duration_ms",
		elapsed.Milliseconds(),
		"hit",
		result.AnyHits(),
	)

	// Make sure iterator is not nil
	if result.Commits == nil {
		panic("Cache backend returned nil commits iterator; this isn't kosher!")
	}

	return result, nil
}

func (c *Cache) Add(commits []git.Commit) error {
	start := time.Now()

	err := c.backend.Add(commits)
	if err != nil {
		return err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug(
		"cache add",
		"duration_ms",
		elapsed.Milliseconds(),
	)

	return nil
}

func (c *Cache) Clear() error {
	err := c.backend.Clear()
	if err != nil {
		return err
	}

	logger().Debug("cache clear")
	return nil
}
