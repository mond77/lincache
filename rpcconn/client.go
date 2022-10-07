package rpcconn

import (
	"errors"
	pt "lincache/proto"
	"net"
	"sync"

	"google.golang.org/protobuf/proto"
)

var keepalive bool

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

	//mu sync.Mutex
	pending map[string]*Call

	close bool
}

func NewRPCClient(addr string) (*RPCClient, error) {
	cli := &RPCClient{dest: addr, pending: make(map[string]*Call)}
	return cli, nil
}

func (cli *RPCClient) call(req *pt.Request, resp *pt.Response) error {
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
	cli.wg.Add(1)
	call := &Call{
		req:  req,
		resp: resp,
		Done: make(chan *Call),
	}
	cli.send(call)
	<-call.Done
	cli.wg.Done()
	go func(){
		if !keepalive {
			cli.once.Do(func(){
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
		respbytes := make([]byte,1024)
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

func Setkeepalive(keep bool) {
	keepalive = keep
}
