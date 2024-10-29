package main

import (
	"fmt"
	"geeCache/consistenthash"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// as the server's base url prefix
const defaultBasePath = "/_geecache/"
const defaultReplicas = 50

// 服务端
// 集成一致性哈希以及以客户端访问远端节点的能力
type HTTPPool struct {
	self     string
	basePath string // 为了与其他服务进行区分

	mu          sync.Mutex             // guards the httpGetter
	peers       *consistenthash.Map    // 一致性哈希映射器
	httpGetters map[string]*httpGetter // 远端服务节点
}

// NewHTTPPol initializes an HTTP pool for peers
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:        self,
		basePath:    defaultBasePath,
		peers:       consistenthash.New(defaultReplicas, nil),
		httpGetters: make(map[string]*httpGetter),
	}
}

// Log to record the request history
func (p *HTTPPool) Log(format string, args ...interface{}) {
	log.Printf("[server %s] %s", p.self, fmt.Sprintf(format, args...))
}

// to implement the http.Handler
// 服务端
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, defaultBasePath) {
		panic("not serve path " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// 获取defaultBasePath后的接口
	parts := strings.SplitN(r.URL.Path[len(defaultBasePath):], "/", 2)
	// 检查是否符合服务规则
	if len(parts) != 2 {
		p.Log("not found the group or key %s", r.URL.Path)
		http.Error(w, "not found the group or key ", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 从本地获取缓存数据
	// 获取缓存组
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "not match the group", http.StatusNotFound)
		return
	}

	// 尝试获取key对应的value
	val, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream") // 以字节流的形式返回
	w.Write(val.ByteSlice())
}

// 添加远端服务的节点
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 添加物理节点
	// 先要将新物理节点添加到一致性哈希的映射上
	// 同时要记录物理节点名到服务名的映射
	p.peers.Add(peers...)
	// 是否需要检查相同的peer的情况，如果设置了相同的peer可能需要panic或错误处理
	for _, peer := range peers {
		p.httpGetters[peer] = newHttpGetter(peer)
	}
}

// 调用一致性缓存获取key到realnode的映射，然后向realnode转发请求
// 或者这个realnode是本机的，可以直接访问本机的group
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	peer := p.peers.Get(key)
	if peer == "" || peer == p.self { // 未注册远端节点或peer直接本地的HTTPPool，则返回false；找不到服务的节点或直接等于本机的节点应该由本机的group处理
		// 应该对应Group处理流程的代码理解比较合适
		return nil, false
	}

	hg, ok := p.httpGetters[peer]
	return hg, ok
}

// 提供远端访问节点的功能
// 以客户端作为角色
type httpGetter struct {
	baseURL string // remote node's ip:port
}

func newHttpGetter(baseURL string) *httpGetter {
	return &httpGetter{baseURL: baseURL}
}

// Query from remote node
func (s *httpGetter) Get(group string, key string) ([]byte, error) {
	// 向远端节点的请求地址
	m := fmt.Sprintf("%v%v/%v",
		s.baseURL+defaultBasePath, // defaultBasePath 作为跟路由表示请求的是cache服务
		url.QueryEscape(group),    // url.QueryEscape 用于对字符串进行URL编码，用于在URL中嵌入特殊字符，将非数字字符转化为百分号后跟两位十六进制数，使得这些字符可以安全地被包含在url中
		url.QueryEscape(key),
	)

	response, err := http.Get(m)
	if err != nil {
		log.Printf("[m:%s] Get Error %s ", m, err.Error())
		return nil, err
	}
	defer response.Body.Close() // don't forget to close body or will be unsafe

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned:  %v", response.Status)
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %v", err)
	}

	// success Get
	log.Printf("success Get from %v", m)
	return bytes, nil
}
