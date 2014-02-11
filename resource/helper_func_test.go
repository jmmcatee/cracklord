package resource

import (
	"log"
	"net"
	"net/rpc"
)

func startRPCOnce(n string, addr string, q *Queue) net.Listener {
	res := rpc.NewServer()
	res.Register(q)

	listen, err := net.Listen(n, addr)
	if err != nil {
		panic(err.Error())
	}

	go func() {
		conn, err := listen.Accept()
		if err != nil {
			panic(err.Error())
		}

		res.ServeConn(conn)
	}()

	return listen
}
