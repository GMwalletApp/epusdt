package service

import (
	"errors"
	"fmt"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/model/request"
	"github.com/assimon/luuu/telegram"
	"github.com/assimon/luuu/util/constant"
	"github.com/assimon/luuu/util/log"
	"github.com/assimon/luuu/util/math"
	"github.com/dromara/carbon/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"math/big"
	"strings"
)

func TryProcessTronTRC20Transfer(toAddr string, rawValue *big.Int, txHash string, blockTsMs int64) {
	defer func() {
		if err := recover(); err != nil {
			log.Sugar.Errorf("[TRC20][%s] TryProcessTronTRC20Transfer panic: %v", toAddr, err)
		}
	}()

	addr := strings.TrimSpace(toAddr)
	if addr == "" || rawValue == nil || rawValue.Sign() <= 0 {
		return
	}

	decimalQuant := decimal.NewFromBigInt(rawValue, 0)
	amount := math.MustParsePrecFloat64(decimalQuant.Div(decimal.NewFromInt(1_000_000)).InexactFloat64(), 2)
	if amount <= 0 {
		return
	}

	tradeID, err := data.GetTradeIdByWalletAddressAndAmountAndToken(mdb.NetworkTron, addr, "USDT", amount)
	if err != nil {
		log.Sugar.Warnf("[TRC20][%s] lock lookup: %v", addr, err)
		return
	}
	if tradeID == "" {
		log.Sugar.Debugf("[TRC20][%s] skip unmatched tx hash=%s amount=%.2f", addr, txHash, amount)
		return
	}

	order, err := data.GetOrderInfoByTradeId(tradeID)
	if err != nil {
		log.Sugar.Warnf("[TRC20][%s] load order: %v", addr, err)
		return
	}
	if blockTsMs > 0 && blockTsMs < order.CreatedAt.TimestampMilli() {
		log.Sugar.Warnf("[TRC20][%s] skip tx %s because block time %d is before order create time %d", addr, txHash, blockTsMs, order.CreatedAt.TimestampMilli())
		return
	}

	req := &request.OrderProcessingRequest{
		ReceiveAddress:     addr,
		Token:              "USDT",
		Network:            mdb.NetworkTron,
		TradeId:            tradeID,
		Amount:             amount,
		BlockTransactionId: txHash,
	}
	err = OrderProcessing(req)
	if err != nil {
		if errors.Is(err, constant.OrderBlockAlreadyProcess) || errors.Is(err, constant.OrderStatusConflict) {
			log.Sugar.Infof("[TRC20][%s] skip resolved transfer trade_id=%s hash=%s err=%v", addr, tradeID, txHash, err)
			return
		}
		log.Sugar.Errorf("[TRC20][%s] OrderProcessing trade_id=%s hash=%s: %v", addr, tradeID, txHash, err)
		return
	}

	sendPaymentNotification(order)
	log.Sugar.Infof("[TRC20][%s] payment processed trade_id=%s hash=%s", addr, tradeID, txHash)
}

