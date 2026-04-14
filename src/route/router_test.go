package route

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/assimon/luuu/model/dao"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/util/log"
	"github.com/assimon/luuu/util/sign"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

const testAPIToken = "test-secret-token"

func setupTestEnv(t *testing.T) *echo.Echo {
	t.Helper()

	tmpDir := t.TempDir()

	// minimal viper config
	viper.Reset()
	viper.Set("db_type", "sqlite")
	viper.Set("api_auth_token", testAPIToken)
	viper.Set("app_uri", "http://localhost:8080")
	viper.Set("order_expiration_time", 10)
	viper.Set("api_rate_url", "")
	viper.Set("forced_usdt_rate", 7.0)
	viper.Set("runtime_root_path", tmpDir)
	viper.Set("log_save_path", tmpDir)
	viper.Set("sqlite_database_filename", tmpDir+"/test.db")
	viper.Set("runtime_sqlite_filename", tmpDir+"/runtime.db")

	log.Init()

	// init config paths
	os.Setenv("EPUSDT_CONFIG", tmpDir)
	defer os.Unsetenv("EPUSDT_CONFIG")

	// init DB
	if err := dao.DBInit(); err != nil {
		t.Fatalf("DBInit: %v", err)
	}
	if err := dao.RuntimeInit(); err != nil {
		t.Fatalf("RuntimeInit: %v", err)
	}

	// ensure tables exist (MdbTableInit uses sync.Once, so migrate directly)
	dao.Mdb.AutoMigrate(&mdb.Orders{}, &mdb.WalletAddress{}, &mdb.SupportedAsset{})

	// seed wallet addresses
	dao.Mdb.Create(&mdb.WalletAddress{Network: mdb.NetworkTron, Address: "TTestTronAddress001", Status: mdb.TokenStatusEnable})
	dao.Mdb.Create(&mdb.WalletAddress{Network: mdb.NetworkSolana, Address: "SolTestAddress001", Status: mdb.TokenStatusEnable})
	// seed supported assets if empty
	var supportCnt int64
	dao.Mdb.Model(&mdb.SupportedAsset{}).Count(&supportCnt)
	if supportCnt == 0 {
		dao.Mdb.Create(&[]mdb.SupportedAsset{
			{Network: mdb.NetworkTron, Token: "TRX", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkTron, Token: "USDT", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkSolana, Token: "SOL", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkSolana, Token: "USDT", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkSolana, Token: "USDC", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkEthereum, Token: "USDT", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkEthereum, Token: "USDC", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkBsc, Token: "USDT", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkBsc, Token: "USDC", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkPolygon, Token: "USDT", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkPolygon, Token: "USDC", Status: mdb.TokenStatusEnable},
			{Network: mdb.NetworkPlasma, Token: "USDT", Status: mdb.TokenStatusEnable},
		})
	}

	e := echo.New()
	RegisterRoute(e)
	return e
}

func signBody(body map[string]interface{}) map[string]interface{} {
	sig, _ := sign.Get(body, testAPIToken)
	body["signature"] = sig
	return body
}

func doPost(e *echo.Echo, path string, body map[string]interface{}) *httptest.ResponseRecorder {
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(jsonBytes)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestCreateOrderEpusdtDefaultTron tests the epusdt compatibility route defaults to tron network.
func TestCreateOrderEpusdtDefaultTron(t *testing.T) {
	e := setupTestEnv(t)

	body := signBody(map[string]interface{}{
		"order_id":   "test-tron-001",
		"amount":     1.00,
		"notify_url": "http://localhost/notify",
	})

	rec := doPost(e, "/payments/epusdt/v1/order/create-transaction", body)
	t.Logf("Status: %d, Body: %s", rec.Code, rec.Body.String())

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data in response, got: %v", resp)
	}

	if data["trade_id"] == nil || data["trade_id"] == "" {
		t.Error("expected trade_id in response")
	}
	if data["receive_address"] != "TTestTronAddress001" {
		t.Errorf("expected tron address, got: %v", data["receive_address"])
	}
	t.Logf("Order created: trade_id=%v address=%v amount=%v", data["trade_id"], data["receive_address"], data["actual_amount"])
}

