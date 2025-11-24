package cache

import (
	"sync"
	"time"
)

// LinkedListCache cache that will use single-linked list to delete the oldest entries.
type LinkedListCache[K comparable, V any] struct {
	id   uint64
	lock *sync.RWMutex

	cache map[K]*linkedEntry[K, V]
	head  *linkedEntry[K, V]
	tail  *linkedEntry[K, V]

	weight uint

	expireAfterWrite int64
	maxWeight        uint
	keyWeigher       Weighter[K]
	valueWeigher     Weighter[V]
}

type linkedEntry[K any, V any] struct {
	key       K
	value     V
	expiresAt int64

	prev *linkedEntry[K, V]
	next *linkedEntry[K, V]
}

func NewLinkedListCache[K comparable, V any](
	maxEntries uint,
	expireAfterWrite time.Duration) *LinkedListCache[K, V] {

	lock.Lock()
	defer lock.Unlock()
	lastCacheId++
	cache := &LinkedListCache[K, V]{
		id:               lastCacheId,
		lock:             &sync.RWMutex{},
		cache:            make(map[K]*linkedEntry[K, V], maxEntries+1),
		head:             nil,
		tail:             nil,
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

func (c *LinkedListCache[K, V]) Put(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()
	e, ok := c.cache[key]
	if ok {
		c.remove(e)
	}

	e = &linkedEntry[K, V]{
		key:       key,
		value:     value,
		expiresAt: time.Now().UnixMilli() + c.expireAfterWrite,
		prev:      c.tail,
	}
	c.cache[key] = e

	if c.tail != nil {
		c.tail.next = e
	}
	c.tail = e
	if c.head == nil {
		c.head = e
	}

	c.weight += c.keyWeigher.WeightOf(key) + c.valueWeigher.WeightOf(value)
	for c.weight > c.maxWeight {
		c.removeEldest()
	}
}

func (c *LinkedListCache[K, V]) Get(key K) (value V, ok bool) {
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

func (c *LinkedListCache[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()
	e, ok := c.cache[key]
	if ok {
		c.remove(e)
	}
}

func (c *LinkedListCache[K, V]) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache = make(map[K]*linkedEntry[K, V])
	c.head = nil
	c.tail = nil
	c.weight = 0
}

func (c *LinkedListCache[K, V]) Close() {
	lock.Lock()
	defer lock.Unlock()
	delete(cacheCleanups, c.id)

	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache = nil
}

func (c *LinkedListCache[K, V]) removeEldest() {
	if c.head != nil {
		c.remove(c.head)
	}
}

func (c *LinkedListCache[K, V]) removeExpiredWithLock() {
	c.lock.Lock()
	defer c.lock.Unlock()
	for c.removeExpired() {
	}
}

func (c *LinkedListCache[K, V]) removeExpired() bool {
	if c.head != nil {
		if c.head.expiresAt < time.Now().UnixMilli() {
			c.remove(c.head)
			return true
		}
	}
	return false
}

func (c *LinkedListCache[K, V]) remove(e *linkedEntry[K, V]) {
	c.weight -= c.keyWeigher.WeightOf(e.key)
	c.weight -= c.valueWeigher.WeightOf(e.value)
	c.unlink(e)
	delete(c.cache, e.key)
}

func (c *LinkedListCache[K, V]) unlink(e *linkedEntry[K, V]) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}
}
