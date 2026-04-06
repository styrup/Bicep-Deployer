package handler

import (
	"context"
	"sync"
	"time"
)

// cachedTemplate holds a downloaded template's content and its fetch time.
type cachedTemplate struct {
	content   string
	fetchedAt time.Time
}

// CachedStore wraps a TemplateStore with an in-memory TTL cache for template
// content and the template list. This avoids downloading every template from
// blob storage on each list request.
type CachedStore struct {
	inner TemplateStore
	ttl   time.Duration

	mu        sync.RWMutex
	listCache []string
	listTime  time.Time
	templates map[string]cachedTemplate
}

// NewCachedStore creates a caching wrapper around a TemplateStore.
func NewCachedStore(inner TemplateStore, ttl time.Duration) *CachedStore {
	return &CachedStore{
		inner:     inner,
		ttl:       ttl,
		templates: make(map[string]cachedTemplate),
	}
}

// ListTemplates returns cached blob names, refreshing when the TTL expires.
func (c *CachedStore) ListTemplates(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	if c.listCache != nil && time.Since(c.listTime) < c.ttl {
		names := c.listCache
		c.mu.RUnlock()
		return names, nil
	}
	c.mu.RUnlock()

	names, err := c.inner.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.listCache = names
	c.listTime = time.Now()
	c.mu.Unlock()

	return names, nil
}

// DownloadTemplate returns cached content, refreshing when the TTL expires.
func (c *CachedStore) DownloadTemplate(ctx context.Context, name string) (string, error) {
	c.mu.RLock()
	if cached, ok := c.templates[name]; ok && time.Since(cached.fetchedAt) < c.ttl {
		c.mu.RUnlock()
		return cached.content, nil
	}
	c.mu.RUnlock()

	content, err := c.inner.DownloadTemplate(ctx, name)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.templates[name] = cachedTemplate{content: content, fetchedAt: time.Now()}
	c.mu.Unlock()

	return content, nil
}
