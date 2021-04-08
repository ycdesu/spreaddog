package max

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/fastjson"

	"github.com/ycdesu/spreaddog/pkg/types"
)

type PublicService struct {
	client *RestClient
}

type Market struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	BaseUnit           string  `json:"base_unit"`
	BaseUnitPrecision  int     `json:"base_unit_precision"`
	QuoteUnit          string  `json:"quote_unit"`
	QuoteUnitPrecision int     `json:"quote_unit_precision"`
	MinBaseAmount      float64 `json:"min_base_amount"`
	MinQuoteAmount     float64 `json:"min_quote_amount"`
}

type Ticker struct {
	Time time.Time

	At          int64  `json:"at"`
	Buy         string `json:"buy"`
	Sell        string `json:"sell"`
	Open        string `json:"open"`
	High        string `json:"high"`
	Low         string `json:"low"`
	Last        string `json:"last"`
	Volume      string `json:"vol"`
	VolumeInBTC string `json:"vol_in_btc"`
}

func (s *PublicService) Timestamp() (serverTimestamp int64, err error) {
	// sync timestamp with server
	req, err := s.client.newRequest("GET", "v2/timestamp", nil, nil)
	if err != nil {
		return 0, err
	}

	response, err := s.client.sendRequest(req)
	if err != nil {
		return 0, err
	}

	err = response.DecodeJSON(&serverTimestamp)
	if err != nil {
		return 0, err
	}

	return serverTimestamp, nil
}

func (s *PublicService) Markets() ([]Market, error) {
	req, err := s.client.newRequest("GET", "v2/markets", url.Values{}, nil)
	if err != nil {
		return nil, err
	}

	response, err := s.client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	var m []Market
	if err := response.DecodeJSON(&m); err != nil {
		return nil, err
	}

	return m, nil
}

func (s *PublicService) Tickers() (map[string]Ticker, error) {
	var endPoint = "v2/tickers"
	req, err := s.client.newRequest("GET", endPoint, url.Values{}, nil)
	if err != nil {
		return nil, err
	}

	response, err := s.client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	v, err := fastjson.ParseBytes(response.Body)
	if err != nil {
		return nil, err
	}

	o, err := v.Object()
	if err != nil {
		return nil, err
	}

	var tickers = make(map[string]Ticker)
	o.Visit(func(key []byte, v *fastjson.Value) {
		var ticker = mustParseTicker(v)
		tickers[string(key)] = ticker
	})

	return tickers, nil
}

func (s *PublicService) Ticker(market string) (*Ticker, error) {
	var endPoint = "v2/tickers/" + market
	req, err := s.client.newRequest("GET", endPoint, url.Values{}, nil)
	if err != nil {
		return nil, err
	}

	response, err := s.client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	v, err := fastjson.ParseBytes(response.Body)
	if err != nil {
		return nil, err
	}

	var ticker = mustParseTicker(v)
	return &ticker, nil
}

func mustParseTicker(v *fastjson.Value) Ticker {
	var at = v.GetInt64("at")
	return Ticker{
		Time:        time.Unix(at, 0),
		At:          at,
		Buy:         string(v.GetStringBytes("buy")),
		Sell:        string(v.GetStringBytes("sell")),
		Volume:      string(v.GetStringBytes("vol")),
		VolumeInBTC: string(v.GetStringBytes("vol_in_btc")),
		Last:        string(v.GetStringBytes("last")),
		Open:        string(v.GetStringBytes("open")),
		High:        string(v.GetStringBytes("high")),
		Low:         string(v.GetStringBytes("low")),
	}
}

type Interval int64

func ParseInterval(a string) (Interval, error) {
	switch strings.ToLower(a) {

	case "1m":
		return 1, nil

	case "5m":
		return 5, nil

	case "15m":
		return 15, nil

	case "30m":
		return 30, nil

	case "1h":
		return 60, nil

	case "2h":
		return 60 * 2, nil

	case "3h":
		return 60 * 3, nil

	case "4h":
		return 60 * 4, nil

	case "6h":
		return 60 * 6, nil

	case "8h":
		return 60 * 8, nil

	case "12h":
		return 60 * 12, nil

	case "1d":
		return 60 * 24, nil

	case "3d":
		return 60 * 24 * 3, nil

	case "1w":
		return 60 * 24 * 7, nil

	}

	return 0, fmt.Errorf("incorrect resolution: %q", a)
}

type KLine struct {
	Symbol                 string
	Interval               string
	StartTime, EndTime     time.Time
	Open, High, Low, Close float64
	Volume                 float64
	Closed                 bool
}

func (k KLine) KLine() types.KLine {
	return types.KLine{
		Exchange:  "max",
		Symbol:    strings.ToUpper(k.Symbol), // global symbol
		Interval:  types.Interval(k.Interval),
		StartTime: k.StartTime,
		EndTime:   k.EndTime,
		Open:      k.Open,
		Close:     k.Close,
		High:      k.High,
		Low:       k.Low,
		Volume:    k.Volume,
		// QuoteVolume:    util.MustParseFloat(k.QuoteAssetVolume),
		// LastTradeID:    0,
		// NumberOfTrades: k.TradeNum,
		Closed: k.Closed,
	}
}

func (s *PublicService) KLines(symbol string, resolution string, start time.Time, limit int) ([]KLine, error) {
	queries := url.Values{}
	queries.Set("market", symbol)

	interval, err := ParseInterval(resolution)
	if err != nil {
		return nil, err
	}
	queries.Set("period", strconv.Itoa(int(interval)))

	nilTime := time.Time{}
	if start != nilTime {
		queries.Set("timestamp", strconv.FormatInt(start.Unix(), 10))
	}

	if limit > 0 {
		queries.Set("limit", strconv.Itoa(limit)) // default to 30, max limit = 10,000
	}

	req, err := s.client.newRequest("GET", fmt.Sprintf("%s/k", s.client.BaseURL), queries, nil)
	if err != nil {
		return nil, fmt.Errorf("request build error: %s", err.Error())
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %s", err.Error())
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.WithError(err).Error("failed to close resp body")
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseKLines(body, symbol, resolution, interval)
}

func parseKLines(payload []byte, symbol, resolution string, interval Interval) (klines []KLine, err error) {
	var parser fastjson.Parser

	v, err := parser.ParseBytes(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse payload: %s", payload)
	}

	arr, err := v.Array()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get array: %s", payload)
	}

	for _, x := range arr {
		slice, err := x.Array()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get array: %s", payload)
		}

		if len(slice) < 6 {
			return nil, fmt.Errorf("unexpected length of ohlc elements: %s", payload)
		}

		ts, err := slice[0].Int64()
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %s", payload)
		}

		startTime := time.Unix(ts, 0)
		endTime := time.Unix(ts, 0).Add(time.Duration(interval)*time.Minute - time.Millisecond)
		isClosed := time.Now().Before(endTime)
		klines = append(klines, KLine{
			Symbol:    symbol,
			Interval:  resolution,
			StartTime: startTime,
			EndTime:   endTime,
			Open:      slice[1].GetFloat64(),
			High:      slice[2].GetFloat64(),
			Low:       slice[3].GetFloat64(),
			Close:     slice[4].GetFloat64(),
			Volume:    slice[5].GetFloat64(),
			Closed:    isClosed,
		})
	}

	return klines, nil
}