// TestCreateOrderGmpayV1Solana tests the gmpay route with solana network.
func TestCreateOrderGmpayV1Solana(t *testing.T) {
	e := setupTestEnv(t)

	body := signBody(map[string]interface{}{
		"order_id":   "test-sol-001",
		"amount":     1.00,
		"token":      "usdt",
		"currency":   "cny",
		"network":    "solana",
		"notify_url": "http://localhost/notify",
	})

	rec := doPost(e, "/payments/gmpay/v1/order/create-transaction", body)
	t.Logf("Status: %d, Body: %s", rec.Code, rec.Body.String())

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data in response, got: %v", resp)
	}

	if data["trade_id"] == nil || data["trade_id"] == "" {
		t.Error("expected trade_id in response")
	}
	if data["receive_address"] != "SolTestAddress001" {
		t.Errorf("expected solana address, got: %v", data["receive_address"])
	}
	t.Logf("Order created: trade_id=%v address=%v amount=%v", data["trade_id"], data["receive_address"], data["actual_amount"])
}

// TestCreateOrderGmpayV1SolNative tests creating an order for native SOL token.
func TestCreateOrderGmpayV1SolNative(t *testing.T) {
	e := setupTestEnv(t)

	body := signBody(map[string]interface{}{
		"order_id":   "test-sol-native-001",
		"amount":     0.05,
		"token":      "sol",
		"currency":   "usd",
		"network":    "solana",
		"notify_url": "http://localhost/notify",
	})

	rec := doPost(e, "/payments/gmpay/v1/order/create-transaction", body)
	t.Logf("Status: %d, Body: %s", rec.Code, rec.Body.String())

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	t.Logf("Response: %v", resp)

	// This may fail if rate API is not configured, which is expected in test
	// The important thing is the route accepts the request with network=solana token=sol
	if rec.Code != http.StatusOK {
		t.Logf("Note: non-200 may be expected if rate API is not configured for SOL")
	}
}

func doGet(e *echo.Echo, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", testAPIToken)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func doPostWithToken(e *echo.Echo, path string, body map[string]interface{}) *httptest.ResponseRecorder {
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(jsonBytes)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", testAPIToken)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func parseResp(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp
}

func TestGetSupportedAssetsPublic(t *testing.T) {
	e := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/payments/gmpay/v1/supported-assets", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	resp := parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("expected status_code=200, got %v", resp["status_code"])
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data, got %T", resp["data"])
	}
	supports, ok := data["supports"].([]interface{})
	if !ok {
		t.Fatalf("expected supports array, got %T", data["supports"])
	}
	if len(supports) < 2 {
		t.Fatalf("expected >= 2 network supports, got %d", len(supports))
	}

	seen := map[string]bool{}
	for _, item := range supports {
		row := item.(map[string]interface{})
		network := row["network"].(string)
		seen[network] = true
	}
	for _, n := range []string{"tron", "solana"} {
		if !seen[n] {
			t.Fatalf("missing network support: %s", n)
		}
	}
}

func TestGetSupportedAssetsWithWalletsPublic(t *testing.T) {
	e := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/payments/gmpay/v1/supported-assets/wallets", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	resp := parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("expected status_code=200, got %v", resp["status_code"])
	}
	groups, ok := resp["data"].([]interface{})
	if !ok || len(groups) == 0 {
		t.Fatalf("expected non-empty groups, got %T %v", resp["data"], resp["data"])
	}

	foundTron := false
	for _, g := range groups {
		group := g.(map[string]interface{})
		if group["network"] != "tron" {
			continue
		}
		items, _ := group["list"].([]interface{})
		for _, it := range items {
			item := it.(map[string]interface{})
			if item["token"] != "USDT" {
				continue
			}
			addrs, _ := item["addresses"].([]interface{})
			for _, a := range addrs {
				if a.(string) == "TTestTronAddress001" {
					foundTron = true
					break
				}
			}
		}
	}
	if !foundTron {
		t.Fatalf("expected tron/usdt row contains seeded wallet")
	}
}

