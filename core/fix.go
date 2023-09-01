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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/google/uuid"
	"github.com/quickfixgo/quickfix"
	"log"
	"strconv"
	"strings"
)

func (app *TradeApp) CreateHeader(portfolioID, messageType string) (*quickfix.Message, string) {
	message := quickfix.NewMessage()

	message.Header.SetString(quickfix.Tag(FixTagMsgType), messageType)
	message.Header.SetString(quickfix.Tag(FixTagPortfolioId), portfolioID)

	clOrdID := uuid.New().String()
	message.Header.SetString(quickfix.Tag(FixTagClOrdId), clOrdID)

	return message, clOrdID
}

func (app *TradeApp) OnCreate(sessionID quickfix.SessionID) {
	fmt.Println(Green+"OnCreate : Session "+Reset, sessionID)
	app.SessionID = sessionID
	return
}

func (app *TradeApp) OnLogon(sessionID quickfix.SessionID) {
	fmt.Println("---------------Successful Logon----------------")
	app.SessionID = sessionID
	fmt.Println(Ascii)
	return
}

func (app *TradeApp) OnLogout(sessionID quickfix.SessionID) {
	fmt.Println("OnLogout")
	return
}

func (app *TradeApp) onMessage(message *quickfix.Message, sessionID quickfix.SessionID) (reject quickfix.MessageRejectError) {
	msgTypeField, err := message.Header.GetString(quickfix.Tag(FixTagMsgType))
	if err != nil {
	}

	switch msgTypeField {
	case FixMsgExecType:
		if strings.Contains(message.String(), FixTagNewOrder) {
			app.getExecType(message)
		}
	case FixMsgReject:
		if textField, err := message.Body.GetString(quickfix.Tag(FixTagText)); err == nil {
			fmt.Println("Message Rejected, Reason:", textField)
		} else {
			fmt.Println("Message Rejected, Reason: Not Returned")
		}
	}

	return nil
}

func (app *TradeApp) getExecType(message *quickfix.Message) {
	execTypeField, err := message.Body.GetString(quickfix.Tag(FixTagExecType))
	if err != nil {
		log.Printf("Error parsing execTypeField: %v", err)
		return
	}

	execTypeDescription, ok := execTypeDescriptions[execTypeField]
	if !ok {
		execTypeDescription = "Unknown"
	}

	var reason string
	if textField, err := message.Body.GetString(quickfix.Tag(FixTagText)); err == nil {
		reason = textField
	} else {
		reason = "Not Returned"
	}

	orderIDField, err := message.Body.GetString(quickfix.Tag(FixTagOrderId))
	if err != nil {
		log.Printf("Error parsing orderIDField: %v", err)
		return
	}

	clOrdIDField, err := message.Body.GetString(quickfix.Tag(FixTagClOrdId))
	if err != nil {
		log.Printf("Error parsing clOrdIDField: %v", err)
		return
	}

	if tempOrder, ok := TempStopOrders[clOrdIDField]; ok {

		tempOrder.PlacedOrderID = orderIDField
		TempStopOrders[clOrdIDField] = tempOrder
		if !orderExistsInStopOrders(orderIDField) {
			StopOrders = append(StopOrders, tempOrder)
		}
	}

	if execTypeDescription == "Fill" || execTypeDescription == "Canceled" {
		index := findOrderIndexByID(orderIDField)
		if index != -1 {
			StopOrders = append(StopOrders[:index], StopOrders[index+1:]...)
		}
	}

	if reason == "Not Returned" {
		fmt.Printf(Green+"ExecType: %s (%s), OrderID: %s\n"+Reset, execTypeField, execTypeDescription, orderIDField)
	} else {
		fmt.Printf(Green+"ExecType: %s (%s), Reason: %s, OrderID: %s\n"+Reset, execTypeField, execTypeDescription, reason, orderIDField)
	}
}

func (app *TradeApp) ToAdmin(message *quickfix.Message, sessionID quickfix.SessionID) {
	msgTypeField, _ := message.Header.GetString(quickfix.Tag(FixTagMsgType))

	if msgTypeField == FixMsgLogon {
		sendingTime, _ := message.Header.GetString(quickfix.Tag(FixTagSendingTime))
		msgSeqNum, _ := message.Header.GetInt(quickfix.Tag(FixTagMsgSeqNum))
		targetCompID, _ := message.Header.GetString(quickfix.Tag(FixTagTargetCompId))
		rawData := app.sign(sendingTime, msgTypeField, strconv.Itoa(msgSeqNum), app.APIKey, targetCompID, app.Passphrase)

		message.Header.SetField(quickfix.Tag(FixTagPassword), quickfix.FIXString(app.Passphrase))
		message.Header.SetField(quickfix.Tag(FixTagRawData), quickfix.FIXString(rawData))
		message.Header.SetField(quickfix.Tag(FixTagRawDataLen), quickfix.FIXInt(len(rawData)))
		message.Header.SetField(quickfix.Tag(FixTagAccessKey), quickfix.FIXString(app.APIKey))
	}
	fmt.Println(Green+"(Admin) S >> "+Reset, message)
}

func (app *TradeApp) ToApp(message *quickfix.Message, sessionID quickfix.SessionID) (err error) {
	return
}

func (app *TradeApp) FromAdmin(message *quickfix.Message, sessionID quickfix.SessionID) (reject quickfix.MessageRejectError) {
	fmt.Println(Green+"(Admin) R << "+Reset, message)
	app.onMessage(message, sessionID)
	return nil
}

func (app *TradeApp) FromApp(message *quickfix.Message, sessionID quickfix.SessionID) (reject quickfix.MessageRejectError) {
	app.onMessage(message, sessionID)
	return nil
}

func (app *TradeApp) sign(t, msgType, seqNum, accessKey, targetCompID, passphrase string) string {
	message := []byte(t + msgType + seqNum + accessKey + targetCompID + passphrase)
	hmac256 := hmac.New(sha256.New, []byte(app.APISecret))
	hmac256.Write(message)
	signature := base64.StdEncoding.EncodeToString(hmac256.Sum(nil))
	return signature
}
