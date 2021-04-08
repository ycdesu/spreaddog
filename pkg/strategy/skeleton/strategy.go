package skeleton

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/ycdesu/spreaddog/pkg/bbgo"
	"github.com/ycdesu/spreaddog/pkg/types"
)

const ID = "skeleton"

func init() {
	bbgo.RegisterStrategy(ID, &Strategy{})
}

type Strategy struct {
	Symbol string `json:"symbol"`

	types.Market
}

func (s *Strategy) ID() string {
	return ID
}


func (s *Strategy) Subscribe(session *bbgo.ExchangeSession) {
	session.Subscribe(types.KLineChannel, s.Symbol, types.SubscribeOptions{Interval: "1m"})
}

func (s *Strategy) Run(ctx context.Context, orderExecutor bbgo.OrderExecutor, session *bbgo.ExchangeSession) error {
	session.Stream.OnKLineClosed(func(kline types.KLine) {
		quoteBalance, ok := session.Account.Balance(s.Market.QuoteCurrency)
		if !ok {
			return
		}
		_ = quoteBalance

		_, err := orderExecutor.SubmitOrders(ctx, types.SubmitOrder{
			Symbol:   kline.Symbol,
			Side:     types.SideTypeBuy,
			Type:     types.OrderTypeMarket,
			Quantity: 0.01,
		})

		if err != nil {
			log.WithError(err).Error("submit order error")
		}
	})

	return nil
}
