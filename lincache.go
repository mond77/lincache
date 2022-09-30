package lincache

import (
	"fmt"
	"lincache/singleflight"
	"log"
	
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

//define recall function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

//A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name      string    
	getter    Getter
	mainCache cache

	//integrete the distributed peers 
	peers 	  PeerPicker

	//use singleflight.Group to make sure that each key is only fetched once
	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

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
		loader: &singleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	return g.load(key)
}


//使用g.loader.Do将原来的load逻辑包裹起来，这样确保了并发场景下针对相同的key，load过程只会调用一次
func (g *Group) load(key string) (value ByteView, err error) {
	view,err := g.loader.Do(key,func() (interface{}, error){
		if g.peers != nil{
			if peer,ok := g.peers.PickPeer(key);ok{
				//remote peer picked,and get from peer
				if value,err := peer.Get(g.name,key);err !=nil{
					log.Println("[Lincache] Failed to get from peer",err)
				}else{
					return ByteView{value},nil
				}
			}
		}
	
		return g.getLocally(key)
	})
	if err != nil{
		return
	}
	return view.(ByteView),err
}

//call user method g.getter.Get()
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

//RegisterPeers registers a httpconn.PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil{
		panic("Registerhttpconn.PeerPicker")
	}
	g.peers = peers
}