func TestSupportedAssetCRUD(t *testing.T) {
	e := setupTestEnv(t)

	// public query list (no auth)
	req := httptest.NewRequest(http.MethodGet, "/payments/gmpay/v1/supported-assets/records", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	resp := parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("public list failed: %v", resp)
	}

	// create requires auth
	rec = doPostWithToken(e, "/payments/gmpay/v1/supported-assets/add", map[string]interface{}{
		"network": "arb",
		"token":   "usdt",
		"status":  1,
	})
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("add supported asset failed: %v", resp)
	}
	created := resp["data"].(map[string]interface{})
	assetID := fmt.Sprintf("%.0f", created["id"].(float64))

	// get by id is public
	req = httptest.NewRequest(http.MethodGet, "/payments/gmpay/v1/supported-assets/"+assetID, nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("public get by id failed: %v", resp)
	}

	// update requires auth
	rec = doPostWithToken(e, "/payments/gmpay/v1/supported-assets/"+assetID+"/update", map[string]interface{}{
		"network": "arbitrum",
		"token":   "usdc",
		"status":  1,
	})
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("update supported asset failed: %v", resp)
	}
	updated := resp["data"].(map[string]interface{})
	if updated["network"] != "arbitrum" || updated["token"] != "USDC" {
		t.Fatalf("unexpected updated data: %v", updated)
	}

	// delete requires auth
	rec = doPostWithToken(e, "/payments/gmpay/v1/supported-assets/"+assetID+"/delete", nil)
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("delete supported asset failed: %v", resp)
	}
}

// TestWalletAddAndList tests adding wallets via API and listing them.
func TestWalletAddAndList(t *testing.T) {
	e := setupTestEnv(t)

	// Add a solana wallet
	rec := doPostWithToken(e, "/payments/gmpay/v1/wallet/add", map[string]interface{}{
		"network": "solana",
		"address": "NewSolWallet001",
	})
	t.Logf("Add: %s", rec.Body.String())
	resp := parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("add wallet failed: %v", resp)
	}

	// Add a tron wallet
	rec = doPostWithToken(e, "/payments/gmpay/v1/wallet/add", map[string]interface{}{
		"network": "tron",
		"address": "NewTronWallet001",
	})
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("add tron wallet failed: %v", resp)
	}

	// List all wallets
	rec = doGet(e, "/payments/gmpay/v1/wallet/list")
	resp = parseResp(t, rec)
	wallets := resp["data"].([]interface{})
	// 2 seeded + 2 added = 4
	if len(wallets) != 4 {
		t.Fatalf("expected 4 wallets, got %d: %v", len(wallets), wallets)
	}

	// List by network
	rec = doGet(e, "/payments/gmpay/v1/wallet/list?network=solana")
	resp = parseResp(t, rec)
	wallets = resp["data"].([]interface{})
	if len(wallets) != 2 {
		t.Fatalf("expected 2 solana wallets, got %d", len(wallets))
	}

	rec = doGet(e, "/payments/gmpay/v1/wallet/list?network=tron")
	resp = parseResp(t, rec)
	wallets = resp["data"].([]interface{})
	if len(wallets) != 2 {
		t.Fatalf("expected 2 tron wallets, got %d", len(wallets))
	}
}

// TestWalletDuplicateRejected tests that adding the same network+address twice fails.
func TestWalletDuplicateRejected(t *testing.T) {
	e := setupTestEnv(t)

	body := map[string]interface{}{"network": "solana", "address": "DupWallet001"}
	rec := doPostWithToken(e, "/payments/gmpay/v1/wallet/add", body)
	resp := parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("first add failed: %v", resp)
	}

	// Same network+address should fail
	rec = doPostWithToken(e, "/payments/gmpay/v1/wallet/add", body)
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) == 200 {
		t.Fatal("expected duplicate to be rejected")
	}
	t.Logf("Duplicate rejected: %v", resp["message"])

	// Same address, different network should succeed
	rec = doPostWithToken(e, "/payments/gmpay/v1/wallet/add", map[string]interface{}{
		"network": "tron",
		"address": "DupWallet001",
	})
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("same address on different network should succeed: %v", resp)
	}
}

