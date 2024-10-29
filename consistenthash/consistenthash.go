package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to int32
type Hash func(data []byte) uint32

// Map contains all hashed keys
type Map struct {
	hash     Hash           // hash function
	replicas int            //虚拟节点倍数
	keys     []int          //存储所有虚拟节点映射到的key，sorted
	hashMap  map[int]string // 虚拟节点到真是节点名称的映射
}

// New Create a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		keys:     make([]int, 0),
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE // 循环冗余校验码
	}
	return m
}

// Add adds some keys to the hash
func (m *Map) Add(keys ...string) {
	for _, k := range keys {
		for i := 0; i < m.replicas; i++ {
			// hs := int(m.hash([]byte(strconv.Itoa(i) + k)))
			hs := int(m.hash([]byte(k + "::" + strconv.Itoa(i))))
			m.keys = append(m.keys, hs)
			m.hashMap[hs] = k // 虚拟节点到真实节点的映射
		}
	}
	sort.Ints(m.keys)
}

// get the closet item int the hash to the provided key
// 当然这里可以用二分的方式
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hs := int(m.hash([]byte(key)))
	if hs > m.keys[len(m.keys)-1] {
		return m.hashMap[m.keys[0]]
	}
	l := 0
	r := len(m.keys) - 1
	for l < r {
		mid := (l + r) >> 1
		if m.keys[mid] >= hs { // 二分找到大于hs的最小值
			r = mid
		} else {
			l = mid + 1
		}
	}

	return m.hashMap[m.keys[l]]
}
