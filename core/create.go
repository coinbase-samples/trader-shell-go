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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/quickfixgo/quickfix"
)

const (
	credsFile     = "creds.json"
	priceFetchGap = 10 * time.Second
	MaxOrderSize  = 50000.0
)

type Config struct {
	Passphrase   string
	APIKey       string
	APISecret    string
	PortfolioID  string
	SVCAccountID string
}

type TradeApp struct {
	*quickfix.MessageRouter
	Config
	SessionID    quickfix.SessionID
	OrderBook    *OrderBookProcessor
	disconnect   bool
	FirstPrint   bool
	MaxOrderSize float64
}

var supportedProducts = []string{
	"ETH-USD",
	"LTC-USD",
}
var StopOrders []StopOrder

func DisplayMainMenu() {
	fmt.Println("----------------------------------------------")
	fmt.Println("Choose an option:")
	fmt.Println("1. Trade input")
	fmt.Println("2. Market data")
	fmt.Println("3. Order manager")
	fmt.Println("4. OCO manager")
	fmt.Println("Type 'x' to quit.")
}

func HandleMainMenuChoice(choice string, app *TradeApp, reader *bufio.Reader) {
	switch choice {
	case "1":
		tradeInputMode(app, reader)
	case "2":
		app.MarketDataMode(reader)
	case "3":
		orderManagerMode(app, reader)
	case "4":
		app.DisplayStopOrders()
	case "x":
		fmt.Println("Exiting...")
		os.Exit(0)
	default:
		fmt.Println("Invalid choice. Please select a valid option.")
	}
}

func tradeInputMode(app *TradeApp, reader *bufio.Reader) {
	for {
		usdBalance, err := app.GetAssetBalance("USD")
		if err != nil {
			fmt.Println("Error fetching USD balance:", err)
		} else {
			fmt.Printf(Blue+"USD Balance - Total: %s | Holds: %s | Available: %s\n"+Reset, usdBalance.Amount, usdBalance.Holds, usdBalance.WithdrawableAmount)
		}

		fmt.Println("Enter trade. type 'h' for help. Type 'x' to quit.")
		input, err := GetUserInput(reader)
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		if strings.ToLower(input) == "x" {
			break
		}

		args := strings.Split(input, " ")

		app.ProcessSimpleTradeInput(args)
		if strings.ToLower(input) != "h" {
			fmt.Println("------------------------------")
		}
		time.Sleep(time.Second * 1)
	}
}

func orderManagerMode(app *TradeApp, reader *bufio.Reader) {
	for {
		fmt.Println("------------------------------")
		fmt.Println("Select an option:")
		fmt.Println("1. Manage open orders")
		fmt.Println("2. View recent closed orders")
		fmt.Println("3. View portfolio balances")
		fmt.Println("Type 'x' to cancel")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "x" {
			return
		}

		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Invalid choice. Please select again.")
			continue
		}

		switch choice {
		case 1:
			err := app.GetOpenOrders()
			if err != nil {
				return
			}
		case 2:
			err := app.GetAllOrders()
			if err != nil {
				return
			}
		case 3:
			err := app.ViewPortfolioBalances()
			if err != nil {
				return
			}
		default:
			fmt.Println("Invalid choice. Please select again.")
		}
	}
}

func loadConfig(fileName string) (*os.File, error) {
	return os.Open(fileName)
}

func loadAppSettings(cfg *os.File) (*quickfix.Settings, error) {
	return quickfix.ParseSettings(cfg)
}

func loadCredentials(fileName string) (*Config, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	credentials := &Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&credentials)
	return credentials, err
}

func GetUserInput(reader *bufio.Reader) (string, error) {
	fmt.Print("> ")
	input, err := reader.ReadString('\n')
	return strings.TrimSpace(input), err
}

func InitializeApp(args []string) (*quickfix.Settings, *Config) {
	cfg, err := loadConfig(args[1])
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	appSettings, err := loadAppSettings(cfg)
	if err != nil {
		log.Fatalf("Error parsing settings: %v", err)
	}

	credentials, err := loadCredentials(credsFile)
	if err != nil {
		log.Fatalf("Error loading credentials: %v", err)
	}

	return appSettings, credentials
}

func CreateTradeApp(credentials *Config) *TradeApp {
	return &TradeApp{
		MessageRouter: quickfix.NewMessageRouter(),
		Config:        *credentials,
		FirstPrint:    true,
		MaxOrderSize:  MaxOrderSize,
	}
}

func StartServices(app *TradeApp, appSettings *quickfix.Settings) {
	storeFactory := quickfix.NewFileStoreFactory(appSettings)
	logFactory := quickfix.NewNullLogFactory()

	initiator, err := quickfix.NewInitiator(app, storeFactory, appSettings, logFactory)
	if err != nil {
		log.Fatalf("Error creating initiator: %v", err)
	}

	go initiator.Start()
	time.Sleep(time.Second * 2)

	products := supportedProducts
	StartPriceFetchingTask(app, products, priceFetchGap)
}
