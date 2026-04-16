package task

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/assimon/luuu/config"
	tron "github.com/assimon/luuu/crypto"
	"github.com/assimon/luuu/model/service"
	"github.com/assimon/luuu/util/log"
)

const (
	TronAPIBase = "https://api.trongrid.io"

	// USDT 合约地址 (TRC20 主网)
	USDTContractHex = "41a614f803b6fd780986a42c78ec9c7f77e6ded13c" // Base58: TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
	// transfer(address,uint256) 方法签名前4字节
	TransferMethodID = "a9059cbb"

	PollInterval   = 3 * time.Second
	RequestTimeout = 10 * time.Second
)

func HexToTronAddress(hexAddr string) (string, error) {
	hexAddr = strings.TrimPrefix(hexAddr, "0x")
	hexAddr = strings.TrimPrefix(hexAddr, "0X")

	raw, err := hex.DecodeString(hexAddr)
	if err != nil {
		return "", err
	}

	// 确保有 0x41 前缀（TRON 主网地址前缀）
	if len(raw) == 20 {
		raw = append([]byte{0x41}, raw...)
	}
	if len(raw) != 21 {
		return "", fmt.Errorf("地址长度非法: %d bytes", len(raw))
	}

	return tron.EncodeCheck(raw), nil
}

type BlockHeader struct {
	RawData struct {
		Number         int64  `json:"number"`
		Timestamp      int64  `json:"timestamp"`
		WitnessAddress string `json:"witness_address"`
		ParentHash     string `json:"parentHash"`
		Version        int    `json:"version"`
	} `json:"raw_data"`
}

type TriggerSmartContractValue struct {
	OwnerAddress    string `json:"owner_address"`
	ContractAddress string `json:"contract_address"`
	Data            string `json:"data"`
	CallValue       int64  `json:"call_value"`
}

type ContractParam struct {
	TypeURL string          `json:"type_url"`
	Value   json.RawMessage `json:"value"`
}

type Transaction struct {
	TxID    string `json:"txID"`
	RawData struct {
		Contract []struct {
			Type      string        `json:"type"`
			Parameter ContractParam `json:"parameter"`
		} `json:"contract"`
		Timestamp int64 `json:"timestamp"`
		FeeLimit  int64 `json:"fee_limit"`
	} `json:"raw_data"`
	Ret []struct {
		ContractRet string `json:"contractRet"`
	} `json:"ret"`
}

type Block struct {
	BlockID      string        `json:"blockID"`
	BlockHeader  BlockHeader   `json:"block_header"`
	Transactions []Transaction `json:"transactions"`
}

type USDTTransfer struct {
	TxID   string
	From   string
	To     string
	Raw    *big.Int // 原始数值（6 decimals）
	Status string
}

type TRXTransfer struct {
	TxID   string
	From   string
	To     string
	RawSun int64 // 单位: SUN
	Status string
}

type TransferContractValue struct {
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
	Amount       int64  `json:"amount"` // 单位: SUN
}

var httpClient = &http.Client{Timeout: RequestTimeout}

