package main

import pb "geeCache/cachepb"

// PeerPicker is the interface that ,ust be implemented to locate
// the peer that owns a specific key
// 接口需要实现传入key选择相应的节点
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implement by a peer
// 接口PeerGetter的Get方法用于从对应的group查找缓存值
type PeerGetter interface {
	// Get(group string, key string) ([]byte, error)
	Get(in *pb.Request, out *pb.Response) error
}
