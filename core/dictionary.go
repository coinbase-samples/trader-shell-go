/*
Copyright 2023-present Coinbase Global, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

var execTypeDescriptions = map[string]string{
	"0": "ExecType_NEW",
	"1": "ExecType_PARTIAL_FILL",
	"2": "ExecType_FILL",
	"3": "ExecType_DONE_FOR_DAY",
	"4": "ExecType_CANCELED",
	"5": "ExecType_REPLACED",
	"6": "ExecType_PENDING_CANCEL",
	"7": "ExecType_STOPPED",
	"8": "ExecType_REJECTED",
	"9": "ExecType_SUSPENDED",
	"A": "ExecType_PENDING_NEW",
	"B": "ExecType_CALCULATED",
	"C": "ExecType_EXPIRED",
	"D": "ExecType_RESTATED",
	"E": "ExecType_PENDING_REPLACE",
}

const (
	BuyPriceMultiplier  = 1.05
	SellPriceMultiplier = 0.95
)

const (
	Reset           = "\033[0m"
	Red             = "\033[31m"
	Green           = "\033[32m"
	Yellow          = "\033[33m"
	Blue            = "\033[34m"
	Purple          = "\033[35m"
	Cyan            = "\033[36m"
	Gray            = "\033[37m"
	White           = "\033[97m"
	SuccessfulLogon = "---------------Successful Logon---------------"
	LineSpacer      = "----------------------------------------------"
	Ascii           = `
 ██████╗██████╗      ██████╗ ███████╗███╗   ███╗███████╗
██╔════╝██╔══██╗    ██╔═══██╗██╔════╝████╗ ████║██╔════╝
██║     ██████╔╝    ██║   ██║█████╗  ██╔████╔██║███████╗
██║     ██╔══██╗    ██║   ██║██╔══╝  ██║╚██╔╝██║╚════██║
╚██████╗██████╔╝    ╚██████╔╝███████╗██║ ╚═╝ ██║███████║
 ╚═════╝╚═════╝      ╚═════╝ ╚══════╝╚═╝     ╚═╝╚══════╝`
)

const (
	FixMsgExecType     = "8"
	FixMsgReject       = "3"
	FixMsgLogon        = "A"
	FixTagNewOrder     = "20=0"
	FixTagPortfolioId  = 1
	FixTagClOrdId      = 11
	FixTagMsgSeqNum    = 34
	FixTagMsgType      = 35
	FixTagOrderId      = 37
	FixTagOrderQty     = 38
	FixTagOrdType      = 40
	FixTagPrice        = 44
	FixTagSendingTime  = 52
	FixTagSide         = 54
	FixTagSymbol       = 55
	FixTagTargetCompId = 56
	FixTagText         = 58
	FixTagTimeInForce  = 59
	FixTagRawDataLen   = 95
	FixTagRawData      = 96
	FixTagExecType     = 150
	FixTagPassword     = 554
	FixTagExecInst     = 847
	FixTagAccessKey    = 9407
	FixOrdTypeMarket   = "1"
	FixOrdTypeLimit    = "2"
	FixTimeInForceGTC  = "1"
	FixTimeInForceIOC  = "3"
	FixExecInstMarket  = "M"
	FixExecInstLimit   = "L"
	FixSideBuy         = "1"
	FixSideSell        = "2"
	FixExecNotReturned = "Not Returned"
	FixExecCanceled    = "Canceled"
	FixExecFill        = "Fill"
)

const (
	SelectTrade     = "1"
	SelectMarket    = "2"
	SelectOrder     = "3"
	SelectExit      = "x"
	SelectExitWs    = "X"
	AppendCancel    = "-c"
	ArgMarket       = "mkt"
	ArgLimit        = "lim"
	ArgBuy          = "b"
	ArgSell         = "s"
	TradeTypeMarket = "MARKET"
	TradeTypeLimit  = "LIMIT"
	TradeSideBuy    = "BUY"
	TradeSideSell   = "SELL"
	LevelSideBid    = "bid"
	LevelSideOffer  = "offer"
	MinRequiredArgs = 4
)

const (
	SelectOpenOrders = iota + 1
	SelectClosedOrders
	SelectBalances
)

const (
	TradeInput = iota + 1
	MarketData
	OrderManager
)
