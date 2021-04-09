package spreadmonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/ycdesu/spreaddog/pkg/bbgo"
	"github.com/ycdesu/spreaddog/pkg/types"
)

const (
	ID = "spreadmonitor"
)

func init() {
	bbgo.RegisterStrategy(ID, &Strategy{})
}

var durationRegex = regexp.MustCompile(`(\d+)([smh])`)

func duration(durationStr string) (time.Duration, error) {
	r := durationRegex.FindStringSubmatch(durationStr)
	if len(r) != 3 {
		return 0, fmt.Errorf("duration example: 1s, 1m, 1h. input: %s", durationStr)
	}

	d, err := strconv.ParseInt(r[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration: %s", r[0])
	}

	unitStr := strings.ToLower(r[2])
	var unit time.Duration
	switch unitStr {
	case "s":
		unit = time.Second
	case "m":
		unit = time.Minute
	case "h":
		unit = time.Hour
	default:
		return 0, fmt.Errorf("unsupported unit: %s", r[1])
	}
	return time.Duration(d) * unit, nil
}

type StrategyConfig struct {
	SourceExchange         string  `json:"sourceExchange"`
	SourceExchangeTakerFee float64 `json:"sourceExchangeTakerFee,omitempty"`
	SourceExchangeSide     string  `json:"sourceExchangeSide"`
	SourceExchangeMarket   string  `json:"sourceExchangeMarket"`

	TargetExchange         string  `json:"targetExchange"`
	TargetExchangeTakerFee float64 `json:"targetExchangeTakerFee,omitempty"`
	TargetExchangeSide     string  `json:"targetExchangeSide"`
	TargetExchangeMarket   string  `json:"targetExchangeMarket"`

	UpperLimitMessage   string `json:"upperLimitMessage,omitempty"`
	SpreadUpperLimitBps int64  `json:"spreadUpperLimitBps,omitempty"`
	AboveLimitDuration  time.Duration
	LowerLimitMessage   string `json:"lowerLimitMessage,omitempty"`
	SpreadLowerLimitBps int64  `json:"spreadLowerLimitBps,omitempty"`
	BelowLimitDuration  time.Duration

	SlackChannelName string `json:"slackChannelName"`
	QuietDuration    time.Duration
}

func (c *StrategyConfig) UnmarshalJSON(data []byte) error {
	type alias StrategyConfig
	temp := struct {
		AboveLimitDurationStr string `json:"aboveLimitDuration,omitempty"`
		BelowLimitDurationStr string `json:"belowLimitDuration,omitempty"`
		QuietDurationStr      string `json:"quietDuration,omitempty"`

		*alias
	}{
		alias: (*alias)(c),
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	str := strings.ToLower(strings.TrimSpace(temp.AboveLimitDurationStr))
	if str != "" {
		d, err := duration(str)
		if err != nil {
			return err
		}
		c.AboveLimitDuration = d
	}

	str = strings.ToLower(strings.TrimSpace(temp.BelowLimitDurationStr))
	if str != "" {
		d, err := duration(str)
		if err != nil {
			return err
		}
		c.BelowLimitDuration = d
	}

	str = strings.ToLower(strings.TrimSpace(temp.QuietDurationStr))
	if str != "" {
		d, err := duration(str)
		if err != nil {
			return err
		}
		c.QuietDuration = d
	}

	return nil
}

type message struct {
	channelName string
	msg         string
}

type Strategy struct {
	*bbgo.Notifiability

	Config []StrategyConfig

	notifyC chan message
}

func (s *Strategy) UnmarshalJSON(data []byte) error {
	var c []StrategyConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("failed to unmarshal %s config: %w", s.ID(), err)
	}
	s.Config = c
	return nil
}

func (s *Strategy) ID() string {
	return ID
}

func (s *Strategy) CrossSubscribe(sessions map[string]*bbgo.ExchangeSession) {}

func (s *Strategy) startNotifier(ctx context.Context) {
	tk := time.NewTicker(8 * time.Hour)
	defer tk.Stop()

	s.Notify("i'm alive.")

	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			s.Notify("i'm still alive.")
		case m := <-s.notifyC:
			s.NotifyTo(m.channelName, m.msg)
		}
	}
}

func (s *Strategy) enqueueMessage(msg message) error {
	select {
	case s.notifyC <- msg:
		return nil
	default:
		return fmt.Errorf("msg channel is full")
	}
}

