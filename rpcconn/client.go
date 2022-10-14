package rpcconn

import (
	"context"
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
	Done chan struct{}//仅作为一个信号
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

func (cli *RPCClient) call(ctx context.Context,req *pt.Request, resp *pt.Response)( err error) {
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
		Done: make(chan struct{}),
	}
	//发送并且开一个goroutine接收
	err = cli.send(call)
	go cli.receive(call)
	if err!=nil{
		call.err = err
		return call.err
	}
	go closeconn(cli)

	select{
	case <-ctx.Done():
		//接受未完成的cli.receive(call)的信号，以便在该函数关闭channel
		go func() {<-call.Done}()
		call.err = errors.New("time out")
	case <-call.Done:
	}
	cli.wg.Done()
	return call.err
}

func (cli *RPCClient) send(call *Call) error {
	reqbytes, err := proto.Marshal(call.req)
	if err != nil {//return err后call函数中会赋给call.Done
		return err
	}
	cli.sending.Lock()
	defer cli.sending.Unlock()
	
	_, err = cli.conn.Write(reqbytes)
	if err != nil {
		return err
	}

	return nil
}

//收到回复，或者错误，call.Done
func (cli *RPCClient) receive(call *Call) {
	var err error
	if err == nil {
		respbytes := make([]byte, 1024)
		n, err := cli.conn.Read(respbytes)
		if err != nil {
			call.err = err
			call.Done <- struct{}{}
			return
		}
		respbytes = respbytes[:n]
		err = proto.Unmarshal(respbytes, call.resp)
		if err != nil {
			call.err = err
			call.Done <- struct{}{}
			return
		}
		call.Done <- struct{}{}
	}
	//生产者主动关闭channel
	close(call.Done)
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

func closeconn(cli *RPCClient) {
	if !cli.keepalive {
		cli.once.Do(func() {
			//等待所有请求结束
			cli.wg.Wait()
			cli.conn.Close()
			cli.conn = nil
		})
	}
}