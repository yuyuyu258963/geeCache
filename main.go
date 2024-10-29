package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var Tdb = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
	"Ywh":  "555",
	"jc":   "123",
}

// 创建本地group
// 统一都是 scores 分组
func createGroup() *Group {
	return NewGroup("scores", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		if v, ok := Tdb[key]; ok {
			fmt.Println("[slow db] load key ", key)
			time.Sleep(time.Second * 2) // mock slow db load data slowly
			return []byte(v), nil
		}
		return nil, fmt.Errorf("[slow db] key: %s not exist", key)
	}))
}

// 开启一个缓存服务器
// 使用每个节点的服务端功能
func startCacheServer(addr string, addrs []string, g *Group) {
	peers := NewHTTPPool(addr) // 创建一个PeerPicker
	peers.Set(addrs...)        // 设置一致性哈希中的节点
	g.Register(peers)          // 将PeerPicker这个传入到g中，之后Group进行数据查找的时候就可以调用远端节点
	log.Println("cache is running at ", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 提供了一个类似于服务器的作用
func startAPIServer(apiAddr string, g *Group) {
	http.Handle("/api", http.HandlerFunc( // 监听api这个路由下的
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := g.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		},
	))

	log.Println("fontend server is running at ", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func serve() {
	// 获取运行指定参数
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "server port")
	flag.BoolVar(&api, "api", false, "start a api server")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{ // 远端节点组
		7998: "http://localhost:7998",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}
	g := createGroup()
	if api {
		go startAPIServer(apiAddr, g)
	}

	startCacheServer(addrMap[port], []string(addrs), g)
}

func main() {
	serve()
}
