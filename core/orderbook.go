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
	"math"
	"sort"
	"strconv"
)

type LevelJSON struct {
	Side string `json:"side"`
	Px   string `json:"px"`
	Qty  string `json:"qty"`
}

type Level struct {
	Side string  `json:"side"`
	Px   float64 `json:"px"`
	Qty  float64 `json:"qty"`
}

type OrderBookProcessor struct {
	Bids   []Level
	Offers []Level
}

func NewOrderBookProcessor(snapshot string) *OrderBookProcessor {
	var snapshotData struct {
		Events []struct {
			Updates []LevelJSON
		}
	}

	err := json.Unmarshal([]byte(snapshot), &snapshotData)
	if err != nil {
		log.Printf("Failed to parse snapshot JSON: %v", err)
		return nil
	}

	var bids, offers []Level
	for _, event := range snapshotData.Events {
		for _, update := range event.Updates {
			level, err := levelFromJSON(update)
			if err != nil {
				log.Printf("Error converting LevelJSON to Level: %v", err)
				continue
			}
			if level.Side == "bid" {
				bids = append(bids, *level)
			} else if level.Side == "offer" {
				offers = append(offers, *level)
			}
		}
	}

	processor := &OrderBookProcessor{
		Bids:   bids,
		Offers: offers,
	}
	processor.sort()

	return processor
}

func levelFromJSON(l LevelJSON) (*Level, error) {
	px, err := strconv.ParseFloat(l.Px, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Px to float64: %v", err)
	}

	qty, err := strconv.ParseFloat(l.Qty, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Qty to float64: %v", err)
	}

	return &Level{Side: l.Side, Px: px, Qty: qty}, nil
}

func (p *OrderBookProcessor) ApplyUpdate(data string) {
	var event struct {
		Channel string
		Events  []struct {
			Updates []LevelJSON
		}
	}

	err := json.Unmarshal([]byte(data), &event)
	if err != nil {
		log.Printf("Failed to parse update JSON: %v", err)
		return
	}

	if event.Channel != "l2_data" {
		return
	}

	for _, e := range event.Events {
		for _, update := range e.Updates {
			p.apply(update)
		}
	}
	p.filterClosed()
	p.sort()
}

func (p *OrderBookProcessor) apply(levelJSON LevelJSON) {
	level, err := levelFromJSON(levelJSON)
	if err != nil {
		log.Printf("Error converting LevelJSON to Level: %v", err)
		return
	}

	target := &p.Bids
	if level.Side == "offer" {
		target = &p.Offers
	} else if level.Side != "bid" {
		log.Printf(Red+"Error: Unrecognized side: %s"+Reset, level.Side)
		return
	}

	found := false
	for i, existing := range *target {
		if existing.Px == level.Px {
			(*target)[i] = *level
			found = true
			break
		}
	}
	if !found {
		*target = append(*target, *level)
	}
}

func (p *OrderBookProcessor) filterClosed() {
	p.Bids = filterZeroQty(p.Bids)
	p.Offers = filterZeroQty(p.Offers)
}

func filterZeroQty(levels []Level) []Level {
	var result []Level
	for _, level := range levels {
		if level.Qty > 0 {
			result = append(result, level)
		}
	}
	return result
}

func (p *OrderBookProcessor) GetTopNBids(n int) []Level {
	if n > len(p.Bids) {
		return p.Bids
	}
	return p.Bids[:n]
}

func (p *OrderBookProcessor) GetTopNOffers(n int) []Level {
	if n > len(p.Offers) {
		return p.Offers
	}
	return p.Offers[:n]
}

func (p *OrderBookProcessor) sort() {
	sort.Slice(p.Bids, func(i, j int) bool {
		return p.Bids[i].Px > p.Bids[j].Px
	})
	sort.Slice(p.Offers, func(i, j int) bool {
		return p.Offers[i].Px < p.Offers[j].Px
	})
}

func displayOrderBook(app *TradeApp, processor *OrderBookProcessor, n int) {
	if !app.FirstPrint {
		fmt.Printf("\033[%dA", 2*n)
	} else {
		app.FirstPrint = false
	}

	topBids := processor.GetTopNBids(n)
	topOffers := processor.GetTopNOffers(n)

	for i, j := 0, len(topOffers)-1; i < j; i, j = i+1, j-1 {
		topOffers[i], topOffers[j] = topOffers[j], topOffers[i]
	}

	printLevels(topOffers, Red+"Ask: %.2f @ %.2f\n"+Reset)
	printLevels(topBids, Green+"Bid: %.2f @ %.2f\n"+Reset)
}

func printLevels(levels []Level, format string) {
	for _, level := range levels {
		roundedQty := math.Round(level.Qty*100) / 100
		roundedPx := math.Round(level.Px*100) / 100
		fmt.Printf(format, roundedQty, roundedPx)
	}
}
