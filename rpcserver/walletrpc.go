package rpcserver

import "github.com/elnosh/btcw/wallet"

type WalletRPC struct {
	wallet *wallet.Wallet
}

func (w *WalletRPC) GetBalance(args struct{}, reply *int64) error {
	*reply = int64(w.wallet.GetBalance())
	return nil
}

func (w *WalletRPC) GetNewAddress(args struct{}, reply *string) error {
	address, err := w.wallet.GetNewAddress()
	if err != nil {
		return err
	}

	*reply = address
	return nil
}
