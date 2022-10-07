# lincache
distributed k/v cache system with rpc communication surpported

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

To fix it :	add **sync.Mutex** and **sync.once** to RPCClient making sure that all the ongoing calls will be done before the conn is closed once.
