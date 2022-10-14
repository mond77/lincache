package touse

import (
	"lincache"
	"lincache/rpcconn"
	"log"
	"net/http"
)

func StartCacheServer(addr string, addrs []string, gee *lincache.Group) {
	peers := lincache.NewHTTPPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("lincache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func StartRPCServer(addr string, addrs []string, gee *lincache.Group) {
	peers := rpcconn.NewRPCPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("lincache is running at", addr)
	rpcconn.NewRPCServer(peers.Getaddr())
}

func StartAPIServer(apiAddr string, gee *lincache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")

			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			log.Printf("[ApiServer]get key %s,value %v",key,string(view.ByteSlice()))

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}