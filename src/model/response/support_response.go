package response

type NetworkTokenSupport struct {
	Network string   `json:"network"`
	Tokens  []string `json:"tokens"`
}

type SupportedAssetsResponse struct {
	Supports []NetworkTokenSupport `json:"supports"`
}

type NetworkTokenAddressSupport struct {
	Network   string   `json:"network"`
	Token     string   `json:"token"`
	Addresses []string `json:"addresses"`
}

type TokenAddressItem struct {
	Token     string   `json:"token"`
	Addresses []string `json:"addresses"`
}

type NetworkTokenAddressGroup struct {
	Network string             `json:"network"`
	List    []TokenAddressItem `json:"list"`
}
