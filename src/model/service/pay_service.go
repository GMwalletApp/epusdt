package service

import (
	"errors"
	"strings"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/model/response"
)

var ErrOrder = errors.New("不存在待支付订单或已过期")

var checkoutPaymentOptionCatalog = []response.PaymentOption{
	{Token: "USDT", Network: mdb.NetworkTron},
	{Token: "TRX", Network: mdb.NetworkTron},
	{Token: "USDT", Network: mdb.NetworkSolana},
	{Token: "USDC", Network: mdb.NetworkSolana},
	{Token: "USDT", Network: mdb.NetworkEthereum},
	{Token: "USDC", Network: mdb.NetworkEthereum},
}

func buildCheckoutPaymentOptions(order *mdb.Orders) ([]response.PaymentOption, error) {
	wallets, err := data.GetAvailableWalletAddress()
	if err != nil {
		return nil, err
	}

	enabledNetworks := make(map[string]struct{}, len(wallets))
	for _, wallet := range wallets {
		enabledNetworks[strings.ToLower(strings.TrimSpace(wallet.Network))] = struct{}{}
	}

	options := make([]response.PaymentOption, 0, len(checkoutPaymentOptionCatalog))
	seen := make(map[string]struct{}, len(checkoutPaymentOptionCatalog)+1)
	addOption := func(token string, network string) {
		token = strings.ToUpper(strings.TrimSpace(token))
		network = strings.ToLower(strings.TrimSpace(network))
		if token == "" || network == "" {
			return
		}
		key := network + ":" + token
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		options = append(options, response.PaymentOption{
			Token:   token,
			Network: network,
		})
	}

	if order != nil {
		addOption(order.Token, order.Network)
	}
	for _, option := range checkoutPaymentOptionCatalog {
		if _, ok := enabledNetworks[strings.ToLower(option.Network)]; !ok {
			continue
		}
		addOption(option.Token, option.Network)
	}
	return options, nil
}

// GetCheckoutCounterByTradeId returns checkout info for a pending order.
func GetCheckoutCounterByTradeId(tradeId string) (*response.CheckoutCounterResponse, error) {
	orderInfo, err := data.GetOrderInfoByTradeId(tradeId)
	if err != nil {
		return nil, err
	}
	if orderInfo.ID <= 0 || orderInfo.Status != mdb.StatusWaitPay {
		return nil, ErrOrder
	}

	resp := &response.CheckoutCounterResponse{
		TradeId:        orderInfo.TradeId,
		Amount:         orderInfo.Amount,
		ActualAmount:   orderInfo.ActualAmount,
		Token:          orderInfo.Token,
		Currency:       orderInfo.Currency,
		ReceiveAddress: orderInfo.ReceiveAddress,
		Network:        orderInfo.Network,
		ExpirationTime: orderInfo.CreatedAt.AddMinutes(config.GetOrderExpirationTime()).TimestampMilli(),
		RedirectUrl:    orderInfo.RedirectUrl,
		CreatedAt:      orderInfo.CreatedAt.TimestampMilli(),
		IsSelected:     orderInfo.IsSelected,
	}
	resp.PaymentOptions, err = buildCheckoutPaymentOptions(orderInfo)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
