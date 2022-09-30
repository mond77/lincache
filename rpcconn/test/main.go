package main

import (
	"flag"
	"fmt"
	"lincache"
	"lincache/rpcconn"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

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

func startRPCServer(addr string, addrs []string, gee *lincache.Group) {
	peers := rpcconn.NewRPCPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("lincache is running at", addr)
	rpcconn.NewRPCServer(peers.Getaddr())
}

func startAPIServer(apiAddr string, gee *lincache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			fmt.Println(key)
			
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "lincache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "0.0.0.0:8001",
		8002: "0.0.0.0:8002",
		8003: "0.0.0.0:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go startAPIServer(apiAddr, gee)
	}
	startRPCServer(addrMap[port], []string(addrs), gee)
}
