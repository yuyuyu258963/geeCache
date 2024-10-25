package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("got %v, want %v", v, expect)
	}
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "888",
	"Mike": "321",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	gee := NewGroup("scores", 2<<10,
		GetterFunc(func(key string) ([]byte, error) { // 设置回调函数
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok { // 第一次添加的情况
					loadCounts[key] = 0
				}
				// 记录其中的数据都都缺失了几次cache
				loadCounts[key]++
				return []byte(v), nil
			}
			return nil, fmt.Errorf("not found key:%s", key)
		}),
	)

	// 测试get是否能够成功
	for k, v := range db {
		if res, err := gee.Get(k); err != nil || res.String() != v {
			t.Fatalf("not get the value of %s", k)
		}
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 { //看看populateCache是否成功
			t.Fatalf("cache %s miss twice", k)
		}
	}

	// 测试不存在数据是否返回nil
	if view, err := gee.Get("unknown"); err == nil {
		t.Fatalf("the value of unknown should be empty, but %v got\n", view)
	}

}
