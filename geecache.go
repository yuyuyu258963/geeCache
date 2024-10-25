package main

import "sync"

/**
														是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

// A Getter loads data for a key
type Getter interface {
	Get(string) ([]byte, error)
}

// 这么做实际上将一个函数封装成了一个
// 实现Getter接口的回调函数
// 之后用户只要是传入 funcGetter 类型的函数，就可以被封装为Getter函数
type GetterFunc func(string) ([]byte, error)

// 函数接口
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

var (
	mu     sync.RWMutex // 负责实现归Groups的并发访问
	groups = make(map[string]*Group)
)

// Get value from group's cache
func (g *Group) Get(key string) (val ByteView, err error) {
	var ok bool

	val, ok = g.mainCache.get(key)
	if ok { // 直接在cache中找到了数据
		return val, nil
	}
	// 未找到则从回调函数中查找
	val, err = g.load(key)
	return val, err
}

// 加载未在缓存上的数据
// 留出加载远程节点 or 源数据的接口
func (g *Group) load(key string) (ByteView, error) {
	return g.getLocally(key)
}

// 未找到数据时，根据回调函数获取key对应的cache
// 如果没拿到数据那就返回空
// 如果拿到了，需要将这个新拿到的kv记录到cache中
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil { // 出现错误则返回
		return ByteView{}, err
	}
	value := ByteView{bytes}
	g.populateCache(key, value)
	return value, nil
}

// 将没找到但是心找到的数据添加到cache中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g

	return g
}

// GetGroup returns the named group previously created with NewGroup
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}