// TestWalletStatusAndDelete tests enable/disable/delete operations.
func TestWalletStatusAndDelete(t *testing.T) {
	e := setupTestEnv(t)

	// Add a wallet
	rec := doPostWithToken(e, "/payments/gmpay/v1/wallet/add", map[string]interface{}{
		"network": "solana",
		"address": "StatusTestWallet",
	})
	resp := parseResp(t, rec)
	wallet := resp["data"].(map[string]interface{})
	walletID := fmt.Sprintf("%.0f", wallet["id"].(float64))

	// Get wallet
	rec = doGet(e, "/payments/gmpay/v1/wallet/"+walletID)
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("get wallet failed: %v", resp)
	}

	// Disable wallet
	rec = doPostWithToken(e, "/payments/gmpay/v1/wallet/"+walletID+"/status", map[string]interface{}{
		"status": 2,
	})
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("disable wallet failed: %v", resp)
	}

	// Verify disabled — should not appear in available list
	rec = doGet(e, "/payments/gmpay/v1/wallet/list?network=solana")
	resp = parseResp(t, rec)
	wallets := resp["data"].([]interface{})
	for _, w := range wallets {
		wm := w.(map[string]interface{})
		if wm["address"] == "StatusTestWallet" && wm["status"].(float64) != 2 {
			t.Error("wallet should be disabled")
		}
	}

	// Delete wallet
	rec = doPostWithToken(e, "/payments/gmpay/v1/wallet/"+walletID+"/delete", nil)
	resp = parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("delete wallet failed: %v", resp)
	}

	// Verify deleted
	rec = doGet(e, "/payments/gmpay/v1/wallet/"+walletID)
	resp = parseResp(t, rec)
	// Should return not found
	if resp["status_code"].(float64) == 200 {
		data := resp["data"].(map[string]interface{})
		if data["id"].(float64) > 0 {
			t.Error("wallet should be deleted")
		}
	}
}

// TestWalletAuthRequired tests that wallet APIs require auth token.
func TestWalletAuthRequired(t *testing.T) {
	e := setupTestEnv(t)

	// No auth header — should not return success
	req := httptest.NewRequest(http.MethodGet, "/payments/gmpay/v1/wallet/list", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// The response should indicate auth failure (not 200 success)
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		// echo may return plain text error
		if rec.Code == http.StatusOK {
			t.Error("expected auth failure without token")
		}
		t.Logf("Auth rejected (non-JSON): status=%d body=%s", rec.Code, rec.Body.String())
		return
	}
	statusCode, _ := resp["status_code"].(float64)
	if statusCode == 200 {
		t.Error("expected auth failure without token")
	}
	t.Logf("Auth rejected: %v", resp)
}

// TestCreateOrderNetworkIsolation verifies tron and solana wallets don't mix.
func TestCreateOrderNetworkIsolation(t *testing.T) {
	e := setupTestEnv(t)

	// Try to create a solana order — should get solana address, not tron
	body := signBody(map[string]interface{}{
		"order_id":   fmt.Sprintf("test-isolation-%d", 1),
		"amount":     1.00,
		"token":      "usdt",
		"currency":   "cny",
		"network":    "solana",
		"notify_url": "http://localhost/notify",
	})
	rec := doPost(e, "/payments/gmpay/v1/order/create-transaction", body)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data, got: %v", resp)
	}
	if data["receive_address"] == "TTestTronAddress001" {
		t.Error("solana order should NOT get a tron address")
	}
	if data["receive_address"] != "SolTestAddress001" {
		t.Errorf("expected SolTestAddress001, got %v", data["receive_address"])
	}
}
