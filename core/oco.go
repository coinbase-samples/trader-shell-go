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
	"fmt"
	"os"
	"strconv"
	"strings"
)

func (app *TradeApp) DisplayStopOrders() {
	reader := bufio.NewReader(os.Stdin)
	for {
		if len(StopOrders) == 0 {
			fmt.Println("No stop orders found!")
			return
		}

		app.printStopOrders()

		fmt.Print("Select a stop order by number with '-c' to cancel, or type 'x' to return to previous menu: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}
		input = strings.TrimSpace(input)

		if input == "x" {
			return
		}

		autoCancel := false
		if strings.HasSuffix(input, "-c") {
			autoCancel = true
			input = strings.TrimSuffix(input, "-c")
			input = strings.TrimSpace(input)
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice <= 0 || choice > len(StopOrders) {
			fmt.Println("Invalid choice. Please select again.")
			continue
		}

		if autoCancel {
			StopOrders = append(StopOrders[:choice-1], StopOrders[choice:]...)
			fmt.Printf("Removed stop order #%d\n", choice)
		}
	}
}

func (app *TradeApp) printStopOrders() {
	fmt.Println(Blue + "No. | Product | Side | Amount | Stop Price | Linked Order ID" + Reset)
	fmt.Println("---------------------------------------------------------")
	for i, order := range StopOrders {
		fmt.Printf(Blue+"%d. %s | %s | %f | %f | %s\n"+Reset, i+1, order.Product, order.Side, order.Amount, order.StopPrice, order.PlacedOrderID)
	}
}

func orderExistsInStopOrders(orderID string) bool {
	for _, order := range StopOrders {
		if order.PlacedOrderID == orderID {
			return true
		}
	}
	return false
}

func findOrderIndexByID(orderID string) int {
	for i, order := range StopOrders {
		if order.PlacedOrderID == orderID {
			return i
		}
	}
	return -1
}
