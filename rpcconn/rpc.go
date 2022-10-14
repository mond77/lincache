package rpcconn

import (
	"context"
	"fmt"
	"lincache"
	"lincache/consistenthash"
	"lincache/proto"
	"log"
	"sync"
	"time"

	
)

const (
	defaultReplicas    = 50
	defaultkeepalive   = false
	defaultCallTimeOut = 10 * time.Second
)

type RPCPool struct {
	addr string

	mu    sync.Mutex //guards peers and rpcGetters
	peers *consistenthash.Map

	rpcGetters map[string]lincache.PeerGetter
}

func NewRPCPool(addr string) *RPCPool {
	return &RPCPool{addr: addr}
}

func (rp *RPCPool) Getaddr() string {
	return rp.addr
}

func (rp *RPCPool) PickPeer(key string) (peer lincache.PeerGetter, ok bool) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if peer := rp.peers.Get(key); peer != "" && peer != rp.addr {
		rp.Log("Pick peer %s for key : %s", peer, key)
		return rp.rpcGetters[peer], true
	}
	return nil, false
}
func (p *RPCPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.addr, fmt.Sprintf(format, v...))

}

func (p *RPCPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.rpcGetters = make(map[string]lincache.PeerGetter, len(peers))
	for _, peer := range peers {
		cli := NewRPCClient(peer, defaultkeepalive)
		p.rpcGetters[peer] = &RPCGetter{cli, defaultCallTimeOut}
	}
}

//RPC调用超时控制
type RPCGetter struct {
	cli     *RPCClient
	timeOut time.Duration
}

func (rg *RPCGetter) Get(group string, key string) ([]byte, error) {
	resp := &proto.Response{}
	req := &proto.Request{
		Group: group,
		Key:   key,
	}
	ctx, _ := context.WithTimeout(context.Background(), rg.timeOut)
	err := rg.cli.call(ctx, req, resp)
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}
