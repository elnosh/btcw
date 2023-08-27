package wallet

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

type NodeClient interface {
	GetBlockCount() (int64, error)
	GetBlockHash(int64) (*chainhash.Hash, error)
	GetBlockVerboseTx(*chainhash.Hash) (*btcjson.GetBlockVerboseTxResult, error)
	SendRawTransaction(*wire.MsgTx, bool) (*chainhash.Hash, error)
	EstimateFee(int64) float64
}

var (
	defaultFee float64 = 0.00005
)

type BtcdClient struct {
	client *rpcclient.Client
}

func NewBtcdClient(testnet bool, rpcuser, rpcpass string) (*BtcdClient, error) {
	port := "18334"
	if !testnet {
		port = "18556"
	}

	certHomeDir := btcutil.AppDataDir("btcd", false)
	certs, err := os.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %v", err)
	}
	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:" + port,
		User:         rpcuser,
		Pass:         rpcpass,
		Certificates: certs,
		HTTPPostMode: true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, fmt.Errorf("rpcclient.New: %v", err)
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

func (btcd *BtcdClient) EstimateFee(numBlocks int64) float64 {
	fee, err := btcd.client.EstimateFee(numBlocks)
	if err != nil {
		return defaultFee
	}
	return fee
}

type BitcoinCoreClient struct {
	client *rpcclient.Client
}

func NewBitcoinCoreClient(testnet bool, rpcuser, rpcpass string) (*BitcoinCoreClient, error) {
	port := "18332"
	if !testnet {
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

func (core *BitcoinCoreClient) EstimateFee(numBlocks int64) float64 {
	feeRes, err := core.client.EstimateSmartFee(numBlocks, &btcjson.EstimateModeConservative)
	if err != nil {
		return defaultFee
	}
	return *feeRes.FeeRate
}
