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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	uri     = "wss://ws-feed.prime.coinbase.com"
	channel = "l2_data"
)

func (app *TradeApp) StartWebSocket(productID string, n int) {
	app.disconnect = false
	log.Println("Type 'x' to disconnect.")

	for {
		doneCh := make(chan struct{})
		if err := app.mainLoop(productID, doneCh, n); err != nil {
			<-doneCh
			if app.disconnect {
				app.FirstPrint = true
				return
			}
			log.Printf(Red+"Error: %v. Retrying in 5 seconds..."+Reset, err)
			time.Sleep(5 * time.Second)
		} else {
			if app.disconnect {
				app.FirstPrint = true
				break
			}
		}
	}
}

func (app *TradeApp) mainLoop(productID string, doneCh chan struct{}, n int) error {
	defer close(doneCh)

	c, _, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		return err
	}
	defer c.Close()

	authMessage, err := app.createAuthMessage(productID)
	if err != nil {
		return err
	}

	if err = c.WriteMessage(websocket.TextMessage, authMessage); err != nil {
		return err
	}

	exitCh := make(chan struct{})
	continueLoop := true

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			if input == "x" {
				app.disconnect = true
				close(exitCh)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf(Red+"Scanner error: %v"+Reset, err)
		}
	}()

	isFirstMessage := true
	for continueLoop {
		select {
		case <-exitCh:
			if err := c.Close(); err != nil {
				log.Println("Failed to close WebSocket:", err)
			} else {
				log.Println("WebSocket closed successfully")
			}
			continueLoop = false

		default:
			messageType, response, err := c.ReadMessage()
			if err != nil {
				log.Println("Failed to read WebSocket message:", err)
				return err
			}
			c.SetReadDeadline(time.Now().Add(10 * time.Second))

			if messageType == websocket.TextMessage {
				if isFirstMessage {
					isFirstMessage = false
					app.OrderBook = NewOrderBookProcessor(string(response))
				} else {
					app.OrderBook.ApplyUpdate(string(response))
				}
				displayOrderBook(app, app.OrderBook, n)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	return nil
}

func (app *TradeApp) createAuthMessage(productID string) ([]byte, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := wsSign(channel, app.APIKey, app.APISecret, app.SVCAccountID, productID, timestamp)

	msg := map[string]interface{}{
		"type":        "subscribe",
		"channel":     channel,
		"access_key":  app.APIKey,
		"api_key_id":  app.SVCAccountID,
		"timestamp":   timestamp,
		"passphrase":  app.Passphrase,
		"signature":   signature,
		"product_ids": []string{productID},
	}

	return json.Marshal(msg)
}

func wsSign(channel, key, secret, accountID, productID, timestamp string) string {
	msg := channel + key + accountID + timestamp + productID
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (app *TradeApp) MarketDataMode(reader *bufio.Reader) {
	for {
		fmt.Println("Enter product to subscribe to (format: asset1-asset2 n) where n is number of top bids/asks (1-9) or type 'x' to return to main menu:")

		input, _ := reader.ReadString('\n')
		input = strings.ToUpper(strings.TrimSpace(input))
		if input == "X" {
			return
		}

		parts := strings.Split(input, " ")
		if len(parts) != 2 || !validateProductFormat(parts[0]) {
			fmt.Println("Invalid input format. Please try again.")
			continue
		}

		product, nStr := parts[0], parts[1]
		n, err := strconv.Atoi(nStr)
		if err != nil || n < 1 || n > 9 {
			fmt.Println("Invalid number of top bids/asks. Please enter a value between 1 and 9.")
			continue
		}

		assetParts := strings.Split(product, "-")
		if len(assetParts) > 0 {
			asset := assetParts[0]
			balance, err := app.GetAssetBalance(asset)
			if err != nil {
				fmt.Printf("Error fetching balance for %s: %s\n", asset, err)
			} else {
				fmt.Printf(Blue+"Balance for %s: Total: %s, Holds: %s, Available: %s\n"+Reset,
					asset, balance.Amount, balance.Holds, balance.WithdrawableAmount)
			}
		}

		app.StartWebSocket(product, n)
	}
}

func validateProductFormat(product string) bool {
	return len(strings.Split(product, "-")) == 2
}
