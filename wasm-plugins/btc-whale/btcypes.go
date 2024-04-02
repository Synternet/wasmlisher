package main

type BitcoinTransaction struct {
	Txid     string    `json:"txid"`
	Hash     string    `json:"hash"`
	Version  int       `json:"version"`
	Size     int       `json:"size"`
	Vsize    int       `json:"vsize"`
	Weight   int       `json:"weight"`
	Locktime int       `json:"locktime"`
	Vin      []BtcVin  `json:"vin"`
	Vout     []BtcVout `json:"vout"`
}

type BtcVin struct {
	Txid      string `json:"txid"`
	Vout      int    `json:"vout"`
	ScriptSig struct {
		Asm string `json:"asm"`
		Hex string `json:"hex"`
	} `json:"scriptSig"`
	Txinwitness []string `json:"txinwitness"`
	Sequence    int64    `json:"sequence"`
}

type BtcVout struct {
	Value        float64 `json:"value"`
	N            int     `json:"n"`
	ScriptPubKey struct {
		Asm     string `json:"asm"`
		Desc    string `json:"desc"`
		Hex     string `json:"hex"`
		Address string `json:"address"`
		Type    string `json:"type"`
	} `json:"scriptPubKey"`
}

type FilteredMessage struct {
	To    string  `json:"to"`
	TxID  string  `json:"txid"`
	Value float64 `json:"value"`
	//	Timestamp int     `json:"timestamp"`
}
