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

func (app *TradeApp) CreateHeader(portfolioId, messageType string) (*quickfix.Message, string) {
	message := quickfix.NewMessage()

	message.Header.SetString(quickfix.Tag(FixTagMsgType), messageType)
	message.Header.SetString(quickfix.Tag(FixTagPortfolioId), portfolioId)

	clOrdId := uuid.New().String()
	message.Header.SetString(quickfix.Tag(FixTagClOrdId), clOrdId)

	return message, clOrdId
}

func (app *TradeApp) OnCreate(sessionId quickfix.SessionID) {
	fmt.Println(Green+"OnCreate : Session "+Reset, sessionId)
	app.SessionId = sessionId
	return
}

func (app *TradeApp) OnLogon(sessionId quickfix.SessionID) {
	fmt.Println(SuccessfulLogon)
	app.SessionId = sessionId
	fmt.Println(Ascii)
	app.LogonChannel <- true
	return
}

func (app *TradeApp) OnLogout(sessionId quickfix.SessionID) {
	fmt.Println("OnLogout")
	return
}

func (app *TradeApp) onMessage(message *quickfix.Message, sessionId quickfix.SessionID) (reject quickfix.MessageRejectError) {
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
			fmt.Println("Message Rejected, Reason:", FixExecNotReturned)
		}
	}

	return nil
}

func (app *TradeApp) getExecType(message *quickfix.Message) {
	stopOrdersMutex.Lock()
	defer stopOrdersMutex.Unlock()

	execTypeField, err := message.Body.GetString(quickfix.Tag(FixTagExecType))
	if err != nil {
		log.Printf("Error parsing execTypeField: %v", err)
		return
	}

	execTypeDescription, ok := execTypeDescriptions[execTypeField]
	if !ok {
		execTypeDescription = "Unknown"
	}

	reason := "Not Returned"
	if textField, err := message.Body.GetString(quickfix.Tag(FixTagText)); err == nil {
		reason = textField
	}

	orderIdField, err := message.Body.GetString(quickfix.Tag(FixTagOrderId))
	if err != nil {
		log.Printf("Error parsing orderIdField: %v", err)
		return
	}

	clOrdIdField, err := message.Body.GetString(quickfix.Tag(FixTagClOrdId))
	if err != nil {
		log.Printf("Error parsing clOrdIdField: %v", err)
		return
	}

	if tempOrder, ok := tempStopOrders[clOrdIdField]; ok {

		tempOrder.PlacedOrderId = orderIdField
		delete(tempStopOrders, clOrdIdField)

		if !orderExistsInStopOrders(orderIdField) {
			stopOrders = append(stopOrders, tempOrder)
		}
	}

	if execTypeDescription == FixExecFill || execTypeDescription == FixExecCanceled {
		index := findOrderIndexById(orderIdField)
		if index != -1 {
			stopOrders = append(stopOrders[:index], stopOrders[index+1:]...)
		}
	}

	if reason == FixExecNotReturned {
		fmt.Printf(Green+"ExecType: %s (%s), OrderId: %s\n"+Reset, execTypeField, execTypeDescription, orderIdField)
	} else {
		fmt.Printf(Green+"ExecType: %s (%s), Reason: %s, OrderId: %s\n"+Reset, execTypeField, execTypeDescription, reason, orderIdField)
	}
}

func (app *TradeApp) ToAdmin(message *quickfix.Message, sessionId quickfix.SessionID) {
	msgTypeField, err := message.Header.GetString(quickfix.Tag(FixTagMsgType))
	if err != nil {
		log.Fatalf("Error setting header: %v", err)
	}

	if msgTypeField == FixMsgLogon {
		sendingTime, _ := message.Header.GetString(quickfix.Tag(FixTagSendingTime))
		msgSeqNum, _ := message.Header.GetInt(quickfix.Tag(FixTagMsgSeqNum))
		targetCompId, _ := message.Header.GetString(quickfix.Tag(FixTagTargetCompId))
		rawData := app.sign(sendingTime, msgTypeField, strconv.Itoa(msgSeqNum), targetCompId)

		message.Header.SetField(quickfix.Tag(FixTagPassword), quickfix.FIXString(app.Passphrase))
		message.Header.SetField(quickfix.Tag(FixTagRawData), quickfix.FIXString(rawData))
		message.Header.SetField(quickfix.Tag(FixTagRawDataLen), quickfix.FIXInt(len(rawData)))
		message.Header.SetField(quickfix.Tag(FixTagAccessKey), quickfix.FIXString(app.ApiKey))
	}
	fmt.Println(Green+"(Admin) S >> "+Reset, message)
}

func (app *TradeApp) ToApp(message *quickfix.Message, sessionId quickfix.SessionID) (err error) {
	return
}

func (app *TradeApp) FromAdmin(message *quickfix.Message, sessionId quickfix.SessionID) (reject quickfix.MessageRejectError) {
	fmt.Println(Green+"(Admin) R << "+Reset, message)
	app.onMessage(message, sessionId)
	return nil
}

func (app *TradeApp) FromApp(message *quickfix.Message, sessionId quickfix.SessionID) (reject quickfix.MessageRejectError) {
	app.onMessage(message, sessionId)
	return nil
}

func (app TradeApp) sign(t, msgType, seqNum, targetCompId string) string {
	message := []byte(t + msgType + seqNum + app.ApiKey + targetCompId + app.Passphrase)
	hmac256 := hmac.New(sha256.New, []byte(app.ApiSecret))
	hmac256.Write(message)
	signature := base64.StdEncoding.EncodeToString(hmac256.Sum(nil))
	return signature
}
