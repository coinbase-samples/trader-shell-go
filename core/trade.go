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

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/quickfixgo/quickfix"
)

type ParsedTradeParams struct {
	Product      string
	OrderType    string
	Side         string
	BaseQuantity string
}

type StopOrder struct {
	Product       string
	Side          string
	Amount        float64
	StopPrice     float64
	PlacedOrderID string
	BaseQuantity  string
}

var TempStopOrders = make(map[string]StopOrder)

func (app *TradeApp) ProcessSimpleTradeInput(args []string) {
	isPreview := false
	isOCO := false
	var ocoPrice float64
	var err error
	var clOrdID string
	var newOrder StopOrder
	var limitPrice float64

	for i := 0; i < len(args); {
		switch args[i] {
		case "-p":
			isPreview = true
			args = append(args[:i], args[i+1:]...)
			i--
		case "-oco":
			isOCO = true
			if i+1 < len(args) {
				ocoPrice, err = strconv.ParseFloat(args[i+1], 64)
				if err != nil {
					fmt.Println("Error: Invalid OCO price.")
					return
				}
				args = append(args[:i], args[i+2:]...)
				i -= 2
			} else {
				fmt.Println("Error: -oco flag should be followed by a valid price.")
				return
			}
		case "h":
			printHelp()
			return
		}
		i++
	}

	if isPreview && isOCO {
		fmt.Println("Error: -p and -oco flags cannot be used together.")
		return
	}

	if len(args) < 4 {
		fmt.Println("Error: Insufficient parameters.")
		return
	}

	params, limitPriceStr, err := parseArgs(args)
	if err != nil {
		fmt.Println(err)
		return
	}

	if isOCO && params.OrderType != "LIMIT" {
		fmt.Println("Error: -oco can only be used with limit (lim) orders.")
		return
	}

	if params.OrderType != "MARKET" {
		limitPrice, err = strconv.ParseFloat(limitPriceStr, 64)
		if err != nil {
			fmt.Println("Error parsing limit price:", err)
			return
		}
	} else {
		limitPriceStr = ""
	}

	amount, err := strconv.ParseFloat(params.BaseQuantity, 64)
	if err != nil {
		fmt.Println("Error: Invalid order size.")
		return
	}

	if isOCO && (params.Side == "b" && ocoPrice <= limitPrice || params.Side == "s" && ocoPrice >= limitPrice) {
		fmt.Println("Error: Invalid relationship between order price and OCO price.")
		return
	}

	if !app.validateOrderAgainstFFP(params.Product, params.Side, params.OrderType, limitPriceStr, amount) {
		return
	}

	if isPreview {
		err := app.PreviewOrder(params, limitPriceStr)
		if err != nil {
			log.Printf("Failed to preview order: %v", err)
		}
		return
	}

	clOrdID = app.ConstructTrade(params, limitPriceStr, app.SessionID)

	if isOCO {
		newOrder = StopOrder{
			Product:      params.Product,
			Side:         params.Side,
			BaseQuantity: params.BaseQuantity,
			Amount:       amount,
			StopPrice:    ocoPrice,
		}
		TempStopOrders[clOrdID] = newOrder
	}
}

func printHelp() {
	fmt.Println(Purple + "Accepts market (mkt) and limit (lim) base quantity orders.")
	fmt.Println("Append '-p' to submit an order preview over REST.")
	fmt.Println("Append '-oco' to submit an OCO order. Manage OCOs from main menu.")
	fmt.Println("Format: product mkt/lim b/s lim_price base_quantity")
	fmt.Println("Ex: eth-usd mkt s 0.001")
	fmt.Println("Ex: eth-usd lim b 1400 0.001")
	fmt.Println("Ex: ltc-usd lim s 100 15 -p")
	fmt.Println("Ex: eth-usd lim b 1500 0.001 -oco 2000\n" + Reset)
}

func parseArgs(args []string) (ParsedTradeParams, string, error) {
	product := strings.ToUpper(args[0])
	orderType := getTradeType(args[1])
	side := getTradeSide(args[2])
	baseQuantity := args[3]

	params := ParsedTradeParams{
		Product:      product,
		OrderType:    orderType,
		Side:         side,
		BaseQuantity: baseQuantity,
	}

	if params.OrderType == "LIMIT" {
		if len(args) <= 4 {
			return params, "", fmt.Errorf("limit price required for limit order")
		}
		limitPrice := args[3]
		params.BaseQuantity = args[4]
		return params, limitPrice, nil
	}

	return params, "", nil
}

func getTradeType(arg string) string {
	if arg == "mkt" {
		return "MARKET"
	}
	return "LIMIT"
}

func getTradeSide(arg string) string {
	if arg == "b" {
		return "BUY"
	}
	return "SELL"
}

func (app *TradeApp) ConstructTrade(params ParsedTradeParams, limitPrice string, sessionID quickfix.SessionID) string {
	msg, clOrdID := app.CreateHeader(app.PortfolioID, "D")
	setTradeMessage(msg, params, limitPrice)

	err := quickfix.SendToTarget(msg, sessionID)
	if err != nil {
		log.Printf("Error sending trade: %v", err)
	}
	return clOrdID
}

func setTradeMessage(msg *quickfix.Message, params ParsedTradeParams, limitPrice string) {
	msg.Body.SetString(quickfix.Tag(FixTagSymbol), params.Product)
	setOrderType(msg, params.OrderType, limitPrice)
	setSide(msg, params.Side)
	setQuantity(msg, params.BaseQuantity)
}

func setOrderType(msg *quickfix.Message, orderType, limitPrice string) {
	if orderType == "MARKET" {
		msg.Body.SetString(quickfix.Tag(FixTagOrdType), FixOrdTypeMarket)
		msg.Body.SetString(quickfix.Tag(FixTagTimeInForce), FixTimeInForceIOC)
		msg.Body.SetString(quickfix.Tag(FixTagExecInst), FixExecInstMarket)
	} else if orderType == "LIMIT" {
		msg.Body.SetString(quickfix.Tag(FixTagOrdType), FixOrdTypeLimit)
		msg.Body.SetString(quickfix.Tag(FixTagTimeInForce), FixTimeInForceGTC)
		msg.Body.SetString(quickfix.Tag(FixTagExecInst), FixExecInstLimit)
		msg.Body.SetString(quickfix.Tag(FixTagPrice), limitPrice)
	}
}

func setSide(msg *quickfix.Message, side string) {
	if side == "BUY" {
		msg.Body.SetString(quickfix.Tag(FixTagSide), FixSideBuy)
	} else {
		msg.Body.SetString(quickfix.Tag(FixTagSide), FixSideSell)
	}
}

func setQuantity(msg *quickfix.Message, baseQuantity string) {
	quantity, err := strconv.ParseFloat(baseQuantity, 64)
	if err != nil {
		log.Printf("Error parsing quantity: %v", err)
		return
	}
	quantityStr := fmt.Sprintf("%f", quantity)
	msg.Body.SetString(quickfix.Tag(FixTagOrderQty), quantityStr)
}
