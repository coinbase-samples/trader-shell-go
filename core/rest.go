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
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	BaseURL          = "https://api.prime.coinbase.com"
	HeaderAccessSig  = "X-CB-ACCESS-SIGNATURE"
	HeaderAccessTime = "X-CB-ACCESS-TIMESTAMP"
	HeaderAccessKey  = "X-CB-ACCESS-KEY"
	HeaderPassphrase = "X-CB-ACCESS-PASSPHRASE"
)

var ErrOrderCanceled = errors.New("order Canceled")

type OrderPreviewResponse struct {
	BaseQuantity       string `json:"base_quantity"`
	QuoteValue         string `json:"quote_value"`
	LimitPrice         string `json:"limit_price"`
	Commission         string `json:"commission"`
	Slippage           string `json:"slippage"`
	BestBid            string `json:"best_bid"`
	BestAsk            string `json:"best_ask"`
	AverageFilledPrice string `json:"average_filled_price"`
	OrderTotal         string `json:"order_total"`
}

type Balance struct {
	Amount             string `json:"amount"`
	Holds              string `json:"holds"`
	WithdrawableAmount string `json:"withdrawable_amount"`
	FiatAmount         string `json:"fiat_amount"`
}

type BalanceResponse struct {
	Balances []Balance `json:"balances"`
}

func (app *TradeApp) makeAuthenticatedRequest(method, path, queryParams string, body []byte) ([]byte, error) {
	uri := BaseURL + path
	if queryParams != "" {
		uri += "?" + queryParams
	}

	timestamp := strconv.Itoa(int(time.Now().Unix()))
	message := timestamp + method + path
	if body != nil {
		message += string(body)
	}
	signature := computeHMAC256(message, app.ApiSecret)

	headers := map[string]string{
		HeaderAccessSig:  signature,
		HeaderAccessTime: timestamp,
		HeaderAccessKey:  app.ApiKey,
		HeaderPassphrase: app.Passphrase,
		"Accept":         "application/json",
	}

	return makeRequest(method, uri, body, headers)
}

func (app *TradeApp) extractOrdersFromResponse(body []byte) ([]interface{}, error) {
	var parsedResponse map[string]interface{}
	if err := json.Unmarshal(body, &parsedResponse); err != nil {
		return nil, err
	}

	orders, ok := parsedResponse["orders"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to extract orders from response")
	}

	return orders, nil
}

func (app *TradeApp) GetOpenOrders() error {
	path := fmt.Sprintf("/v1/portfolios/%s/open_orders", app.PortfolioId)
	body, err := app.makeAuthenticatedRequest("GET", path, "", nil)
	if err != nil {
		return err
	}

	orders, err := app.extractOrdersFromResponse(body)
	if err != nil {
		return err
	}

	if err := app.displayAndSelectOrder(orders, false); err != nil {
		if err == ErrOrderCanceled {
			return app.GetOpenOrders()
		}
		return err
	}
	return nil
}

func (app *TradeApp) GetAllOrders() error {
	path := fmt.Sprintf("/v1/portfolios/%s/orders", app.PortfolioId)
	body, err := app.makeAuthenticatedRequest("GET", path, "", nil)
	if err != nil {
		return err
	}

	orders, err := app.extractOrdersFromResponse(body)
	if err != nil {
		return err
	}

	app.displayAndSelectOrder(orders, true)
	return nil
}

func (app *TradeApp) displayAndSelectOrder(orders []interface{}, allOrders bool) error {
	for {
		if len(orders) == 0 {
			if allOrders {
				fmt.Println("No orders found!")
			} else {
				fmt.Println("No open orders found!")
			}
			return fmt.Errorf("no orders found")
		}

		if allOrders && len(orders) > 20 {
			orders = orders[:20]
		}

		fmt.Println(Blue + "#  | Id                                   | Product | Side | Type   | Lim Px  | Base Qty| Quote Val" + Reset)
		for i, order := range orders {
			orderMap, ok := order.(map[string]interface{})
			if !ok {
				log.Println("Order is not a valid map")
				return fmt.Errorf("invalid order map")
			}

			id := valueOrX(orderMap["id"].(string))
			product := valueOrX(orderMap["product_id"].(string))
			side := valueOrX(orderMap["side"].(string))
			orderType := valueOrX(orderMap["type"].(string))
			limitPrice := valueOrX(orderMap["limit_price"].(string))
			baseQuantity := valueOrX(orderMap["base_quantity"].(string))
			quoteValue := valueOrX(orderMap["quote_value"].(string))

			fmt.Printf(Blue+"%-3d| %-37s| %-8s| %-5s| %-7s| %-8s| %-8s| %s\n"+Reset, i+1, id, product, side, orderType, limitPrice, baseQuantity, quoteValue)

		}

		if allOrders {
			fmt.Print("Type 'x' to return to previous menu: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == SelectExit {
				return nil
			}

			fmt.Println("Invalid choice, please type 'x' to return to previous menu.")
			continue
		}

		fmt.Print("\nSelect an order by number, add '-c' to cancel, or type 'x' to return to previous menu: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == SelectExit {
			return nil
		}

		autoCancel := false
		if strings.HasSuffix(input, "-c") {
			autoCancel = true
			input = strings.TrimSuffix(input, "-c")
			input = strings.TrimSpace(input)
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice <= 0 || choice > len(orders) {
			log.Println("Invalid choice")
			return fmt.Errorf("invalid choice")
		}

		selectedOrder := orders[choice-1]

		if !autoCancel {
			orderJson, err := json.MarshalIndent(selectedOrder, "", "  ")
			if err != nil {
				log.Println("Failed to marshal order:", err)
				return err
			}

			fmt.Println(string(orderJson))
		} else {
			if err := app.userActionOnOpenOrder(selectedOrder, orders, autoCancel); err != nil {
				return ErrOrderCanceled
			}
		}
	}
	return nil
}

func valueOrX(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (app *TradeApp) userActionOnOpenOrder(order interface{}, orders []interface{}, autoCancel bool) error {
	if autoCancel {
		orderMap, ok := order.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid order map")
		}

		id, ok := orderMap["id"].(string)
		if !ok {
			return fmt.Errorf("invalid order Id")
		}
		if err := app.CancelOrder(id); err != nil {
			log.Println("Failed to cancel order:", err)
			return err
		}
		time.Sleep(time.Second * 1)
		return fmt.Errorf("order Canceled")
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\nType 'c' to cancel the order or type 'x' to go back to the order Id selector.")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "c":
			orderMap, ok := order.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid order map")
			}

			id, ok := orderMap["id"].(string)
			if !ok {
				return fmt.Errorf("invalid order Id")
			}
			if err := app.CancelOrder(id); err != nil {
				log.Println("Failed to cancel order:", err)
				return err
			}
			time.Sleep(time.Second * 1)
			return fmt.Errorf("order Canceled")

		case SelectExit:
			return nil
		default:
			fmt.Println("Invalid choice. Please select again.")
		}
	}
	return nil
}

