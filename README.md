# lincache
distributed k/v cache system with rpc communication surpported.most of codes here refer to geecache at https://github.com/geektutu/7days-golang.Thanks for geektutu.

### httpserver test 

eg:

```sh
cd server
./run.sh
```

### rpcconn test

 eg:

```sh
cd rpcconn/test
./run.sh
```

#### ilumination

In the test of rpcconn,we get keys from the api server by the command of "curl "http://localhost:9999/api?key=[key]". The api server communicates with peers from remote server that we replace with localhost socket by rpc call, which is implemented in the rpcconn file.

#### Some bugs encountered

[Lincache] Failed to get from peer read tcp 127.0.0.1:53568->127.0.0.1:8003: use of closed network connection

To fix it :	add **sync.WaitGroup** and **sync.Once** to RPCClient making sure that all the ongoing calls will be done before the conn is closed once.

#### promoting
test only one group means not overall test with multiple groups.later i will complete it.
solution:   所有group存在全局变量groups里，
    peers := lincache.NewHTTPPool(addr)//服务端
	peers.Set(addrs...)//设置客户端
	g.RegisterPeers(peers)//为g这个group注册PeerPicker
    http.ListenAndServe(addr[7:], peers)//启动服务，在本地的groups里的所有group
