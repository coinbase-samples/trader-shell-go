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
	"0": "New",
	"1": "Partial Fill",
	"2": "Fill",
	"3": "Done for day",
	"4": "Canceled",
	"5": "Replace",
	"6": "Pending Cancel",
	"7": "Stopped",
	"8": "Rejected",
	"9": "Suspended",
	"A": "Pending New",
	"B": "Calculated",
	"C": "Expired",
	"D": "Restated",
	"E": "Pending Replace",
}

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
	Ascii  = `
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
)
