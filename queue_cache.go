package cache

import (
	"sync"
	"time"

	"github.com/viger-pro/go-collections"
)

// QueueCache cache that will use a queue to delete the oldest entries.
type QueueCache[K comparable, V any] struct {
	id   uint64
	lock *sync.RWMutex

	cache map[K]*queueEntry[K, V]
	queue collections.Queue[*queueEntry[K, V]]

	weight uint

	expireAfterWrite int64
	maxWeight        uint
	keyWeigher       Weighter[K]
	valueWeigher     Weighter[V]
}

type queueEntry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt int64
}

func NewQueueCache[K comparable, V any](maxEntries uint, expireAfterWrite time.Duration) *QueueCache[K, V] {
	lock.Lock()
	defer lock.Unlock()
	lastCacheId++
	cache := &QueueCache[K, V]{
		id:               lastCacheId,
		lock:             &sync.RWMutex{},
		cache:            make(map[K]*queueEntry[K, V], maxEntries+1),
		queue:            collections.NewArrayQueue[*queueEntry[K, V]](),
		weight:           0,
		expireAfterWrite: expireAfterWrite.Milliseconds(),
		maxWeight:        maxEntries,
		keyWeigher:       &CountElementsWeigher[K]{},
		valueWeigher:     &ZeroWeighter[V]{},
	}
	cacheCleanups[lastCacheId] = cache.removeExpiredWithLock
	startCachesCleanup()
	return cache
}

func (c *QueueCache[K, V]) Put(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()
	e, ok := c.cache[key]
	if ok {
		c.remove(e)
	}

	e = &queueEntry[K, V]{
		key:   key,
		value: value,
		expiresAt: time.Now().UnixMilli() + c.expireAfterWrite,
	}
	c.cache[key] = e
	c.queue.AddLast(e)

	c.weight += c.keyWeigher.WeightOf(key) + c.valueWeigher.WeightOf(value)
	for c.weight > c.maxWeight {
		c.removeEldest()
	}
}

func (c *QueueCache[K, V]) Get(key K) (value V, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	e, ok := c.cache[key]
	if ok {
		if e.expiresAt > time.Now().UnixMilli() {
			return e.value, true
		}
	}
	return value, false
}

func (c *QueueCache[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()
	e, ok := c.cache[key]
	if ok {
		c.remove(e)
	}
}

func (c *QueueCache[K, V]) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache = make(map[K]*queueEntry[K, V])
	c.queue = collections.NewArrayQueue[*queueEntry[K, V]]()
	c.weight = 0
}

func (c *QueueCache[K, V]) Close() {
	lock.Lock()
	delete(cacheCleanups, c.id)
	lock.Unlock()
	c.lock.Lock()
	c.cache = nil
	c.queue = nil
	c.lock.Unlock()
}

func (c *QueueCache[K, V]) removeEldest() {
	e, err := c.queue.RemoveFirst()
	if err == nil {
		c.remove(e)
	}
}

func (c *QueueCache[K, V]) removeExpiredWithLock() {
	c.lock.Lock()
	defer c.lock.Unlock()
	for c.removeExpired() {
	}
}

func (c *QueueCache[K, V]) removeExpired() bool {
	entry, err := c.queue.RemoveFirst()
	if err == nil && entry.expiresAt < time.Now().UnixMilli() {
		c.remove(entry)
		return true
	}
	return false
}

func (c *QueueCache[K, V]) remove(e *queueEntry[K, V]) {
	if e.expiresAt >= 0 {
		c.weight -= c.keyWeigher.WeightOf(e.key)
		c.weight -= c.valueWeigher.WeightOf(e.value)
		e.expiresAt = -1
		delete(c.cache, e.key)
	}
}



