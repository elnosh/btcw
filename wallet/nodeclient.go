package wallet

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

type NodeClient interface {
	GetBlockCount() (int64, error)
	GetBlockHash(int64) (*chainhash.Hash, error)
	GetBlockVerboseTx(*chainhash.Hash) (*btcjson.GetBlockVerboseTxResult, error)
	SendRawTransaction(*wire.MsgTx, bool) (*chainhash.Hash, error)
	EstimateFee(int64) btcutil.Amount
	LoadTxFilter(bool, []btcutil.Address, []wire.OutPoint) error
}

var (
	defaultFee = btcutil.Amount(5)
)

type BtcdClient struct {
	client *rpcclient.Client
}

func NewBtcdClient(wallet *Wallet, net *chaincfg.Params, rpcuser, rpcpass string) (*BtcdClient, error) {
	port := "18334"
	if net != &chaincfg.TestNet3Params {
		port = "18556"
	}

	// notification handler for when new block is added to the chain
	ntfnHandlers := rpcclient.NotificationHandlers{
		OnFilteredBlockConnected: func(height int32, header *wire.BlockHeader, txs []*btcutil.Tx) {
			go wallet.scanBlockTxs(header.BlockHash().String(), txs)
		},
	}

	btcdHomeDir := btcutil.AppDataDir("btcd", false)
	certs, err := os.ReadFile(filepath.Join(btcdHomeDir, "rpc.cert"))
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %v", err)
	}
	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:" + port,
		Endpoint:     "ws",
		User:         rpcuser,
		Pass:         rpcpass,
		Certificates: certs,
	}

	client, err := rpcclient.New(connCfg, &ntfnHandlers)
	if err != nil {
		return nil, fmt.Errorf("rpcclient.New: %v", err)
	}

	if err := client.NotifyBlocks(); err != nil {
		return nil, fmt.Errorf("client.NotifyBlocks: %v", err)
	}

	btcdClient := &BtcdClient{client: client}
	return btcdClient, nil
}

func (btcd *BtcdClient) GetBlockCount() (int64, error) {
	return btcd.client.GetBlockCount()
}

func (btcd *BtcdClient) GetBlockHash(height int64) (*chainhash.Hash, error) {
	return btcd.client.GetBlockHash(height)
}

func (btcd *BtcdClient) GetBlockVerboseTx(hash *chainhash.Hash) (*btcjson.GetBlockVerboseTxResult, error) {
	return btcd.client.GetBlockVerboseTx(hash)
}

func (btcd *BtcdClient) SendRawTransaction(tx *wire.MsgTx, highFees bool) (*chainhash.Hash, error) {
	return btcd.client.SendRawTransaction(tx, highFees)
}

func (btcd *BtcdClient) EstimateFee(numBlocks int64) btcutil.Amount {
	estimateFee, err := btcd.client.EstimateFee(numBlocks)
	if err != nil || estimateFee == 0 {
		return defaultFee
	}
	fee, _ := btcutil.NewAmount(estimateFee)
	return fee
}

// loadTxFilter sends the list of wallet external addresses
// in the loadtxfilter RPC call. This is specific to btcd.
// to be used whenever a new address is generated so that
// btcd notifcations will add the new address to the filter
func (w *Wallet) loadTxFilter() error {
	addrs := make([]btcutil.Address, len(w.addresses))
	i := 0
	for k := range w.addresses {
		addr, _ := btcutil.DecodeAddress(k, w.network)
		addrs[i] = addr
		i++
	}

	if err := w.client.LoadTxFilter(true, addrs, []wire.OutPoint{}); err != nil {
		return fmt.Errorf("client.LoadTxFilter: %v", err)
	}
	return nil
}

func (btcd *BtcdClient) LoadTxFilter(reload bool, addresses []btcutil.Address, outpoints []wire.OutPoint) error {
	return btcd.client.LoadTxFilter(reload, addresses, outpoints)
}

type BitcoinCoreClient struct {
	client *rpcclient.Client
}

func NewBitcoinCoreClient(net *chaincfg.Params, rpcuser, rpcpass string) (*BitcoinCoreClient, error) {
	port := "18332"
	if net != &chaincfg.TestNet3Params {
		port = "18443"
	}

	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:" + port,
		User:         rpcuser,
		Pass:         rpcpass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, fmt.Errorf("rpcclient.New: %v", err)
	}
	coreClient := &BitcoinCoreClient{client: client}
	return coreClient, nil
}

func (core *BitcoinCoreClient) GetBlockCount() (int64, error) {
	return core.client.GetBlockCount()
}

func (core *BitcoinCoreClient) GetBlockHash(height int64) (*chainhash.Hash, error) {
	return core.client.GetBlockHash(height)
}

func (core *BitcoinCoreClient) GetBlockVerboseTx(hash *chainhash.Hash) (*btcjson.GetBlockVerboseTxResult, error) {
	return core.client.GetBlockVerboseTx(hash)
}

func (core *BitcoinCoreClient) SendRawTransaction(tx *wire.MsgTx, highFees bool) (*chainhash.Hash, error) {
	return core.client.SendRawTransaction(tx, highFees)
}

func (core *BitcoinCoreClient) EstimateFee(numBlocks int64) btcutil.Amount {
	feeRes, err := core.client.EstimateSmartFee(numBlocks, &btcjson.EstimateModeConservative)
	if err != nil || *feeRes.FeeRate == 0 {
		return defaultFee
	}
	fee, _ := btcutil.NewAmount(*feeRes.FeeRate)
	return fee
}

func (core *BitcoinCoreClient) LoadTxFilter(reload bool, addresses []btcutil.Address, outpoints []wire.OutPoint) error {
	return nil
}
