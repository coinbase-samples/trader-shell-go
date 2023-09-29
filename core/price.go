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
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"log"
	"net/http"
	"time"
)

type PriceData struct {
	Ask   string    `json:"ask"`
	Bid   string    `json:"bid"`
	Price string    `json:"price"`
	Time  time.Time `json:"time"`
}

var priceCache = make(map[string]PriceData)

func getAndCheckPrice(app *TradeApp, productId string) {
	currentPrice, err := fetchPrice(productId)
	if err != nil {
		log.Printf("Failed to fetch price for %s: %v", productId, err)
		return
	}

	processStopOrders(app, productId, currentPrice)
}

func fetchPrice(productId string) (decimal.Decimal, error) {
	url := "https://api.exchange.coinbase.com/products/" + productId + "/ticker"
	resp, err := http.Get(url)
	if err != nil {
		return decimal.Decimal{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Decimal{}, fmt.Errorf("non-200 response code when fetching price for %s: %d", productId, resp.StatusCode)
	}

	var data PriceData
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&data); err != nil {
		return decimal.Decimal{}, fmt.Errorf("failed to decode price data for %s: %v", productId, err)
	}

	priceCache[productId] = data
	return decimal.NewFromString(data.Price)
}

func processStopOrders(app *TradeApp, productId string, currentPrice decimal.Decimal) {
	app.stopOrdersMutex.Lock()
	defer app.stopOrdersMutex.Unlock()

	var toRemove []int
	for i := len(stopOrders) - 1; i >= 0; i-- {
		order := stopOrders[i]
		if order.Product != productId {
			continue
		}

		if order.Side == TradeSideBuy && currentPrice.GreaterThanOrEqual(order.StopPrice) {
			log.Printf("Triggering buy order for %s at price: %s", productId, order.StopPrice.String())
			executeStopBuyOco(app, order)
			toRemove = append(toRemove, i)
		} else if order.Side == TradeSideSell && currentPrice.LessThanOrEqual(order.StopPrice) {
			log.Printf("Triggering sell order for %s at price: %s", productId, order.StopPrice.String())
			executeStopSellOco(app, order)
			toRemove = append(toRemove, i)
		}
	}

	for i := len(toRemove) - 1; i >= 0; i-- {
		removeStopOrder(toRemove[i])
	}
}

func removeStopOrder(index int) {
	if index < 0 || index >= len(stopOrders) {
		log.Printf("Attempted to remove stop order at invalid index %d, stopOrders Length: %d", index, len(stopOrders))
		return
	}
	stopOrders = append(stopOrders[:index], stopOrders[index+1:]...)
}

func executeStopBuyOco(app *TradeApp, order stopOrder) {
	tradeParams := parsedTradeParams{
		Product:      order.Product,
		OrderType:    "MARKET",
		Side:         order.Side,
		BaseQuantity: order.BaseQuantity,
	}
	app.ConstructTrade(tradeParams, "", app.SessionId)

	if err := app.CancelOrder(order.PlacedOrderId); err != nil {
		log.Printf("Failed to cancel order with Id %s: %v", order.PlacedOrderId, err)
	}
}

func executeStopSellOco(app *TradeApp, order stopOrder) {
	tradeParams := parsedTradeParams{
		Product:      order.Product,
		Side:         order.Side,
		BaseQuantity: order.BaseQuantity,
	}
	app.ConstructTrade(tradeParams, fmt.Sprintf("%.2f", order.StopPrice), app.SessionId)

	if err := app.CancelOrder(order.PlacedOrderId); err != nil {
		log.Printf("Failed to cancel order with Id %s: %v", order.PlacedOrderId, err)
	}
}

func StartPriceFetchingTask(app *TradeApp, products []string, interval time.Duration) {
	for _, product := range products {
		getAndCheckPrice(app, product)
	}

	ticker := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				for _, product := range products {
					getAndCheckPrice(app, product)
				}
			}
		}
	}()
}

func (app *TradeApp) validateOrderAgainstFFP(product, side, orderType, limitPrice string, amount float64) bool {
	priceData, exists := priceCache[product]
	if !exists {
		fmt.Printf(Yellow+"Warning: Product not added to fat finger protection. Add %s to products in main.go.\n"+Reset, product)
		return true
	}

	var maxLimPrice, bestPrice decimal.Decimal
	var err error
	switch side {
	case TradeSideBuy:
		bestPrice, err = decimal.NewFromString(priceData.Bid)
		if err != nil {
			log.Printf("Error parsing Bid price: %v", err)
			return false
		}
		multiplier := decimal.NewFromFloat(BuyPriceMultiplier)
		maxLimPrice = bestPrice.Mul(multiplier)

	case TradeSideSell:
		bestPrice, err = decimal.NewFromString(priceData.Ask)
		if err != nil {
			log.Printf("Error parsing Ask price: %v", err)
			return false
		}
		multiplier := decimal.NewFromFloat(SellPriceMultiplier)
		maxLimPrice = bestPrice.Mul(multiplier)
	}
	amountDecimal := decimal.NewFromFloat(amount)
	spend := bestPrice.Mul(amountDecimal)

	if spend.GreaterThan(app.MaxOrderSize) {
		fmt.Println("Error: Order size exceeds the max order size limit.")
		return false
	}

	if orderType == TradeTypeLimit {
		limitPriceDecimal, err := decimal.NewFromString(limitPrice)
		if err != nil {
			fmt.Println("Error: Failed to convert limitPrice to decimal.")
			return false
		}

		if (side == TradeSideBuy && limitPriceDecimal.GreaterThan(maxLimPrice)) || (side == TradeSideSell && limitPriceDecimal.LessThan(maxLimPrice)) {
			fmt.Println("Error: Order price deviates more than 5% from the best bid/ask.")
			return false
		}
	}

	return true
}
