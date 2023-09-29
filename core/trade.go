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
	"github.com/shopspring/decimal"
	"log"
	"strconv"
	"strings"

	"github.com/quickfixgo/quickfix"
)

type parsedTradeParams struct {
	Product      string
	OrderType    string
	Side         string
	BaseQuantity string
}

type stopOrder struct {
	Product       string
	Side          string
	Amount        float64
	StopPrice     decimal.Decimal
	PlacedOrderId string
	BaseQuantity  string
}

var tempStopOrders = make(map[string]stopOrder)

func (app *TradeApp) ProcessSimpleTradeInput(args []string) {
	isPreview := false
	isOco := false
	var ocoPrice decimal.Decimal
	var err error
	var clOrdId string
	var newOrder stopOrder
	var limitPrice decimal.Decimal

	for i := 0; i < len(args); {
		switch args[i] {
		case "-p":
			isPreview = true
			args = append(args[:i], args[i+1:]...)
			i--
		case "-oco":
			isOco = true
			if i+1 < len(args) {
				ocoPrice, err = decimal.NewFromString(args[i+1])
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

	if isPreview && isOco {
		fmt.Println("Error: -p and -oco flags cannot be used together.")
		return
	}

	if len(args) < MinRequiredArgs {
		fmt.Println("Error: Insufficient parameters.")
		return
	}

	params, limitPriceStr, err := parseArgs(args)
	if err != nil {
		fmt.Println(err)
		return
	}

	if isOco && params.OrderType != TradeTypeLimit {
		fmt.Println("Error: -oco can only be used with limit (lim) orders.")
		return
	}

	if params.OrderType != TradeTypeMarket {
		limitPrice, err = decimal.NewFromString(limitPriceStr)
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

	if isOco && (params.Side == ArgBuy && ocoPrice.LessThanOrEqual(limitPrice) || params.Side == ArgSell && ocoPrice.GreaterThanOrEqual(limitPrice)) {
		fmt.Println("Error: Invalid relationship between order price and OCO price.")
		return
	}

	if !app.validateOrderAgainstFFP(params.Product, params.Side, params.OrderType, limitPriceStr, amount) {
		return
	}

	if isPreview {
		if err := app.PreviewOrder(params, limitPriceStr); err != nil {
			log.Printf("Failed to preview order: %v", err)
		}
		return
	}

	clOrdId = app.ConstructTrade(params, limitPriceStr, app.SessionId)

	if isOco {
		newOrder = stopOrder{
			Product:      params.Product,
			Side:         params.Side,
			BaseQuantity: params.BaseQuantity,
			Amount:       amount,
			StopPrice:    ocoPrice,
		}
		tempStopOrders[clOrdId] = newOrder
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

func parseArgs(args []string) (parsedTradeParams, string, error) {
	product := strings.ToUpper(args[0])
	orderType := getTradeType(args[1])
	side := getTradeSide(args[2])
	baseQuantity := args[3]

	params := parsedTradeParams{
		Product:      product,
		OrderType:    orderType,
		Side:         side,
		BaseQuantity: baseQuantity,
	}

	if params.OrderType == TradeTypeLimit {
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
	if arg == ArgMarket {
		return TradeTypeMarket
	}
	return TradeTypeLimit
}

func getTradeSide(arg string) string {
	if arg == ArgBuy {
		return TradeSideBuy
	}
	return TradeSideSell
}

func (app *TradeApp) ConstructTrade(params parsedTradeParams, limitPrice string, sessionId quickfix.SessionID) string {
	msg, clOrdId := app.CreateHeader(app.PortfolioId, "D")
	setTradeMessage(msg, params, limitPrice)

	if err := quickfix.SendToTarget(msg, sessionId); err != nil {
		log.Printf("Error sending trade: %v", err)
	}
	return clOrdId
}

func setTradeMessage(msg *quickfix.Message, params parsedTradeParams, limitPrice string) {
	msg.Body.SetString(quickfix.Tag(FixTagSymbol), params.Product)
	setOrderType(msg, params.OrderType, limitPrice)
	setSide(msg, params.Side)
	setQuantity(msg, params.BaseQuantity)
}

func setOrderType(msg *quickfix.Message, orderType, limitPrice string) {
	if orderType == TradeTypeMarket {
		msg.Body.SetString(quickfix.Tag(FixTagOrdType), FixOrdTypeMarket)
		msg.Body.SetString(quickfix.Tag(FixTagTimeInForce), FixTimeInForceIOC)
		msg.Body.SetString(quickfix.Tag(FixTagExecInst), FixExecInstMarket)
	} else if orderType == TradeTypeLimit {
		msg.Body.SetString(quickfix.Tag(FixTagOrdType), FixOrdTypeLimit)
		msg.Body.SetString(quickfix.Tag(FixTagTimeInForce), FixTimeInForceGTC)
		msg.Body.SetString(quickfix.Tag(FixTagExecInst), FixExecInstLimit)
		msg.Body.SetString(quickfix.Tag(FixTagPrice), limitPrice)
	}
}

func setSide(msg *quickfix.Message, side string) {
	if side == TradeSideBuy {
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
