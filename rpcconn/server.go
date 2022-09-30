package rpcconn

import (
	"fmt"
	"lincache"
	pt "lincache/proto"
	"log"
	"net"

	"google.golang.org/protobuf/proto"
)

type RPCServer struct{}

func NewRPCServer(addr string) *RPCServer {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil
	}
	svr := &RPCServer{}
	svr.Accept(l)
	return svr
}

func (s *RPCServer) Accept(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go s.ServeConn(conn)
	}
}

func (s *RPCServer) ServeConn(conn net.Conn) {
	defer conn.Close()
	for {
		reqbytes := make([]byte,1024)
		_, err := conn.Read(reqbytes)
		if err != nil {
			continue
		}
		req := &pt.Request{}
		err = proto.Unmarshal(reqbytes, req)
		if err != nil {
			log.Print(err)
			return
		}
		err = s.handleReq(req, conn)
		if err != nil {
			log.Print(err)
			return
		}
	}
}

func (s *RPCServer) handleReq(req *pt.Request, conn net.Conn) error {
	groupname, key := req.GetGroup(), req.GetKey()
	fmt.Println(groupname, key)

	g := lincache.GetGroup(groupname)
	v, err := g.Get(key)
	if err != nil {
		return err
	}
	resp := &pt.Response{Value: v.ByteSlice()}
	respbytes, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = conn.Write(respbytes)
	return err
}