func TryProcessTronTRXTransfer(toAddr string, rawSun int64, txHash string, blockTsMs int64) {
	defer func() {
		if err := recover(); err != nil {
			log.Sugar.Errorf("[TRX][%s] TryProcessTronTRXTransfer panic: %v", toAddr, err)
		}
	}()

	addr := strings.TrimSpace(toAddr)
	if addr == "" || rawSun <= 0 {
		return
	}

	decimalQuant := decimal.NewFromInt(rawSun)
	amount := math.MustParsePrecFloat64(decimalQuant.Div(decimal.NewFromInt(1_000_000)).InexactFloat64(), 2)
	if amount <= 0 {
		return
	}

	tradeID, err := data.GetTradeIdByWalletAddressAndAmountAndToken(mdb.NetworkTron, addr, "TRX", amount)
	if err != nil {
		log.Sugar.Warnf("[TRX][%s] lock lookup: %v", addr, err)
		return
	}
	if tradeID == "" {
		log.Sugar.Debugf("[TRX][%s] skip unmatched tx hash=%s amount=%.2f", addr, txHash, amount)
		return
	}

	order, err := data.GetOrderInfoByTradeId(tradeID)
	if err != nil {
		log.Sugar.Warnf("[TRX][%s] load order: %v", addr, err)
		return
	}
	if blockTsMs > 0 && blockTsMs < order.CreatedAt.TimestampMilli() {
		log.Sugar.Warnf("[TRX][%s] skip tx %s because block time %d is before order create time %d", addr, txHash, blockTsMs, order.CreatedAt.TimestampMilli())
		return
	}

	req := &request.OrderProcessingRequest{
		ReceiveAddress:     addr,
		Token:              "TRX",
		Network:            mdb.NetworkTron,
		TradeId:            tradeID,
		Amount:             amount,
		BlockTransactionId: txHash,
	}
	err = OrderProcessing(req)
	if err != nil {
		if errors.Is(err, constant.OrderBlockAlreadyProcess) || errors.Is(err, constant.OrderStatusConflict) {
			log.Sugar.Infof("[TRX][%s] skip resolved transfer trade_id=%s hash=%s err=%v", addr, tradeID, txHash, err)
			return
		}
		log.Sugar.Errorf("[TRX][%s] OrderProcessing trade_id=%s hash=%s: %v", addr, tradeID, txHash, err)
		return
	}

	sendPaymentNotification(order)
	log.Sugar.Infof("[TRX][%s] payment processed trade_id=%s hash=%s", addr, tradeID, txHash)
}

func evmChainLogLabel(chainNetwork string) string {
	switch chainNetwork {
	case mdb.NetworkEthereum:
		return "ETH"
	case mdb.NetworkBsc:
		return "BSC"
	case mdb.NetworkPolygon:
		return "POLYGON"
	case mdb.NetworkPlasma:
		return "PLASMA"
	default:
		return "EVM"
	}
}

