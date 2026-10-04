package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func newTestCache(max int) (*Cache, *fakeClock) {
	clk := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	c := New(max)
	c.now = clk.now
	return c, clk
}

func TestCache_SetGet(t *testing.T) {
	c, _ := newTestCache(10)
	c.Set("k", "v", time.Minute)

	got, ok := c.Get("k")
	assert.True(t, ok)
	assert.Equal(t, "v", got)

	_, ok = c.Get("missing")
	assert.False(t, ok)
}

func TestCache_TTLExpiry(t *testing.T) {
	c, clk := newTestCache(10)
	c.Set("k", "v", time.Minute)

	clk.advance(59 * time.Second)
	_, ok := c.Get("k")
	assert.True(t, ok, "still fresh")

	clk.advance(2 * time.Second)
	_, ok = c.Get("k")
	assert.False(t, ok, "expired")
	assert.Equal(t, 0, c.Len(), "expired entry is dropped on read")
}

func TestCache_OverwriteResetsTTLAndValue(t *testing.T) {
	c, clk := newTestCache(10)
	c.Set("k", "old", time.Minute)
	clk.advance(50 * time.Second)
	c.Set("k", "new", time.Minute)
	clk.advance(50 * time.Second)

	got, ok := c.Get("k")
	assert.True(t, ok)
	assert.Equal(t, "new", got)
	assert.Equal(t, 1, c.Len())
}

func TestCache_Delete(t *testing.T) {
	c, _ := newTestCache(10)
	c.Set("k", "v", time.Minute)
	c.Delete("k")
	c.Delete("never-existed") // no-op

	_, ok := c.Get("k")
	assert.False(t, ok)
}

func TestCache_EvictsLeastRecentlyUsedAtCapacity(t *testing.T) {
	c, _ := newTestCache(3)
	c.Set("a", "1", time.Hour)
	c.Set("b", "2", time.Hour)
	c.Set("c", "3", time.Hour)
	c.Get("a") // a is now most recent; b is the LRU

	c.Set("d", "4", time.Hour)

	assert.Equal(t, 3, c.Len())
	_, ok := c.Get("b")
	assert.False(t, ok, "least recently used entry evicted")
	for _, k := range []string{"a", "c", "d"} {
		_, ok := c.Get(k)
		assert.True(t, ok, k)
	}
}

func TestCache_NonPositiveTTLIsNotStored(t *testing.T) {
	c, _ := newTestCache(3)
	c.Set("k", "v", 0)
	_, ok := c.Get("k")
	assert.False(t, ok)
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := New(50)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				k := fmt.Sprintf("k%d", (g*31+i)%80)
				c.Set(k, "v", time.Minute)
				c.Get(k)
				if i%7 == 0 {
					c.Delete(k)
				}
			}
		}(g)
	}
	wg.Wait()
	assert.LessOrEqual(t, c.Len(), 50)
}
