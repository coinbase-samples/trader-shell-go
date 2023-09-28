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

func (app *TradeApp) displayStopOrders() {
	reader := bufio.NewReader(os.Stdin)
	for {
		if len(stopOrders) == 0 {
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

		if input == SelectExit {
			return
		}

		autoCancel := false
		if strings.HasSuffix(input, AppendCancel) {
			autoCancel = true
			input = strings.TrimSuffix(input, AppendCancel)
			input = strings.TrimSpace(input)
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice <= 0 || choice > len(stopOrders) {
			fmt.Println("Invalid choice. Please select again.")
			continue
		}

		if autoCancel {
			stopOrders = append(stopOrders[:choice-1], stopOrders[choice:]...)
			fmt.Printf("Removed stop order #%d\n", choice)
		}
	}
}

func (app *TradeApp) printStopOrders() {
	fmt.Println(Blue + "No. | Product | Side | Amount | Stop Price | Linked Order Id" + Reset)
	fmt.Println(LineSpacer)
	for i, order := range stopOrders {
		fmt.Printf(Blue+"%d. %s | %s | %f | %s | %s\n"+Reset, i+1, order.Product, order.Side, order.Amount, order.StopPrice.String(), order.PlacedOrderId)
	}
}

func orderExistsInStopOrders(orderId string) bool {
	for _, order := range stopOrders {
		if order.PlacedOrderId == orderId {
			return true
		}
	}
	return false
}

func findOrderIndexById(orderId string) int {
	for i, order := range stopOrders {
		if order.PlacedOrderId == orderId {
			return i
		}
	}
	return -1
}
