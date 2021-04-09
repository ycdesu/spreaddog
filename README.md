# SpreadDog

SpreadDog is used to monitor the spread of different markets from different exchanges.

* ***DO NOT RUN THIS PROJECT IN ANY PRODUCTION ENVIRONMENT***
* ***THE CODEBASE IS DIFFERENT FROM ORIGINAL BBGO***
* ***THIS IS JUST A SAMPLE PROJECT***

This is an example project of creating custom strategy in the [bbgo](https://github.com/c9s/bbgo/). bbgo is a very flexible
trading framework, and we can write our own strategies or run builtin strategies from bbgo. I suggest you to trace the
bbgo codes and write your own strategies.

Note that this project is just an *SAMPLE* project to show how flexible bbgo is. In order to make it simple, **I remove
some functions from bbgo**. Please clone bbgo from the original repository and write your own strategies. 


## REGISTRATION

You could register exchange accounts from the following referral link.

* FTX: <https://ftx.com/#a=ycdesu> 
* MAX: <https://max.maicoin.com/signup?r=5b5c8be5>

## Settings

### dotenv file

SpreadDog monitors the public data, so it doesn't need your exchange credentials. You only have to apply the `SLACK_TOKEN`
in your dotenv file. Let's create the dotenv file `.env.local` and specify the `SLACK_TOKEN`. You only have to give 
`chat:write` permission to your slack bot.

```
SLACK_TOKEN=<your slack token>
```

I remove some checks from bbgo, so we don't have to assign exchange credentials here.

### config

modify `config/spreadmonitor.yaml` yourself. 

There are three sections in the config: exchanges, notification and strategy parameters.

1. Define the exchange you want to access. Here we define three exchanges: ftx, max and binance. You don't have to 
assign the credentials.

```yaml
---
sessions:
  ftx:
    exchange: ftx
    envVarPrefix: ftx
    publicOnly: true

  max:
    exchange: max
    envVarPrefix: max
    publicOnly: true

  binance:
    exchange: binance
    envVarPrefix: binance
    publicOnly: true

```

2. Define the slack channel. bbgo also supports telegram, please read the tutorial from the original repo <https://github.com/c9s/bbgo>.

When the SpreadDog is started, it sends a keep alive message to the `defaultChannel` every 8 hours.

```yaml
notifications:
  slack:
    # will send keep alive message to the channel every 8 hours.
    defaultChannel: "general"
```

3. Configure the strategy parameters

You could attach more than one config to the `- spreadmonitor` array. 

The spread definition: `TargetExchangePrice / SourceExchangePrice`. The behavior is described in the following comments:

```yaml
crossExchangeStrategies:
  # The spread definition: TargetExchangePrice / SourceExchangePrice
  - spreadmonitor:
      - sourceExchange: binance
        # taker fee in the exchange, such as 0.0000023
        sourceExchangeTakerFee: 0
        # ask or bid
        sourceExchangeSide: bid
        # MUST follows the naming rule of an exchange. For example, FTX uses `/` to separate base and quote unit, so we should
        # use `LTC/USDT` in ftx but `LTCUSDT` in max and binance.
        sourceExchangeMarket: LTCUSDT

        targetExchange: ftx
        targetExchangeTakerFee: 0
        targetExchangeSide: ask
        targetExchangeMarket: LTC/USD

        # the string will be in the beginning of the alert
        upperLimitMessage: LTC/USD spread of binance > ftx
        # An alert will be sent if the spread is above the upper limit.
        spreadUpperLimitBps: 10
        # support `s` for second, `m` for minute and `h` for hour.
        # An alert will be emitted if the spread is greater than `spreadUpperLimitBps` for `aboveLimitDuration`.
        # In this example, if the current spread is 20 bps for 1 hour, an alert will be emitted.
        aboveLimitDuration: 1h

        lowerLimitMessage: LTC/USD spread of binance < ftx
        # You will receive a notification if the spread is less than or equal to `spreadLowerLimitBps` for `belowLimitDuration`.
        spreadLowerLimitBps: 2
        belowLimitDuration: 10s

        slackChannelName: test
        
        # Within the quietDuration, you will only receive ONE above limit alert and ONE below limit alert.
        # Let's say the `quietDuration` is 1h. When you receive an lower limit alert at 3pm, 
        # you will not receive another one until 4pm.
        quietDuration: 1h
```

4. Start it

```
go run ./cmd/bbgo run --config=config/spreadmonitor.yaml
```