package comm

import (
	"sort"
	"strconv"

	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/response"
	"github.com/assimon/luuu/util/constant"
	"github.com/labstack/echo/v4"
)

type addSupportedAssetRequest struct {
	Network string `json:"network" validate:"required"`
	Token   string `json:"token" validate:"required"`
	Status  int64  `json:"status" validate:"required|in:1,2"`
}

type updateSupportedAssetRequest struct {
	Network string `json:"network" validate:"required"`
	Token   string `json:"token" validate:"required"`
	Status  int64  `json:"status" validate:"required|in:1,2"`
}

// GetSupportedAssets 对外公开可用链与 token 列表（无需鉴权，仅返回已启用项）。
func (c *BaseCommController) GetSupportedAssets(ctx echo.Context) error {
	list, err := data.ListEnabledSupportedAssets()
	if err != nil {
		return c.FailJson(ctx, err)
	}

	grouped := make(map[string][]string)
	for _, item := range list {
		grouped[item.Network] = append(grouped[item.Network], item.Token)
	}

	networks := make([]string, 0, len(grouped))
	for network := range grouped {
		networks = append(networks, network)
	}
	sort.Strings(networks)

	supports := make([]response.NetworkTokenSupport, 0, len(networks))
	for _, network := range networks {
		tokens := grouped[network]
		sort.Strings(tokens)
		supports = append(supports, response.NetworkTokenSupport{
			Network: network,
			Tokens:  tokens,
		})
	}

	return c.SucJson(ctx, response.SupportedAssetsResponse{Supports: supports})
}

// GetSupportedAssetsWithWallets 对外公开支持的链+token+钱包地址（地址从 wallet_address 读取，仅启用钱包）。
func (c *BaseCommController) GetSupportedAssetsWithWallets(ctx echo.Context) error {
	assets, err := data.ListEnabledSupportedAssets()
	if err != nil {
		return c.FailJson(ctx, err)
	}
	wallets, err := data.GetAvailableWalletAddress()
	if err != nil {
		return c.FailJson(ctx, err)
	}

	addrMap := make(map[string][]string)
	for _, w := range wallets {
		addrMap[w.Network] = append(addrMap[w.Network], w.Address)
	}
	for network := range addrMap {
		sort.Strings(addrMap[network])
	}

	flat := make([]response.NetworkTokenAddressSupport, 0, len(assets))
	for _, a := range assets {
		flat = append(flat, response.NetworkTokenAddressSupport{
			Network:   a.Network,
			Token:     a.Token,
			Addresses: addrMap[a.Network],
		})
	}

	groupMap := make(map[string][]response.TokenAddressItem)
	for _, row := range flat {
		groupMap[row.Network] = append(groupMap[row.Network], response.TokenAddressItem{
			Token:     row.Token,
			Addresses: row.Addresses,
		})
	}

	networks := make([]string, 0, len(groupMap))
	for network := range groupMap {
		networks = append(networks, network)
	}
	sort.Strings(networks)

	groups := make([]response.NetworkTokenAddressGroup, 0, len(networks))
	for _, network := range networks {
		items := groupMap[network]
		sort.Slice(items, func(i, j int) bool {
			return items[i].Token < items[j].Token
		})
		groups = append(groups, response.NetworkTokenAddressGroup{
			Network: network,
			List:    items,
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Network == groups[j].Network {
			return len(groups[i].List) < len(groups[j].List)
		}
		return groups[i].Network < groups[j].Network
	})
	return c.SucJson(ctx, groups)
}

// ListSupportedAssetRecords 查询支持项明细（无需鉴权）。
func (c *BaseCommController) ListSupportedAssetRecords(ctx echo.Context) error {
	network := ctx.QueryParam("network")
	list, err := data.ListSupportedAssets(network)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, list)
}

// GetSupportedAsset 查询单条支持项（无需鉴权）。
func (c *BaseCommController) GetSupportedAsset(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	asset, err := data.GetSupportedAssetByID(id)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	if asset.ID <= 0 {
		return c.FailJson(ctx, constant.SupportedAssetNotFound)
	}
	return c.SucJson(ctx, asset)
}

// AddSupportedAsset 新增支持项（鉴权）。
func (c *BaseCommController) AddSupportedAsset(ctx echo.Context) error {
	req := new(addSupportedAssetRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	if err := c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}
	asset, err := data.AddSupportedAsset(req.Network, req.Token, req.Status)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, asset)
}

// UpdateSupportedAsset 修改支持项（鉴权）。
func (c *BaseCommController) UpdateSupportedAsset(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	req := new(updateSupportedAssetRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	if err := c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}
	asset, err := data.UpdateSupportedAsset(id, req.Network, req.Token, req.Status)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, asset)
}

// DeleteSupportedAsset 删除支持项（鉴权）。
func (c *BaseCommController) DeleteSupportedAsset(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	asset, err := data.GetSupportedAssetByID(id)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	if asset.ID <= 0 {
		return c.FailJson(ctx, constant.SupportedAssetNotFound)
	}
	if err := data.DeleteSupportedAssetByID(id); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}
