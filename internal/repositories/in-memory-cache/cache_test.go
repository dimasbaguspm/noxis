package cache

import (
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := New()
	c.Set("key", 42, 0)
	v, ok := c.Get("key")
	if !ok || v != 42 {
		t.Fatalf("Get after Set = %v, %v; want 42, true", v, ok)
	}
	if _, ok := c.Get("missing"); ok {
		t.Fatal("Get(missing) = present; want absent")
	}
}

func TestCacheOverwrite(t *testing.T) {
	c := New()
	c.Set("key", "first", 0)
	c.Set("key", "second", 0)
	v, _ := c.Get("key")
	if v != "second" {
		t.Fatalf("Get = %v; want second", v)
	}
}

func TestCacheExpiry(t *testing.T) {
	c := New()
	c.Set("tmp", 1, 20*time.Millisecond)
	if _, ok := c.Get("tmp"); !ok {
		t.Fatal("fresh entry missing")
	}
	time.Sleep(30 * time.Millisecond)
	if _, ok := c.Get("tmp"); ok {
		t.Fatal("expired entry still present")
	}
	// the expired entry was dropped on read
	if _, ok := c.items["tmp"]; ok {
		t.Fatal("expired entry not removed from store")
	}
}

func TestCacheNoExpiryPersists(t *testing.T) {
	c := New()
	c.Set("keep", "v", 0)
	time.Sleep(10 * time.Millisecond)
	if _, ok := c.Get("keep"); !ok {
		t.Fatal("no-expiry entry dropped")
	}
}

func TestCacheDelete(t *testing.T) {
	c := New()
	c.Set("key", "v", 0)
	c.Delete("key")
	if _, ok := c.Get("key"); ok {
		t.Fatal("Get after Delete = present; want absent")
	}
}