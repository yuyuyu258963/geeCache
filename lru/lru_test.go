package lru

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

type String string

func (d String) Len() int { return len(d) }

func TestGet(t *testing.T) {
	lru := New(int64(100000), nil)

	// 测试是否能够正常增加删除数据
	lru.Add("key1", String("123213"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "123213" {
		t.Fatalf("cache hit key1=123213 failed")
	}

	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache hit key2 is not a valid")
	}

	lru.Add("key1", String("ywh"))

	if lru.Len() != 1 {
		t.Fatalf("len failed len is:%d but expected 1", lru.Len())
	}

	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "ywh" {
		t.Fatalf("cache hit key1=ywh failed")
	}
	t.Logf("cache len is %d\n", lru.Len())
}

func TestDelete(t *testing.T) {
	lru := New(int64(100000), nil)
	lru.Add("key1", String("abc"))
	if lru.Del("aaa") != nil {
		t.Fatal("cache get unset value")
	}
	if val := string(lru.Del("key1").(String)); val != "abc" {
		t.Fatalf("Del return failed key1:abs but got key1:%s", val)
	}

	if lru.nbytes != 0 {
		t.Fatalf("not clear all leave %d bytes", lru.nbytes)
	}

	if lru.Len() != 0 {
		t.Fatalf("len failed len is:%d but expected 1", lru.Len())
	}
}

func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "value3"
	sz := len(k1 + k2 + v1 + v2)

	lru := New(int64(sz), func(s string, v Value) {
		fmt.Println("remove oldest ", s, string(v.(String)))
	})
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))
	if lru.nbytes != int64(len(k2+k3+v2+v3)) {
		log.Fatalf("update sz failed: %v != %v", lru.nbytes, int64(len(k2+k3+v2+v3)))
	}
	if _, ok := lru.Get(k1); ok {
		log.Fatalf("removedest failed\n")
	}
	if _, ok := lru.Get(k2); !ok {
		log.Fatalf("removedest failed\n")
	}

	lru.Del(k2)
	if lru.nbytes != int64(len(k3+v3)) {
		log.Fatalf("update sz failed: %v != %v", lru.nbytes, int64(len(k3+v3)))
	}

	if val, _ := lru.Get(k3); val == nil || string(val.(String)) != v3 {
		log.Fatalf("after removeoldest add operation failed\n")
	}
}

func TestOnEvicted(t *testing.T) {
	removedKeys := make([]string, 0)
	callback := func(key string, value Value) {
		removedKeys = append(removedKeys, key)
	}
	lru := New(int64(10), callback)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("k2"))
	lru.Add("k3", String("k3"))
	lru.Add("k4", String("k4"))

	expect := []string{"key1", "k2"}
	if !reflect.DeepEqual(expect, removedKeys) {
		t.Fatalf("expected onEvicted failed, expect keys %v", expect)
	}
}
