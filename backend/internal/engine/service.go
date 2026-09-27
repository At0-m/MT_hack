package engine

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"
	d "tramflow/internal/domain"
	"tramflow/internal/inference"
)

type Repository interface {
	Snapshot(context.Context, string) (d.Snapshot, d.Model, error)
	// Hours returns small profiles and calendar flags, never the full model features.
	Hours(context.Context, d.Snapshot, SelectionInput) ([]d.Hour, error)
	Features(context.Context, d.Snapshot, []d.Hour) ([]d.Hour, error)
}
type SelectionInput struct {
	Routes []string
	Window d.Window
}
type cacheEntry struct {
	key   string
	value float64
}
type Cache struct {
	mu     sync.Mutex
	limit  int
	lru    *list.List
	values map[string]*list.Element
}

func NewCache(bytes int) *Cache {
	return &Cache{limit: bytes / 512, lru: list.New(), values: map[string]*list.Element{}}
}
func (c *Cache) Get(k string) (float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.values[k]
	if !ok {
		return 0, false
	}
	c.lru.MoveToFront(e)
	return e.Value.(cacheEntry).value, true
}
func (c *Cache) Set(k string, v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.values[k]; ok {
		e.Value = cacheEntry{k, v}
		c.lru.MoveToFront(e)
		return
	}
	c.values[k] = c.lru.PushFront(cacheEntry{k, v})
	for c.lru.Len() > c.limit {
		e := c.lru.Back()
		delete(c.values, e.Value.(cacheEntry).key)
		c.lru.Remove(e)
	}
}

type Service struct {
	budget requestBudget
	Repo   Repository
	Models *inference.Manager
	Cache  *Cache
}

func (s *Service) Calculate(ctx context.Context, desc d.Descriptor) (d.Response, error) {
	snap, model, err := s.Repo.Snapshot(ctx, desc.Selection.SnapshotID)
	if err != nil {
		return d.Response{}, err
	}
	if err = Validate(desc, snap); err != nil {
		return d.Response{}, err
	}
	desc = Normalize(desc)
	release, err := s.budget.acquire(ctx, desc.Selection)
	if err != nil {
		return d.Response{}, err
	}
	defer release()
	hours, err := s.Repo.Hours(ctx, snap, SelectionInput{desc.Selection.RouteIDs, desc.Selection.Window})
	if err != nil {
		return d.Response{}, err
	}
	missing := []d.Hour{}
	indices := []int{}
	keys := make([]string, len(hours))
	for i, h := range hours {
		key := snap.Provenance.SnapshotID + "|" + h.RouteID + "|" + h.Time.UTC().Format(time.RFC3339)
		keys[i] = key
		if v, ok := s.Cache.Get(key); ok {
			hours[i].Boardings = v
		} else {
			missing = append(missing, h)
			indices = append(indices, i)
		}
	}
	const chunkSize = 128
	for start := 0; start < len(missing); start += chunkSize {
		end := min(start+chunkSize, len(missing))
		chunk := missing[start:end]
		if model.Release == "published" {
			chunk, err = s.Repo.Features(ctx, snap, chunk)
			if err != nil {
				return d.Response{}, err
			}
		}
		out, err := s.Models.Predict(ctx, model, snap, chunk)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return d.Response{}, d.Caused(504, "INFERENCE_TIMEOUT", "Время инференса истекло.", "inference", err)
			}
			return d.Response{}, d.Caused(503, "MODEL_UNAVAILABLE", "Модель или её входы недоступны.", "inference", err)
		}
		for j, i := range indices[start:end] {
			hours[i].Boardings = out[j]
			s.Cache.Set(keys[i], out[j])
		}
		// Do not retain large feature maps across chunks or in the returned frames.
		chunk = nil
	}
	result, err := Calculate(desc, snap, hours)
	if err != nil {
		return result, err
	}
	Enrich(&result)
	return result, nil
}
