package wallet

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/libsv/go-bn/zmq"
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
	defaultFee = btcutil.Amount(2)
)

var (
	ErrZMQNotEnabled = errors.New("ZeroMQ is not enabled")
)

type BtcdClient struct {
	client *rpcclient.Client
}

func SetupBtcdClient(wallet *Wallet, net *chaincfg.Params, rpcuser, rpcpass string) (*BtcdClient, error) {
	port := "18334"
	if net != &chaincfg.TestNet3Params {
		port = "18556"
	}

	// notification handler for when new block is added to the chain
	ntfnHandlers := rpcclient.NotificationHandlers{
		OnFilteredBlockConnected: func(height int32, header *wire.BlockHeader, txs []*btcutil.Tx) {
			blockhash := header.BlockHash().String()
			wallet.LogInfo("received new block with id: %s", blockhash)
			go wallet.scanBlockTxs(blockhash, txs)
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

	go func() {
		// scan blocks added to the blockchain while wallet server was not up
		wallet.scanMissingBlocks()

		// setup btcd notifications for when new block is added
		client.NotifyBlocks()
	}()

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

func SetupBitcoinCoreClient(wallet *Wallet, net *chaincfg.Params, rpcuser, rpcpass string) (*BitcoinCoreClient, error) {
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

	go func() {
		// scan blocks added to the blockchain while wallet server was not up
		wallet.scanMissingBlocks()

		// setup ZeroMQ notifications for new blocks added
		err = subscribeZeroMQNotifications(wallet, coreClient)
		// if err with ZeroMQ notifcations, sync manually
		if err != nil {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				errChan := make(chan error)
				go scanForNewBlocks(ctx, wallet, errChan)
				err = <-errChan
				if err != nil {
					wallet.LogError("error scanning blockchain: %v", err)
				}
			}()
		}
	}()

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

func subscribeZeroMQNotifications(wallet *Wallet, core *BitcoinCoreClient) error {
	zmqNotifications, err := core.client.GetZmqNotifications()
	if err != nil || len(zmqNotifications) == 0 {
		return ErrZMQNotEnabled
	}

	for _, notification := range zmqNotifications {
		addr := notification.Address.String()

		z := zmq.NewNodeMQ(zmq.WithHost(addr))
		if err := z.SubscribeHashBlock(func(_ context.Context, hashStr string) {
			wallet.LogInfo("received new block with id: %s", hashStr)
			blockhash, err := chainhash.NewHashFromStr(hashStr)
			if err != nil {
				wallet.LogError("error decoding hash string: %v", err)
			}
			go wallet.scanBlock(blockhash)
		}); err != nil {
			wallet.LogError("error with ZeroMQ notifications: %v", err)
		}

		go func() {
			fmt.Println(z.Connect())
		}()

	}

	return nil
}
