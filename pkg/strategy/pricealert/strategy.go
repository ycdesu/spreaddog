package pricealert

import (
	"context"
	"math"

	"github.com/ycdesu/spreaddog/pkg/bbgo"
	"github.com/ycdesu/spreaddog/pkg/types"
)

const ID = "pricealert"

func init() {
	bbgo.RegisterStrategy(ID, &Strategy{})
}

type Strategy struct {
	// The notification system will be injected into the strategy automatically.
	bbgo.Notifiability

	// These fields will be filled from the config file (it translates YAML to JSON)
	Symbol    string  `json:"symbol"`
	Interval  string  `json:"interval"`
	MinChange float64 `json:"minChange"`
}

func (s *Strategy) ID() string {
	return ID
}

func (s *Strategy) Subscribe(session *bbgo.ExchangeSession) {
	session.Subscribe(types.KLineChannel, s.Symbol, types.SubscribeOptions{Interval: s.Interval})
}

func (s *Strategy) Run(ctx context.Context, orderExecutor bbgo.OrderExecutor, session *bbgo.ExchangeSession) error {
	session.Stream.OnKLine(func(kline types.KLine) {
		market, ok := session.Market(kline.Symbol)
		if !ok {
			return
		}

		if math.Abs(kline.GetChange()) > s.MinChange {
			if channel, ok := s.RouteSymbol(s.Symbol); ok {
				s.NotifyTo(channel, "%s hit price %s, change %f", s.Symbol, market.FormatPrice(kline.Close), kline.GetChange())
			} else {
				s.Notify("%s hit price %s, change %f", s.Symbol, market.FormatPrice(kline.Close), kline.GetChange())
			}
		}
	})
	return nil
}
