package cache

import (
	"sync"
	"time"
)

type Cache[K comparable, V any] interface {
	Put(key K, value V)
	Get(key K) (value V, ok bool)
	Remove(key K)
	Clear()

	Close()
}

type WeightCalculator[T any] interface {
	WeightOf(t T) uint
}

type CountElementsWeightCalculator[T any] struct{}

func (c *CountElementsWeightCalculator[T]) WeightOf(_ T) uint {
	return 1
}

type ZeroWeightCalculator[T any] struct{}

func (c *ZeroWeightCalculator[T]) WeightOf(_ T) uint {
	return 0
}

var lock sync.Mutex = sync.Mutex{}
var lastCacheId uint64 = 0
var cacheCleanups map[uint64]func() = make(map[uint64]func())

func startCachesCleanup() {
	if lastCacheId == 1 {
		ticker := time.NewTicker(1 * time.Minute)
		go func() {
			for {
				<-ticker.C
				lock.Lock()
				for _, f := range cacheCleanups {
					f()
				}
				lock.Unlock()
			}
		}()
	}
}
