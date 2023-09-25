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

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/coinbase-samples/trader-shell-go/core"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Configuration file path is required as an argument.")
	}

	appSettings, credentials := core.InitializeApp(os.Args)
	app := core.CreateTradeApp(credentials)
	core.StartServices(app, appSettings)

	reader := bufio.NewReader(os.Stdin)

	for {
		core.DisplayMainMenu()
		input, err := core.GetUserInput(reader)
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		core.HandleMainMenuChoice(input, app, reader)
	}
}
