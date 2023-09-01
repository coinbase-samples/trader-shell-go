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
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type PriceData struct {
	Ask   string    `json:"ask"`
	Bid   string    `json:"bid"`
	Price string    `json:"price"`
	Time  time.Time `json:"time"`
}

var priceCache = make(map[string]PriceData)

func getAndCheckPrice(app *TradeApp, productID string) {
	currentPrice, err := fetchPrice(productID)
	if err != nil {
		log.Printf("Failed to fetch price for %s: %v", productID, err)
		return
	}

	processStopOrders(app, productID, currentPrice)
}

func fetchPrice(productID string) (float64, error) {
	url := "https://api.exchange.coinbase.com/products/" + productID + "/ticker"
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("non-200 response code when fetching price for %s: %d", productID, resp.StatusCode)
	}

	var data PriceData
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return 0, err
	}

	priceCache[productID] = data
	return strconv.ParseFloat(data.Price, 64)
}

var stopOrdersMutex sync.Mutex

func processStopOrders(app *TradeApp, productID string, currentPrice float64) {
	stopOrdersMutex.Lock()
	defer stopOrdersMutex.Unlock()

	var toRemove []int
	for i := len(StopOrders) - 1; i >= 0; i-- {
		order := StopOrders[i]
		if order.Product != productID {
			continue
		}

		if order.Side == "BUY" && currentPrice >= order.StopPrice {
			log.Printf("Triggering buy order for %s at price: %f", productID, order.StopPrice)
			executeStopBuyOCO(app, order)
			toRemove = append(toRemove, i)
		} else if order.Side == "SELL" && currentPrice <= order.StopPrice {
			log.Printf("Triggering sell order for %s at price: %f", productID, order.StopPrice)
			executeStopSellOCO(app, order)
			toRemove = append(toRemove, i)
		}
	}

	for i := len(toRemove) - 1; i >= 0; i-- {
		removeStopOrder(toRemove[i])
	}
}

func removeStopOrder(index int) {
	if index < 0 || index >= len(StopOrders) {
		log.Printf("Attempted to remove stop order at invalid index %d", index)
		return
	}
	StopOrders = append(StopOrders[:index], StopOrders[index+1:]...)
	fmt.Println("Stop loss order removed")
}

func executeStopBuyOCO(app *TradeApp, order StopOrder) {
	tradeParams := ParsedTradeParams{
		Product:      order.Product,
		OrderType:    "MARKET",
		Side:         order.Side,
		BaseQuantity: order.BaseQuantity,
	}
	app.ConstructTrade(tradeParams, "", app.SessionID)

	err := app.CancelOrder(order.PlacedOrderID)
	if err != nil {
		log.Printf("Failed to cancel order with ID %s: %v", order.PlacedOrderID, err)
	}
}

func executeStopSellOCO(app *TradeApp, order StopOrder) {
	tradeParams := ParsedTradeParams{
		Product:      order.Product,
		Side:         order.Side,
		BaseQuantity: order.BaseQuantity,
	}
	app.ConstructTrade(tradeParams, fmt.Sprintf("%.2f", order.StopPrice), app.SessionID)

	err := app.CancelOrder(order.PlacedOrderID)
	if err != nil {
		log.Printf("Failed to cancel order with ID %s: %v", order.PlacedOrderID, err)
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

	var maxLimPrice float64
	var bestPrice float64
	switch side {
	case "BUY":
		bestPrice, _ = strconv.ParseFloat(priceData.Bid, 64)
		maxLimPrice = bestPrice * 1.05
	case "SELL":
		bestPrice, _ = strconv.ParseFloat(priceData.Ask, 64)
		maxLimPrice = bestPrice * 0.95
	}
	spend := bestPrice * amount

	if spend > app.MaxOrderSize {
		fmt.Println("Error: Order size exceeds the max order size limit.")
		return false
	}

	if orderType == "LIMIT" {
		limitPriceFloat, err := strconv.ParseFloat(limitPrice, 64)
		if err != nil {
			fmt.Println("Error: Failed to convert limitPrice to float.")
			return false
		}

		if (side == "BUY" && limitPriceFloat > maxLimPrice) || (side == "SELL" && limitPriceFloat < maxLimPrice) {
			fmt.Println("Error: Order price deviates more than 5% from the best bid/ask.")
			return false
		}
	}

	return true
}
