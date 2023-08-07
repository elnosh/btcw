package rpcserver

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/elnosh/btcw/wallet"
)

func StartRPCServer(wallet *wallet.Wallet) error {
	walletRPC := &WalletRPC{wallet: wallet}
	rpc.Register(walletRPC)

	listener, err := net.Listen("tcp", ":18557")
	if err != nil {
		return err
	}

	fmt.Printf("rpc server listening on: %v\n", listener.Addr())
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go jsonrpc.ServeConn(conn)
	}
}
