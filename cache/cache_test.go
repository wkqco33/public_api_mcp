package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := New[string, string]()
	c.Set("key", "value", time.Minute)

	got, ok := c.Get("key")
	if !ok {
		t.Fatal("캐시 miss — 기대: hit")
	}
	if got != "value" {
		t.Fatalf("값 불일치: got=%q, want=%q", got, "value")
	}
}

func TestCacheMiss(t *testing.T) {
	c := New[string, int]()

	got, ok := c.Get("notexist")
	if ok {
		t.Fatal("캐시 hit — 기대: miss")
	}
	if got != 0 {
		t.Fatalf("zero value 기대: got=%d", got)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := New[string, string]()
	c.Set("key", "value", 50*time.Millisecond)

	if _, ok := c.Get("key"); !ok {
		t.Fatal("만료 전 캐시 miss — 기대: hit")
	}

	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Get("key"); ok {
		t.Fatal("만료 후 캐시 hit — 기대: miss")
	}
}

func TestCacheOverwrite(t *testing.T) {
	c := New[string, string]()
	c.Set("key", "first", time.Minute)
	c.Set("key", "second", time.Minute)

	got, ok := c.Get("key")
	if !ok {
		t.Fatal("캐시 miss")
	}
	if got != "second" {
		t.Fatalf("덮어쓰기 실패: got=%q, want=%q", got, "second")
	}
}

func TestCacheIntType(t *testing.T) {
	c := New[int, int]()
	c.Set(42, 100, time.Minute)

	got, ok := c.Get(42)
	if !ok || got != 100 {
		t.Fatalf("int 캐시 실패: got=%d, ok=%v", got, ok)
	}

	_, ok = c.Get(99)
	if ok {
		t.Fatal("없는 키에서 hit — 기대: miss")
	}
}

func TestCacheConcurrent(t *testing.T) {
	c := New[int, int]()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			c.Set(n, n*2, time.Minute)
		}(i)
		go func(n int) {
			defer wg.Done()
			c.Get(n)
		}(i)
	}
	wg.Wait()
}
