package models

import "encoding/json"

// WSArg identifies a single WebSocket channel within a subscribe/unsubscribe
// request or a push message's "arg" field.
//
// Docs: https://www.bitget.com/api-doc/uta/websocket/private/Fast-Fill-Channel
type WSArg struct {
	InstType string `json:"instType"`
	Topic    string `json:"topic"`
	Symbol   string `json:"symbol,omitempty"`
	Coin     string `json:"coin,omitempty"`
}

// WSSubscribeRequest is sent to subscribe or unsubscribe from one or more
// channels: {"op":"subscribe","args":[...]}.
type WSSubscribeRequest struct {
	Op   string  `json:"op"`
	Args []WSArg `json:"args"`
}

// WSLoginArg is a single credential entry within a login request.
type WSLoginArg struct {
	APIKey     string `json:"apiKey"`
	Passphrase string `json:"passphrase"`
	Timestamp  string `json:"timestamp"`
	Sign       string `json:"sign"`
}

// WSLoginRequest authenticates a private WebSocket connection.
//
// Docs: https://www.bitget.com/api-doc/uta/guide
type WSLoginRequest struct {
	Op   string       `json:"op"`
	Args []WSLoginArg `json:"args"`
}

// WSEvent is a control-plane response to a subscribe/unsubscribe/login
// request (as opposed to a channel data push).
type WSEvent struct {
	Event  string `json:"event"`
	Arg    *WSArg `json:"arg,omitempty"`
	Code   string `json:"code,omitempty"`
	Msg    string `json:"msg,omitempty"`
	ConnID string `json:"connId,omitempty"`
}

// WSPush is a generic channel data push envelope; unmarshal Data further
// per-channel (e.g. into []FastFill for the "fast-fill" channel).
type WSPush struct {
	Arg    WSArg           `json:"arg"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
	Ts     int64           `json:"ts"`
}

// FastFill is a single push payload entry from the private "fast-fill"
// channel.
//
// Docs: https://www.bitget.com/api-doc/uta/websocket/private/Fast-Fill-Channel
type FastFill struct {
	Symbol      string `json:"symbol"`
	Category    string `json:"category"`
	OrderID     string `json:"orderId"`
	ClientOid   string `json:"clientOid"`
	ExecID      string `json:"execId"`
	Side        string `json:"side"`
	HoldSide    string `json:"holdSide"`
	ExecPrice   string `json:"execPrice"`
	ExecQty     string `json:"execQty"`
	TradeScope  string `json:"tradeScope"`
	ExecTime    string `json:"execTime"`
	UpdatedTime string `json:"updatedTime"`
}
