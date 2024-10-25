package lru

import (
	"container/list"
	"log"
)

// Cache is a LRU cache, It is not safe for concurrent access
type Cache struct {
	maxBytes int64
	nbytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	// optional and executed when an entry is purged
	// 某条记录被移除时的回调函数，可以是nil
	// 因为插入的时候出现了removeOldest，所以使用者可能希望移除的是什么，再对应地去操作
	OnEvicted func(key string, value Value)
}

// entry is the data's type which  is stored in cache
type entry struct {
	key   string // 在双链表的元素也存储key是为了方便在map上做删除
	value Value
}

// Value use len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		nbytes:   0,
		ll:       list.New(),

		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	elem := c.ll.Back()
	if elem != nil {
		c.ll.Remove(elem)
		kv := elem.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Get look ups a key's value
func (c *Cache) Get(key string) (Value, bool) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		val := elem.Value.(*entry)
		return val.value, true
	}
	return nil, false
}

func (c *Cache) onOversized(sz int64) {
	log.Printf("Add failed: kv is too large. maxBytes: %d but sizeof key+val: %d",
		c.maxBytes, sz)
}

// Add can insert a new key value into cache
// if the key already exists then cover the existing value
func (c *Cache) Add(key string, val Value) Value {
	var nEntrySize int64
	if int64(len(key))+int64(val.Len()) > c.maxBytes {
		c.onOversized(int64(len(key)) + int64(val.Len()))
		return nil
	}

	if elem, ok := c.cache[key]; ok { // 覆盖原来的值
		kv := elem.Value.(*entry)
		oldVal := kv.value
		// check if update with too more memory
		nEntrySize = int64(val.Len()) - int64(oldVal.Len())

		for c.nbytes+nEntrySize > c.maxBytes { // will overflow then call LRU
			c.RemoveOldest()
		}

		kv.value = val
		c.ll.MoveToFront(elem)
		c.nbytes += nEntrySize
		return val
	} else {
		nEntrySize = int64(len(key)) + int64(val.Len())
		for c.nbytes+nEntrySize > c.maxBytes { // will overflow then call LRU
			c.RemoveOldest()
		}

		// insert the new entry and update the size
		ele := c.ll.PushFront(&entry{key, val})
		c.cache[key] = ele
		c.nbytes += nEntrySize
	}

	return val
}

// delete a cache entry with key and return the value
// if not found then return nil
func (c *Cache) Del(key string) Value {
	if elem, ok := c.cache[key]; ok {
		kv := c.ll.Remove(elem).(*entry) // remove from list
		delete(c.cache, key)             // delete from cache
		val := kv.value
		c.nbytes -= int64(len(key)) + int64(val.Len()) // update the size of the cache
		return val
	}

	return nil
}

// 获取添加了多少条数据
func (c *Cache) Len() int {
	return c.ll.Len()
}
