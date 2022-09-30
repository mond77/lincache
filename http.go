package lincache

import (
	"fmt"
	"io/ioutil"
	
	"lincache/consistenthash"
	"log"
	
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const(
	defaultBasePath = "/lincache/"
	defaultReplicas = 50
)
type HTTPPool struct {
	//this peer's base URL,e.g. "https://example.net:8000"
	self     string
	//"/lincache/"
	basePath string

	mu sync.Mutex	//guards peers and httpGetters
	peers *consistenthash.Map
	httpGetters map[string]PeerGetter //keyed by e.g. "http://10.0.0.2:8008"
}
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))

}

//处理Http请求，返回结果
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path")
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	//  /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupname := parts[0]
	key := parts[1]

	group := GetGroup(groupname)
	if group == nil {
		http.Error(w, "no such group: "+groupname, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

//////////implement PeerPicker

func (p *HTTPPool) PickPeer(key string) (PeerGetter,bool){
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key);peer != "" && peer != p.self{
		p.Log("Pick peer %s",peer)
		return p.httpGetters[peer],true
	}
	return nil,false
}

//Set updates the pool's list of peers.
func(p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas,nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]PeerGetter,len(peers))
	for _,peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}
//////////http client

type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)
