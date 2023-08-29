package rpcserver

import (
	"fmt"
	"log/slog"
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
		return fmt.Errorf("error starting RPC server: %s", err.Error())
	}

	slog.Info("rpc server listening on: " + listener.Addr().String())
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go jsonrpc.ServeConn(conn)
	}
}
