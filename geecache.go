package main

import (
	pb "geeCache/cachepb"
	"geeCache/singleflight"
	"log"
	"sync"
)

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
// 将缓存抽象为多个group，每个内核都是封装好的cache，支持并发访问
// 类似于redis的1~50的那种group
type Group struct {
	name      string
	getter    Getter // 当本地缓存和远端节点都加载失败的处理方法，用户提供
	mainCache cache

	peers PeerPicker // 用于获取远端处理节点
	// Use singleflight.Group to make sure that
	// each key is only fetch once
	singleLoader *singleflight.Group
}

var (
	mu     sync.RWMutex // 负责实现归Groups的并发访问
	groups = make(map[string]*Group)
)

// register peers for remote peers
func (g *Group) Register(peers PeerPicker) {
	if g.peers != nil {
		panic("registerPeerPicker called more than once")
	}
	g.peers = peers
}

// Get value from group's cache
func (g *Group) Get(key string) (val ByteView, err error) {
	var ok bool

	val, ok = g.mainCache.get(key) // 先尝试去本机的group查找
	if ok {                        // 直接在本机的节点上找到了数据
		return val, nil
	}
	// 未找到则从回调函数中查找
	val, err = g.load(key)
	return val, err
}

// 加载未在本机上缓存的数据
// 留出加载远程节点 or 源数据的接口
func (g *Group) load(key string) (value ByteView, err error) {
	//将短时间内多个相同key的请求合并
	viewi, err := g.singleLoader.Do(key, func() (interface{}, error) {
		if g.peers != nil { // 若有远端节点注册，则去远端节点查看
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", peer, err)
			}
		}

		// 若在远端节点查找失败，则转到本地节点处理
		return g.getLocally(key)
	})
	// 远端请求或slow DB加载数据结束
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
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

// 从远端peer中Get缓存
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}

	err := peer.Get(req, res) // 使用protobuf进行通信
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
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
		name:         name,
		getter:       getter,
		mainCache:    cache{cacheBytes: cacheBytes},
		singleLoader: new(singleflight.Group),
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
