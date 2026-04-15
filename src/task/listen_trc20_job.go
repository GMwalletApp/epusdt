package task

import (
	"context"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	tron "github.com/assimon/luuu/crypto"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/model/service"
	"github.com/assimon/luuu/util/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const tronUsdtContractBase58 = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

// TronGrid 的 JSON-RPC 使用 20-byte hex 地址（去掉 Tron 主网 0x41 前缀）。
var tronUsdtTrc20Contract = common.HexToAddress("0xa614f803b6fd780986a42c78ec9c7f77e6ded13c")

type tronRecipientSnapshot struct {
	addrs map[string]struct{}
}

var tronWatchedRecipients atomic.Pointer[tronRecipientSnapshot]

func StartTronTrc20WebSocketListener() {
	log.Sugar.Infof("[TRON-WS] listening contract base58=%s hex=%s", tronUsdtContractBase58, tronUsdtTrc20Contract.Hex())

	wallets, err := data.GetAvailableWalletAddressByNetwork(mdb.NetworkTron)
	if err != nil {
		log.Sugar.Errorf("[TRON-WS] Failed to get wallet addresses: %v", err)
		return
	}
	storeTronRecipientsFromWallets(wallets)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			w, err := data.GetAvailableWalletAddressByNetwork(mdb.NetworkTron)
			if err != nil {
				log.Sugar.Warnf("[TRON-WS] refresh wallet addresses: %v", err)
				continue
			}
			storeTronRecipientsFromWallets(w)
		}
	}()

	wsURL := "wss://api.trongrid.io/jsonrpc"
	query := ethereum.FilterQuery{
		Addresses: []common.Address{tronUsdtTrc20Contract},
		Topics:    [][]common.Hash{},
	}
	runWsLogListener("[TRON-WS]", wsURL, query, func(client *ethclient.Client, vLog types.Log) {
		if len(vLog.Topics) < 3 {
			return
		}
		if vLog.Topics[0].String() != transferEventHash.String() {
			return
		}

		toAddr := tronHexToBase58(common.HexToAddress(vLog.Topics[2].Hex()))
		//if !isWatchedTronRecipient(toAddr) {
		//	return
		//}

		amount := new(big.Int).SetBytes(vLog.Data)
		if amount.Sign() <= 0 {
			return
		}

		log.Sugar.Infof("[TRON-WS] Detected TRC20 transfer to %s amount=%s tx=%s", toAddr, amount.String(), vLog.TxHash.Hex())
		var blockTsMs int64
		header, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(vLog.BlockNumber)))
		if err != nil {
			log.Sugar.Warnf("[TRON-WS] HeaderByNumber block=%d: %v, using local time", vLog.BlockNumber, err)
			blockTsMs = time.Now().UnixMilli()
		} else {
			blockTsMs = int64(header.Time) * 1000
		}

		service.TryProcessTronTRC20Transfer(toAddr, amount, vLog.TxHash.Hex(), blockTsMs)
	})
}

func storeTronRecipientsFromWallets(wallets []mdb.WalletAddress) int {
	m := make(map[string]struct{})
	for _, w := range wallets {
		addr := strings.TrimSpace(w.Address)
		if addr == "" {
			continue
		}
		m[addr] = struct{}{}
	}
	tronWatchedRecipients.Store(&tronRecipientSnapshot{addrs: m})
	return len(m)
}

func isWatchedTronRecipient(to string) bool {
	snap := tronWatchedRecipients.Load()
	if snap == nil || len(snap.addrs) == 0 {
		return false
	}
	_, ok := snap.addrs[strings.TrimSpace(to)]
	return ok
}

func tronHexToBase58(addr common.Address) string {
	raw := append([]byte{tron.PrefixMainnet}, addr.Bytes()...)
	return tron.EncodeCheck(raw)
}