func (s *Strategy) CrossRun(ctx context.Context, _ bbgo.OrderExecutionRouter, sessions map[string]*bbgo.ExchangeSession) error {
	s.notifyC = make(chan message, 512)
	go s.startNotifier(ctx)

	for _, config := range s.Config {
		c := config
		source, ok := sessions[c.SourceExchange]
		if !ok {
			panic("exchange is not defined: " + c.SourceExchange)
		}
		target, ok := sessions[c.TargetExchange]
		if !ok {
			panic("exchange is not defined: " + c.TargetExchange)
		}

		targetStream := target.Stream
		targetStream.Subscribe(types.BookChannel, c.TargetExchangeMarket, types.SubscribeOptions{})
		targetBook := types.NewStreamBook(c.TargetExchangeMarket)
		targetBook.BindStream(targetStream)

		sourceStream := source.Stream
		sourceStream.SetPublicOnly()
		sourceStream.Subscribe(types.BookChannel, c.SourceExchangeMarket, types.SubscribeOptions{})
		sourceBook := types.NewStreamBook(c.SourceExchangeMarket)
		sourceBook.BindStream(sourceStream)

		checkLowerLimit := compare(lessEqual(c.SpreadLowerLimitBps), c.BelowLimitDuration)
		lowerLimitAlert := s.throttledNotifier(c.SlackChannelName, c.QuietDuration)

		checkUpperLimit := compare(greaterThan(c.SpreadUpperLimitBps), c.AboveLimitDuration)
		upperLimitAlert := s.throttledNotifier(c.SlackChannelName, c.QuietDuration)

		sourceBook.OnUpdate(func(sb *types.OrderBook) {
			if !sourceTargetReady(sb, targetBook) {
				return
			}

			spreadBps := toBps(sourceTargetSpread(c, sb, targetBook))

			checkLowerLimit(spreadBps, func() {
				msg := fmt.Sprintf("%s.\nspread %d bps < %d bps", c.LowerLimitMessage, spreadBps, c.SpreadLowerLimitBps)
				lowerLimitAlert(msg)
			})

			checkUpperLimit(spreadBps, func() {
				msg := fmt.Sprintf("%s.\nspread %d bps > %d bps", c.UpperLimitMessage, spreadBps, c.SpreadUpperLimitBps)
				upperLimitAlert(msg)
			})
		})
	}
	return nil
}

func (s *Strategy) throttledNotifier(channelName string, quietDuration time.Duration) func(msg string) {
	var lastNotifyTime time.Time
	return func(msg string) {
		now := time.Now()
		if now.Sub(lastNotifyTime) > quietDuration {
			if err := s.enqueueMessage(message{channelName: channelName, msg: msg}); err != nil {
				log.Errorf("failed to enqueue the msg to %s", channelName)
				return
			}
			lastNotifyTime = now
		}
	}
}

func toBps(spread float64) int64 {
	return int64((spread - 1) * 10000)
}

type predicate func(bps int64) bool

func lessEqual(threshold int64) predicate {
	return func(bps int64) bool {
		return bps <= threshold
	}
}

func greaterThan(threshold int64) predicate {
	return func(bps int64) bool {
		return bps > threshold
	}
}

func compare(p predicate, observeDuration time.Duration) func(spreadBps int64, callback func()) {
	var firstAlertTime time.Time
	return func(spreadBps int64, callback func()) {
		now := time.Now()
		if p(spreadBps) {
			if firstAlertTime == (time.Time{}) {
				firstAlertTime = now
			}

			if now.Sub(firstAlertTime) > observeDuration {
				callback()
			}
		} else {
			firstAlertTime = time.Time{}
		}
	}
}

func sourceTargetReady(sourceBook *types.OrderBook, targetBook *types.StreamOrderBook) bool {
	t := targetBook.Get()
	_, sb := sourceBook.BestBid()
	_, sa := sourceBook.BestAsk()
	_, tb := t.BestBid()
	_, ta := t.BestAsk()

	return sb && sa && tb && ta
}

func sourceTargetSpread(c StrategyConfig, sourceBook *types.OrderBook, targetBook *types.StreamOrderBook) float64 {
	t := targetBook.Get()
	sourcePrice := getPrice(sourceBook, c.SourceExchangeSide, c.SourceExchangeTakerFee)
	targetPrice := getPrice(&t, c.TargetExchangeSide, c.TargetExchangeTakerFee)

	return targetPrice / sourcePrice
}

// actually we don't care about the precision loss here so using float.
// bps is just a really small number.
func getPrice(book *types.OrderBook, side string, takerFee float64) float64 {
	s := strings.ToLower(strings.TrimSpace(side))

	var pv types.PriceVolume

	if s == "bid" {
		pv, _ = book.BestBid()
	} else {
		pv, _ = book.BestAsk()
		takerFee = -1 * takerFee
	}

	return pv.Price.Float64() * (1 + takerFee)
}