func (app *TradeApp) CancelOrder(orderId string) error {
	path := fmt.Sprintf("/v1/portfolios/%s/orders/%s/cancel", app.PortfolioId, orderId)
	payload := map[string]string{
		"portfolio_id": app.PortfolioId,
		"order_id":     orderId,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = app.makeAuthenticatedRequest("POST", path, "", payloadBytes)
	return err
}

func (app *TradeApp) ViewPortfolioBalances() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Enter an asset (e.g., 'eth') or type 'x' to cancel: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		asset := strings.TrimSpace(input)

		if asset == "" {
			fmt.Println("Invalid input. Please enter a valid asset.")
			continue
		} else if asset == SelectExit {
			break
		}

		balance, err := app.GetAssetBalance(asset)
		if err != nil {
			fmt.Println("Error fetching balance:", err)
			continue
		}
		fmt.Printf(Blue+"Amount: %s\nHolds: %s\nWithdrawable Amount: %s\nFiat Amount: %s\n"+Reset, balance.Amount, balance.Holds, balance.WithdrawableAmount, balance.FiatAmount)
	}
	return nil
}

func formatToUSD(value string) string {
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}
	return fmt.Sprintf("%.2f", floatValue)
}

func (app *TradeApp) GetAssetBalance(asset string) (Balance, error) {
	path := fmt.Sprintf("/v1/portfolios/%s/balances", app.PortfolioId)
	queryParams := fmt.Sprintf("balance_type=TRADING_BALANCES&symbols=%s", asset)
	body, err := app.makeAuthenticatedRequest("GET", path, queryParams, nil)
	if err != nil {
		return Balance{}, err
	}

	var balanceData BalanceResponse
	if err := json.Unmarshal(body, &balanceData); err != nil {
		return Balance{}, err
	}

	if len(balanceData.Balances) > 0 {
		balance := balanceData.Balances[0]
		if asset == "USD" {
			balance.Amount = formatToUSD(balance.Amount)
			balance.Holds = formatToUSD(balance.Holds)
			balance.WithdrawableAmount = formatToUSD(balance.WithdrawableAmount)
			balance.FiatAmount = formatToUSD(balance.FiatAmount)
		}
		return balance, nil
	} else {
		return Balance{}, errors.New("no balance data available for the specified asset")
	}
}

func (app *TradeApp) PreviewOrder(params parsedTradeParams, limitPrice string) error {
	path := fmt.Sprintf("/v1/portfolios/%s/order_preview", app.PortfolioId)

	payload := map[string]string{
		"product_id":      params.Product,
		"client_order_id": uuid.New().String(),
		"side":            params.Side,
		"type":            params.OrderType,
		"base_quantity":   params.BaseQuantity,
	}

	if params.OrderType == TradeTypeLimit {
		payload["limit_price"] = limitPrice
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	responseBytes, err := app.makeAuthenticatedRequest("POST", path, "", payloadBytes)
	if err != nil {
		return err
	}

	var response OrderPreviewResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return err
	}
	printOrderPreview(response)

	app.handlePreviewAction(params, limitPrice)

	return nil
}

func printOrderPreview(response OrderPreviewResponse) {
	val := reflect.ValueOf(response)
	typeOfVal := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldName := typeOfVal.Field(i).Name
		fieldValue := val.Field(i).Interface()
		fmt.Printf(Blue+"%s: %v\n"+Reset, fieldName, fieldValue)
	}
}

func (app *TradeApp) handlePreviewAction(params parsedTradeParams, limitPrice string) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter 'g' to submit order or 'x' to create a new order.")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "g" {
			app.ConstructTrade(params, limitPrice, app.SessionId)
			break
		} else if input == SelectExit {
			fmt.Println("Returning to order creation...")
			break
		} else {
			fmt.Println("Invalid input. Please enter 'g' or 'x'.")
		}
	}
}

func computeHMAC256(message, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func makeRequest(method, uri string, payload []byte, headers map[string]string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, uri, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
