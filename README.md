# btcw

btcw is a bitcoin testnet wallet. 
Why another bitcoin wallet? So that you could send some coins to it and lose them all! Kidding...
This was purely for educational purposes for me to learn more about bitcoin and see how I would go about building one.

It only has the most basic functionality to generate addresses, build transactions,
sign them, and broadcast to the network. It implements BIP-44 hierarchical deterministic wallets. 


## requirements
* go 1.21
* Bitcoin node ([btcd](https://github.com/btcsuite/btcd) or [bitcoin core](https://github.com/bitcoin/bitcoin))


## build
* have a bitcoin node running in testnet (or regtest/simnet)

* `go build .`

* create wallet
```
./btcw -create
```


* start wallet (by default it will try to connect to a btcd node. If running with bitcoin core, add `-node=core` when starting wallet)
```
./btcw -rpcuser={yourrpcuser} -rpcpass={yourpcpassword}
```

## usage
* `cd cmd/btcw-cli`

* `go build .`

* get new address
```
./btcw-cli getnewaddress
```

* get balance
```
./btcw-cli getbalance
```

* send btc 
```
./btcw-cli sendtoaddress "{address}" amount (in btc)
```