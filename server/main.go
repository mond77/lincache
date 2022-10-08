package main

import (
	"flag"
	"fmt"
	"lincache"
	tu "lincache/touse"
	"log"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

//////HTTP服务端

// func main() {
// 	lincache.NewGroup("scores", 2<<10, lincache.GetterFunc(
// 		func(key string) ([]byte, error) {
// 			log.Println("[SlowDB] search key", key)
// 			if v, ok := db[key]; ok {
// 				return []byte(v), nil
// 			}
// 			return nil, fmt.Errorf("%s not exist", key)
// 		}))

// 	addr := "localhost:9999"
// 	peers := lincache.NewHTTPPool(addr)
// 	log.Println("lincache is running at", addr)
// 	log.Fatal(http.ListenAndServe(addr, peers))
// }

///////分布式节点

func createGroup() *lincache.Group {
	return lincache.NewGroup("scores", 2<<10, lincache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "lincache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go tu.StartAPIServer(apiAddr, gee)
	}
	tu.StartCacheServer(addrMap[port], []string(addrs), gee)
}
