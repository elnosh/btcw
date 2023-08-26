package wallet

import (
	"github.com/btcsuite/btcd/btcjson"
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
