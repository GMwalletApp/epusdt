package service

import (
	"testing"

	"github.com/assimon/luuu/internal/testutil"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/model/response"
)

func paymentOptionSet(options []response.PaymentOption) map[string]struct{} {
	set := make(map[string]struct{}, len(options))
	for _, option := range options {
		set[option.Network+":"+option.Token] = struct{}{}
	}
	return set
}

func TestGetCheckoutCounterByTradeIdFiltersPaymentOptionsByConfiguredNetworks(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if _, err := data.AddWalletAddressWithNetwork(mdb.NetworkTron, "tron_wallet_1"); err != nil {
		t.Fatalf("add tron wallet: %v", err)
	}
	if _, err := data.AddWalletAddressWithNetwork(mdb.NetworkSolana, "sol_wallet_1"); err != nil {
		t.Fatalf("add sol wallet: %v", err)
	}

	order, err := CreateTransaction(newCreateTransactionRequest("checkout-options-1", 1))
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	resp, err := GetCheckoutCounterByTradeId(order.TradeId)
	if err != nil {
		t.Fatalf("get checkout counter: %v", err)
	}

	options := paymentOptionSet(resp.PaymentOptions)
	for _, key := range []string{
		"tron:USDT",
		"tron:TRX",
		"solana:USDT",
		"solana:USDC",
	} {
		if _, ok := options[key]; !ok {
			t.Fatalf("expected payment option %s, got %#v", key, resp.PaymentOptions)
		}
	}
	if _, ok := options["ethereum:USDT"]; ok {
		t.Fatalf("did not expect ethereum options without ethereum wallet, got %#v", resp.PaymentOptions)
	}
}

func TestGetCheckoutCounterByTradeIdDoesNotAdvertiseUnconfiguredNetworks(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if _, err := data.AddWalletAddressWithNetwork(mdb.NetworkTron, "tron_wallet_1"); err != nil {
		t.Fatalf("add tron wallet: %v", err)
	}

	order, err := CreateTransaction(newCreateTransactionRequest("checkout-options-2", 1))
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	resp, err := GetCheckoutCounterByTradeId(order.TradeId)
	if err != nil {
		t.Fatalf("get checkout counter: %v", err)
	}

	options := paymentOptionSet(resp.PaymentOptions)
	if _, ok := options["tron:USDT"]; !ok {
		t.Fatalf("expected current tron option, got %#v", resp.PaymentOptions)
	}
	if _, ok := options["solana:USDT"]; ok {
		t.Fatalf("did not expect solana options without sol wallet, got %#v", resp.PaymentOptions)
	}
	if _, ok := options["ethereum:USDT"]; ok {
		t.Fatalf("did not expect ethereum options without ethereum wallet, got %#v", resp.PaymentOptions)
	}
}
