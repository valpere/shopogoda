// Package cache provides a small in-process TTL cache with a size bound.
package cache

import (
	"container/list"
	"sync"
	"time"
)

// Cache is a concurrency-safe string cache with per-entry TTL. When it holds
// max entries, the least recently used one is evicted. It lives in process
// memory and is empty after a restart.
type Cache struct {
	mu    sync.Mutex
	max   int
	order *list.List // front = most recently used
	items map[string]*list.Element
	now   func() time.Time
}

type entry struct {
	key     string
	value   string
	expires time.Time
}

// New returns a cache holding at most max entries.
func New(max int) *Cache {
	return &Cache{
		max:   max,
		order: list.New(),
		items: make(map[string]*list.Element),
		now:   time.Now,
	}
}

// Get returns the value for key if present and not expired.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return "", false
	}
	e := el.Value.(*entry)
	if !c.now().Before(e.expires) {
		c.remove(el)
		return "", false
	}
	c.order.MoveToFront(el)
	return e.value, true
}

// Set stores value under key for ttl. A non-positive ttl stores nothing.
func (c *Cache) Set(key, value string, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	expires := c.now().Add(ttl)
	if el, ok := c.items[key]; ok {
		e := el.Value.(*entry)
		e.value, e.expires = value, expires
		c.order.MoveToFront(el)
		return
	}
	c.items[key] = c.order.PushFront(&entry{key: key, value: value, expires: expires})
	for c.order.Len() > c.max {
		c.remove(c.order.Back())
	}
}

// Delete removes key; it is a no-op when absent.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.remove(el)
	}
}

// Len returns the number of stored entries, including not-yet-collected expired ones.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

func (c *Cache) remove(el *list.Element) {
	c.order.Remove(el)
	delete(c.items, el.Value.(*entry).key)
}
