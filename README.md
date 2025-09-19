# trader-shell-go

This is an example of a simple order and execution management system (OEMS) built with the [Coinbase Prime API](https://docs.cloud.coinbase.com/prime/reference) for trading over FIX, order tracking and management over REST, and order book management over WebSocket. This tool has several built-in functionalities:

1. Market and limit order placement
2. Order Preview with calculated slippage and commission
3. Fat finger protection for both total notional order size and overly marketable limit orders
4. A maintained, customizable order book for any product supported by Prime
5. Open order management and cancellation
6. Closed order tracking
7. Account balances at the asset level

**Disclaimer: This application is meant to be for example purposes only. It is possible to lose real funds through use of this application. Please exercise extreme caution in using this application when connected to an account with real funds. Further, nothing contained within this application should be viewed as a recommendation to buy or sell digital assets or to employ a particular investment strategy, codes, or APIs. Coinbase makes no representation on the accuracy, suitability, or validity of any information provided herein.**

## Configuring this application

1. Begin by cloning this repository and changing directories with the following commands:
f
```
git clone https://github.com/coinbase-samples/trader-shell-go
cd trader-shell-go
```
2. Ensure all dependencies are available with the following command:
```
go mod download
```
3. Copy and rename the example files that you will use in the next step to provide your credentials:
```
cp creds_ex.json creds.json
cp config_ex.yaml config.yaml
```
4. Provide your Svc_AccountId on line 24 of config.yaml, as well as your API credentials and Portfolio ID to creds.json
5. For FIX to operate, you will need a valid certificate, which you may import directly if you are familiar, or by running this to generate a new certificate:
```
openssl s_client -showcerts -connect fix.prime.coinbase.com:4198 < /dev/null | openssl x509 -outform PEM > fix-prime.coinbase.com.pem
```
5. To run the application, type the following command:
```
go run cmd/cli/* config.yaml
```
## Using this application:
Upon running the application, you should immediately see an OnCreate message and then an `S >>` to admin, followed by a `R <<` response back from the admin saying you are connected to Coinbase Prime over FIX. The most logical scenario in which this step fails is that either the certificate was not properly built, so please make sure you have properly completed step 5.

After successfully getting a response back from the server, you should see a large ASCII art welcoming you to the application, followed by a menu with the following choices:
```
1. trade input
2. market data
3. order manager
```
Type a number and hit enter to make a choice.

1. Trade input is where you can place trades over FIX, or generate an order preview over REST. The required values for order submission are as follows:

`product orderType buyOrSell baseQuantity`

If the orderType is specified as `lim`, the user will need to provide their limit price as the fourth parameter, before `baseQuantity`, as follows:

`product orderType buyOrSell limitPrice baseQuantity`

Examples of common orders are shown below:
```
eth-usd mkt b 0.001
eth-usd lim b 1400 0.001
ltc-usd lim s 100 15 -p
```

These orders translate to the following:

1. I wish to market buy 0.001 ETH on the ETH-USD market
2. I wish to limit buy 0.001 ETH at 1.4k USD on the ETH-USD market
3. I wish to preview a limit sell order for 15 LTC at 100 USD

- The `-p` flag allows you to preview the order (over REST) and then you can hit `g` to submit it
- Fat finger protection will prevent you from placing a market order that costs over a certain notional limit, or a limit order that is more than 5% over the current market price. This functionality requires including additional products to the supportedProducts variable within create.go, as well as adjusting MaxOrderSize, also within create.go.


2. Market data will allow you to subscribe to any available Coinbase Prime product and visualize its order book up to 9 levels deep, e.g.:
```
eth-usd 5
```
This screen will also present your available balance so that you may quickly exit and place a trade with the relevant trade information that this screen provides.

3. Order manager provides insight into open, closed, and balance data. You are able to cancel open orders directly from the open orders screen by naming an order by number and including `-c`, i.e. `1 -c`
