package rpcconn

import (
	"errors"
	pt "lincache/proto"
	"net"
	"sync"

	"google.golang.org/protobuf/proto"
)


type Call struct {
	Seq  uint64
	req  *pt.Request
	resp *pt.Response

	err  error
	Done chan *Call
}

//该项目中只有一种调用
type RPCClient struct {
	sending sync.Mutex

	dest string

	//wg保护还没完成的请求，保证call都完成后再关闭连接
	wg sync.WaitGroup
	//连接只关闭一次
	once sync.Once
	conn net.Conn

	//pending 实现瞬时并发调用的缓存，但singleflight做到了防止缓存击穿
	//mu sync.Mutex
	//pending map[string]*Call
	keepalive bool
	close     bool
}

func NewRPCClient(addr string,keepalive bool) *RPCClient {
	cli := &RPCClient{dest: addr,keepalive: keepalive}
	return cli
}

func (cli *RPCClient) call(req *pt.Request, resp *pt.Response)( err error) {
	if cli.conn == nil {
		conn, err := net.Dial("tcp", cli.dest)
		if err != nil {
			return err
		}
		cli.conn = conn
	}
	if cli.close {
		return errors.New("cli is closed,call failure")
	}
	//get from pending
	cli.wg.Add(1)
	call := &Call{
		req:  req,
		resp: resp,
		Done: make(chan *Call),
	}
	//
	err = cli.send(call)
	if err!=nil{
		call.err = err
		return call.err
	}
	<-call.Done
	cli.wg.Done()
	go func() {
		if !cli.keepalive {
			cli.once.Do(func() {
				cli.wg.Wait()
				cli.conn.Close()
				cli.conn = nil
			})
		}
	}()
	return call.err
}

func (cli *RPCClient) send(call *Call) error {
	reqbytes, err := proto.Marshal(call.req)
	if err != nil {
		call.err = err
		return err
	}
	cli.sending.Lock()
	defer cli.sending.Unlock()
	go call.receive(cli.conn)
	_, err = cli.conn.Write(reqbytes)
	if err != nil {
		call.err = err
		return err
	}

	return nil
}

//收到回复，或者错误，call.Done
func (call *Call) receive(conn net.Conn) {
	var err error
	if err == nil {
		respbytes := make([]byte, 1024)
		n, err := conn.Read(respbytes)
		if err != nil {
			call.err = err
			call.Done <- call
			return
		}
		respbytes = respbytes[:n]
		err = proto.Unmarshal(respbytes, call.resp)
		if err != nil {
			call.err = err
			call.Done <- call
			return
		}

		call.Done <- call
	}

}

// func (cli *RPCClient) addcall(call *Call){
// 	cli.mu.Lock()
// 	defer cli.mu.Unlock()
// 	cli.pending[call.req.GetGroup()+"-"+call.req.GetKey()] = call
// }

// func (cli *RPCClient) getcall(group,key string)*Call{
// 	cli.mu.Lock()
// 	defer cli.mu.Unlock()
// 	if call,ok := cli.pending[group+"-"+key];ok{
// 		return call
// 	}
// 	return nil
// }

