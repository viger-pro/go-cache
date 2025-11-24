package cache

import (
	"log"
	"testing"
	"time"
)

type testCase struct {
	name string
	cache Cache[int, int]
}

func createTestCases(n uint, expireAfterWrite time.Duration) []testCase {
	var result = make([]testCase, 0)
	result = append(result, testCase{
		"linked list cache",
		NewLinkedListCache[int, int](n, expireAfterWrite),
	})
	result = append(result, testCase{
		"queue cache",
		NewQueueCache[int, int](n, expireAfterWrite),
	})
	return result
}

func TestCache(t *testing.T) {
	var n int = 5
	var expirationTime time.Duration = 100 * time.Millisecond
	var testCases = createTestCases(uint(n), expirationTime)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var cache = testCase.cache
			var i int
			for i = 0; i < n; i++ {
				cache.Put(i, i*2)
			}
			for i = 0; i < n; i++ {
				var x int
				var ok bool
				x, ok = cache.Get(i)
				if !ok {
					t.Fatalf("no value for %d", i)
				}
				if x != i*2 {
					t.Fatalf("expected %d, got %d", i*2, x)
				}
			}
			for i = n; i < n*2; i++ {
				cache.Put(i, i*2)
			}
			for i = n; i < n*2; i++ {
				cache.Put(i, i*2)
			}
			for i = 0; i < n; i++ {
				var ok bool
				_, ok = cache.Get(i)
				if ok {
					t.Fatalf("value for %d should have been removed", i)
				}
			}
			for i = n; i < n*2; i++ {
				var x int
				var ok bool
				x, ok = cache.Get(i)
				if !ok {
					t.Fatalf("no value for %d", i)
				}
				if x != i*2 {
					t.Fatalf("expected %d, got %d", i*2, x)
				}
			}
			cache.Remove(n)
			_, ok := cache.Get(n)
			if ok {
				t.Fatalf("value for %d should have been removed", n)
			}

			time.Sleep(1*time.Second + expirationTime)
			for i = 0; i < n*2; i++ {
				_, ok := cache.Get(i)
				if ok {
					t.Fatalf("value for %d should have been removed", i)
				}
			}
		})
	}
}

func TestLargeTestLinkedCache(t *testing.T) {
	linkedCache := NewLinkedListCache[int, int](10_000, time.Minute)
	queueCache := NewQueueCache[int, int](10_000, time.Minute)
	n := 10_000_000
	largeTest("linked", linkedCache, n)
	largeTest("queue", queueCache, n)
}

func largeTest(testName string, cache Cache[int, int], n int) {
	start := time.Now()
	for i := 0; i < n; i++ {
		cache.Put(i, i)
	}
	for i := 0; i < n; i++ {
		cache.Get(i)
	}
	cache.Clear()
	end := time.Now()
	elapsed := end.Sub(start)
	log.Printf("%s test took %s", testName, elapsed)
}

func BenchmarkCachePut(b *testing.B) {
	var n uint = 1000_000
	var testCases = createTestCases(n, 1 * time.Second)
	for _, testCase := range testCases {
		b.Run(testCase.name, func(b *testing.B) {
			var cache = testCase.cache
			for i := 0; i < b.N; i++ {
				cache.Put(i, i * 2)
			}
		})
	}
}

func BenchmarkCacheGet(b *testing.B) {
	var n uint = 1000_000
	var testCases = createTestCases(n, 1 * time.Second)
	for _, testCase := range testCases {
		b.Run(testCase.name, func(b *testing.B) {
			cache := testCase.cache
			b.StopTimer()
			for i := 0; i < b.N; i++ {
				cache.Put(i, i*2)
			}
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				cache.Get(i)
			}
		})
	}
}
