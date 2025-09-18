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
	"github.com/coinbase-samples/trader-shell-go/config"
	"github.com/shopspring/decimal"
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
)

var MaxOrderSize = decimal.NewFromFloat(50000.0)

type TradeApp struct {
	*quickfix.MessageRouter
	config.Config
	SessionId    quickfix.SessionID
	OrderBook    *OrderBookProcessor
	disconnect   bool
	FirstPrint   bool
	MaxOrderSize decimal.Decimal
	LogonChannel chan bool
}

var supportedProducts = []string{
	"ETH-USD",
	"LTC-USD",
}

func DisplayMainMenu() {
	fmt.Println(LineSpacer)
	fmt.Println("Choose an option:")
	fmt.Printf("%d. Trade input\n", TradeInput)
	fmt.Printf("%d. Market data\n", MarketData)
	fmt.Printf("%d. Order manager\n", OrderManager)
	fmt.Printf("Type '%s' to quit.\n", SelectExit)
}

func HandleMainMenuChoice(choice string, app *TradeApp, reader *bufio.Reader) {
	switch choice {
	case SelectTrade:
		app.tradeInputMode(reader)
	case SelectMarket:
		app.MarketDataMode(reader)
	case SelectOrder:
		app.orderManagerMode(reader)
	case SelectExit:
		fmt.Println("Exiting...")
		os.Exit(0)
	default:
		fmt.Println("Invalid choice. Please select a valid option.")
	}
}

func (app *TradeApp) tradeInputMode(reader *bufio.Reader) {
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

		if strings.ToLower(input) == SelectExit {
			break
		}

		args := strings.Split(input, " ")
		app.ProcessSimpleTradeInput(args)
		if strings.ToLower(input) != "h" {
			fmt.Println(LineSpacer)
		}
		time.Sleep(time.Second * 1)
	}
}

func (app *TradeApp) orderManagerMode(reader *bufio.Reader) {
	for {
		fmt.Println(LineSpacer)
		fmt.Println("Select an option:")
		fmt.Printf("%d. Manage open orders\n", SelectOpenOrders)
		fmt.Printf("%d. View recent closed orders\n", SelectClosedOrders)
		fmt.Printf("%d. View portfolio balances\n", SelectBalances)
		fmt.Printf("Type '%s' to cancel\n", SelectExit)

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == SelectExit {
			return
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < SelectOpenOrders || choice > SelectBalances {
			fmt.Println("Invalid choice. Please select again.")
			continue
		}

		switch choice {
		case SelectOpenOrders:
			if err := app.GetOpenOrders(); err != nil {
				fmt.Println("Error:", err)
			}
		case SelectClosedOrders:
			if err := app.GetAllOrders(); err != nil {
				fmt.Println("Error:", err)
			}
		case SelectBalances:
			if err := app.ViewPortfolioBalances(); err != nil {
				fmt.Println("Error:", err)
			}
		}
	}
}

func loadConfig(fileName string) (*os.File, error) {
	return os.Open(fileName)
}

func loadAppSettings(cfg *os.File) (*quickfix.Settings, error) {
	return quickfix.ParseSettings(cfg)
}

func loadCredentials(fileName string) (*config.Config, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	credentials := &config.Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&credentials)
	return credentials, err
}

func GetUserInput(reader *bufio.Reader) (string, error) {
	fmt.Print("> ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.TrimRight(input, "\n\r")), nil
}

func InitializeApp(args []string) (*quickfix.Settings, *config.Config) {
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

func CreateTradeApp(credentials *config.Config) *TradeApp {
	return &TradeApp{
		MessageRouter: quickfix.NewMessageRouter(),
		Config:        *credentials,
		FirstPrint:    true,
		MaxOrderSize:  MaxOrderSize,
		LogonChannel:  make(chan bool),
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

	<-app.LogonChannel

	products := supportedProducts
	StartPriceFetchingTask(app, products, priceFetchGap)
}