func TryProcessEvmERC20Transfer(chainNetwork string, contract common.Address, toAddr common.Address, rawValue *big.Int, txHash string, blockTsMs int64) {
	defer func() {
		if err := recover(); err != nil {
			log.Sugar.Errorf("[%s-WS] TryProcessEvmERC20Transfer panic: %v", evmChainLogLabel(chainNetwork), err)
		}
	}()

	var usdt, usdc common.Address
	var polygonUsdcE common.Address
	switch chainNetwork {
	case mdb.NetworkEthereum:
		usdt = common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
		usdc = common.HexToAddress("0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	case mdb.NetworkBsc:
		usdt = common.HexToAddress("0x55d398326f99059fF775485246999027B3197955")
		usdc = common.HexToAddress("0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d")
	case mdb.NetworkPolygon:
		usdt = common.HexToAddress("0xc2132D05D31c914a87C6611C10748AEb04B58e8F")
		usdc = common.HexToAddress("0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359")
		polygonUsdcE = common.HexToAddress("0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174")
	case mdb.NetworkPlasma:
		// USDT0（官方），6 decimals；链上暂无与 ETH 同级的 Circle USDC 部署，仅匹配 USDT 订单
		usdt = common.HexToAddress("0xB8CE59FC3717ada4C02eaDF9682A9e934F625ebb")
	default:
		return
	}

	var tokenSym string
	switch {
	case contract == usdt:
		tokenSym = "USDT"
	case contract == usdc || (polygonUsdcE != (common.Address{}) && contract == polygonUsdcE):
		tokenSym = "USDC"
	default:
		net := evmChainLogLabel(chainNetwork)
		log.Sugar.Warnf("[%s-WS] skip unsupported contract %s", net, contract.Hex())
		return
	}

	net := evmChainLogLabel(chainNetwork)
	walletAddr := strings.ToLower(toAddr.Hex())
	if rawValue == nil || rawValue.Sign() <= 0 {
		log.Sugar.Infof("[%s-%s][%s] skip non-positive or nil amount", net, tokenSym, walletAddr)
		return
	}
	decimalQuant := decimal.NewFromBigInt(rawValue, 0)
	amount := math.MustParsePrecFloat64(decimalQuant.Div(decimal.NewFromInt(1_000_000)).InexactFloat64(), 2)
	if amount <= 0 {
		log.Sugar.Warnf("[%s-%s][%s] skip non-positive amount %.2f", net, tokenSym, walletAddr, amount)
		return
	}

	log.Sugar.Debugf("[%s-%s][%s] processing transfer hash=%s amount=%.2f", net, tokenSym, walletAddr, txHash, amount)

	tradeID, err := data.GetTradeIdByWalletAddressAndAmountAndToken(chainNetwork, walletAddr, tokenSym, amount)
	if err != nil {
		log.Sugar.Warnf("[%s-%s][%s] lock lookup: %v", net, tokenSym, walletAddr, err)
		return
	}
	if tradeID == "" {
		log.Sugar.Warnf("[%s-%s][%s] skip unmatched tx hash=%s amount=%.2f", net, tokenSym, walletAddr, txHash, amount)
		return
	}

	order, err := data.GetOrderInfoByTradeId(tradeID)
	if err != nil {
		log.Sugar.Warnf("[%s-%s][%s] load order: %v", net, tokenSym, walletAddr, err)
		return
	}
	if strings.ToLower(strings.TrimSpace(order.Network)) != chainNetwork {
		log.Sugar.Warnf("[%s-%s][%s] skip trade_id=%s network=%q", net, tokenSym, walletAddr, tradeID, order.Network)
		return
	}
	if strings.ToUpper(strings.TrimSpace(order.Token)) != tokenSym {
		log.Sugar.Warnf("[%s-%s][%s] skip trade_id=%s token mismatch order=%s", net, tokenSym, walletAddr, tradeID, order.Token)
		return
	}

	req := &request.OrderProcessingRequest{
		ReceiveAddress:     walletAddr,
		Token:              tokenSym,
		Network:            chainNetwork,
		TradeId:            tradeID,
		Amount:             amount,
		BlockTransactionId: txHash,
	}
	err = OrderProcessing(req)
	if err != nil {
		if errors.Is(err, constant.OrderBlockAlreadyProcess) || errors.Is(err, constant.OrderStatusConflict) {
			log.Sugar.Infof("[%s-%s][%s] skip resolved trade_id=%s hash=%s err=%v", net, tokenSym, walletAddr, tradeID, txHash, err)
			return
		}
		log.Sugar.Errorf("[%s-%s][%s] OrderProcessing: %v", net, tokenSym, walletAddr, err)
		return
	}

	sendPaymentNotification(order)
	log.Sugar.Infof("[%s-%s][%s] payment processed trade_id=%s hash=%s", net, tokenSym, walletAddr, tradeID, txHash)
}

func sendPaymentNotification(order *mdb.Orders) {
	msg := fmt.Sprintf(
		"🎉 <b>收款成功通知</b>\n\n"+
			"💰 <b>金额信息</b>\n"+
			"├ 订单金额：<code>%.2f %s</code>\n"+
			"└ 实际到账：<code>%.2f %s</code>\n\n"+
			"📋 <b>订单信息</b>\n"+
			"├ 交易号：<code>%s</code>\n"+
			"├ 订单号：<code>%s</code>\n"+
			"├ 网络：<code>%s</code>\n"+
			"└ 钱包地址：<code>%s</code>\n\n"+
			"⏰ <b>时间信息</b>\n"+
			"├ 创建时间：%s\n"+
			"└ 支付时间：%s",
		order.Amount,
		strings.ToUpper(order.Currency),
		order.ActualAmount,
		strings.ToUpper(order.Token),
		order.TradeId,
		order.OrderId,
		networkDisplay(order.Network),
		order.ReceiveAddress,
		order.CreatedAt.ToDateTimeString(),
		carbon.Now().ToDateTimeString(),
	)
	telegram.SendToBot(msg)
}

func networkDisplay(n string) string {
	switch strings.ToLower(strings.TrimSpace(n)) {
	case mdb.NetworkTron:
		return "Tron"
	case mdb.NetworkSolana:
		return "Solana"
	case mdb.NetworkEthereum:
		return "Ethereum"
	case mdb.NetworkBsc:
		return "BSC"
	case mdb.NetworkPolygon:
		return "Polygon"
	case mdb.NetworkPlasma:
		return "Plasma"
	default:
		if n == "" {
			return "Tron"
		}
		return strings.ToUpper(n)
	}
}
