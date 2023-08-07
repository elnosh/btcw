package rpcserver

import "github.com/elnosh/btcw/wallet"

type WalletRPC struct {
	wallet *wallet.Wallet
}

func (w *WalletRPC) GetBalance(args struct{}, reply *int64) error {
	*reply = w.wallet.GetBalance()
	return nil
}