func doPost(url string, body interface{}) ([]byte, error) {
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if k := strings.TrimSpace(config.TRON_GRID_API_KEY); k != "" {
		req.Header.Set("TRON-PRO-API-KEY", k)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	return b, nil
}

func GetNowBlock() (*Block, error) {
	b, err := doPost(TronAPIBase+"/wallet/getnowblock", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var block Block
	return &block, json.Unmarshal(b, &block)
}

func GetBlockByNum(num int64) (*Block, error) {
	b, err := doPost(TronAPIBase+"/wallet/getblockbynum", map[string]interface{}{"num": num})
	if err != nil {
		return nil, err
	}
	var block Block
	return &block, json.Unmarshal(b, &block)
}

func parseUSDTTransfer(tx Transaction) *USDTTransfer {
	if len(tx.RawData.Contract) == 0 {
		return nil
	}
	c := tx.RawData.Contract[0]
	if c.Type != "TriggerSmartContract" {
		return nil
	}

	var val TriggerSmartContractValue
	if err := json.Unmarshal(c.Parameter.Value, &val); err != nil {
		return nil
	}

	// 检查是否是 USDT 合约
	contractHex := strings.ToLower(strings.TrimPrefix(val.ContractAddress, "0x"))
	if contractHex != USDTContractHex {
		return nil
	}

	// 解析 data 字段
	// 格式: [4字节方法ID][32字节 to 地址][32字节 amount]
	data := strings.TrimPrefix(strings.ToLower(val.Data), "0x")
	if len(data) < 8+64+64 {
		return nil
	}
	if data[:8] != TransferMethodID {
		return nil
	}

	// to 地址：后 40 个十六进制字符（20 字节）
	toHex := data[8+24 : 8+64] // 跳过前12字节填充，取后20字节
	toAddr, err := HexToTronAddress(toHex)
	if err != nil {
		return nil
	}

	// amount：后 32 字节大整数
	amountHex := data[8+64 : 8+64+64]
	amountBytes, err := hex.DecodeString(amountHex)
	if err != nil {
		return nil
	}
	amountBig := new(big.Int).SetBytes(amountBytes)

	// from 地址
	fromAddr, err := HexToTronAddress(val.OwnerAddress)
	if err != nil {
		fromAddr = val.OwnerAddress
	}

	// 交易状态
	status := "SUCCESS"
	if len(tx.Ret) > 0 && tx.Ret[0].ContractRet != "" && tx.Ret[0].ContractRet != "SUCCESS" {
		status = tx.Ret[0].ContractRet
	}

	return &USDTTransfer{
		TxID:   tx.TxID,
		From:   fromAddr,
		To:     toAddr,
		Raw:    amountBig,
		Status: status,
	}
}

func parseTRXTransfer(tx Transaction) *TRXTransfer {
	if len(tx.RawData.Contract) == 0 {
		return nil
	}
	c := tx.RawData.Contract[0]
	if c.Type != "TransferContract" {
		return nil
	}

	var val TransferContractValue
	if err := json.Unmarshal(c.Parameter.Value, &val); err != nil {
		return nil
	}
	if val.Amount <= 0 {
		return nil
	}

	fromAddr, err := HexToTronAddress(val.OwnerAddress)
	if err != nil {
		fromAddr = val.OwnerAddress
	}
	toAddr, err := HexToTronAddress(val.ToAddress)
	if err != nil {
		toAddr = val.ToAddress
	}

	status := "SUCCESS"
	if len(tx.Ret) > 0 && tx.Ret[0].ContractRet != "" && tx.Ret[0].ContractRet != "SUCCESS" {
		status = tx.Ret[0].ContractRet
	}

	return &TRXTransfer{
		TxID:   tx.TxID,
		From:   fromAddr,
		To:     toAddr,
		RawSun: val.Amount,
		Status: status,
	}
}

func processBlock(block *Block) {
	blockTsMs := block.BlockHeader.RawData.Timestamp
	for _, tx := range block.Transactions {
		if t := parseUSDTTransfer(tx); t != nil {
			if t.Status != "SUCCESS" {
				continue
			}
			service.TryProcessTronTRC20Transfer(t.To, t.Raw, t.TxID, blockTsMs)
			continue
		}
		if t := parseTRXTransfer(tx); t != nil {
			if t.Status != "SUCCESS" {
				continue
			}
			service.TryProcessTronTRXTransfer(t.To, t.RawSun, t.TxID, blockTsMs)
		}
	}
}

type Scanner struct {
	lastBlock int64
	// 统计
	totalBlocks  int64
	totalUSDTTxs int64
	totalTRXTxs  int64
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Init() error {
	log.Sugar.Infof("[TRON-BLOCK] node=%s", TronAPIBase)
	block, err := GetNowBlock()
	if err != nil {
		return fmt.Errorf("获取初始块失败: %w", err)
	}
	s.lastBlock = block.BlockHeader.RawData.Number
	log.Sugar.Infof("[TRON-BLOCK] start block=%d", s.lastBlock)
	return nil
}

func (s *Scanner) Run() {
	log.Sugar.Info("[TRON-BLOCK] start scanning (USDT TRC20 + TRX)")
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	statTicker := time.NewTicker(60 * time.Second)
	defer statTicker.Stop()

	for {
		select {
		case <-statTicker.C:
			log.Sugar.Infof("[TRON-BLOCK] stats blocks=%d usdt=%d trx=%d", s.totalBlocks, s.totalUSDTTxs, s.totalTRXTxs)
		case <-ticker.C:
			s.poll()
		}
	}
}

func (s *Scanner) poll() {
	latest, err := GetNowBlock()
	if err != nil {
		log.Sugar.Warnf("[TRON-BLOCK] get latest block: %v", err)
		return
	}
	latestNum := latest.BlockHeader.RawData.Number
	if latestNum <= s.lastBlock {
		return
	}

	for num := s.lastBlock + 1; num <= latestNum; num++ {
		var block *Block
		if num == latestNum {
			block = latest
		} else {
			block, err = GetBlockByNum(num)
			if err != nil {
				log.Sugar.Warnf("[TRON-BLOCK] get block %d: %v", num, err)
				continue
			}
			time.Sleep(200 * time.Millisecond)
		}
		processBlock(block)
		s.lastBlock = num
		s.totalBlocks++

		for _, tx := range block.Transactions {
			if parseUSDTTransfer(tx) != nil {
				s.totalUSDTTxs++
			} else if parseTRXTransfer(tx) != nil {
				s.totalTRXTxs++
			}
		}
	}
}

func StartTronBlockScannerListener() {
	scanner := NewScanner()
	if err := scanner.Init(); err != nil {
		log.Sugar.Errorf("[TRON-BLOCK] init: %v", err)
		return
	}
	scanner.Run()
}
