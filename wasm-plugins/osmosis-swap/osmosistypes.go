package main

type OsmosisTransaction struct {
	Nonce     string              `json:"nonce"`
	Raw       string              `json:"raw"`
	Code      uint32              `json:"code"`
	TxID      string              `json:"tx_id"`
	Signature string              `json:"signature"`
	Tx        Transaction         `json:"tx"`
	TxResult  TransactionResult   `json:"tx_result"`
	Metadata  map[string]IBCTrace `json:"metadata"`
}

type Transaction struct {
	AuthInfo   AuthInfo `json:"auth_info"`
	Body       Body     `json:"body"`
	Signatures []string `json:"signatures"`
}

type AuthInfo struct {
	SignerInfos []SignerInfo `json:"signer_infos"`
	Fee         Fee          `json:"fee"`
}

type SignerInfo struct {
	PublicKey  PublicKeyInfo `json:"public_key"`
	ModeInfo   ModeInfo      `json:"mode_info"`
	Sequence   string        `json:"sequence"`
	Signature  string        `json:"signature,omitempty"`
	PubKeyAlgo string        `json:"pub_key_algo,omitempty"`
	Address    string        `json:"address,omitempty"`
}

type PublicKeyInfo struct {
	Type string `json:"@type"`
	Key  string `json:"key"`
}

type ModeInfo struct {
	Single struct {
		Mode string `json:"mode"`
	} `json:"single"`
}

type Fee struct {
	Amount   []Amount `json:"amount"`
	GasLimit string   `json:"gas_limit"`
	GasPrice string   `json:"gas_price,omitempty"`
}

type Amount struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type Body struct {
	Memo                  string           `json:"memo"`
	Messages              []OsmosisMessage `json:"messages"`
	ExtensionOptions      []interface{}    `json:"extension_options"`
	NonCriticalExtensions []interface{}    `json:"non_critical_extension_options"`
	TimeoutHeight         string           `json:"timeout_height"`
}

type OsmosisMessage struct {
	Type   string `json:"@type"`
	Sender string `json:"sender"`
	Routes []struct {
		PoolID        string  `json:"pool_id"`
		TokenInDenom  *string `json:"token_in_denom,omitempty"`
		TokenOutDenom *string `json:"token_out_denom,omitempty"`
	} `json:"routes"`
	TokenIn *struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	} `json:"token_in,omitempty"`
	TokenInMaxAmount *string `json:"token_in_max_amount,omitempty"`
	TokenOut         *struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	} `json:"token_out,omitempty"`
	TokenOutMinAmount *string `json:"token_out_min_amount,omitempty"`
}

type TransactionResult struct {
	Index  int    `json:"index"`
	Tx     string `json:"tx"`
	Result struct {
		Code   uint32  `json:"code"`
		Data   string  `json:"data"`
		Log    string  `json:"log"`
		Events []Event `json:"events"`
	} `json:"result"`
}

type Event struct {
	Type       string      `json:"type"`
	Attributes []Attribute `json:"attributes"`
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Index bool   `json:"index"`
}

type Metadata struct {
	IBCTrace map[string]IBCTrace `json:"ibc_trace"`
}

type IBCTrace struct {
	Path      string `json:"path"`
	BaseDenom string `json:"base_denom"`
}

type FilteredMessage struct {
	Address           string `json:"address"`
	TxHash            string `json:"txhash"`
	Code              uint32 `json:"code"`
	TokenOutDenom     string `json:"token_out_denom"`
	TokenOutDenomName string `json:"token_out_denom_name"`
	TokenOutAmount    string `json:"token_out_amount"`
	TokenInDenom      string `json:"token_in_denom"`
	TokenInDenomName  string `json:"token_in_denom_name"`
	TokenInAmount     string `json:"token_in_amount"`
	Timestamp         int    `json:"timestamp"`
}
