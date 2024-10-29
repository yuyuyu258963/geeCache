package singleflight

import "sync"

// 用来代表正在进行中，或已经结束的请求。使用wait.Group避免重入
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// 管理不同key的请求（call）
type Group struct {
	mu sync.Mutex // protects m
	m  map[string]*call
}

// 将多个对多个相同的key的请求合并
func (g *Group) Do(key string,
	fn func() (interface{}, error)) (interface{}, error) {

	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	if c, ok := g.m[key]; ok { // 已经存在这个请求
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock() // 此时已经暂时完成了对g上数据结构体的访问

	//后面执行fn获取数据的时间可能比较长，所以得先Unlock一下
	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
