package backtest

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/ycdesu/spreaddog/pkg/types"
)

type Stream struct {
	types.StandardStream

	exchange *Exchange
}

func (s *Stream) Connect(ctx context.Context) error {
	log.Infof("collecting backtest configurations...")

	loadedSymbols := map[string]struct{}{}
	loadedIntervals := map[types.Interval]struct{}{
		// 1m interval is required for the backtest matching engine
		types.Interval1m: {},
		types.Interval1d: {},
	}

	for _, sub := range s.Subscriptions {
		loadedSymbols[sub.Symbol] = struct{}{}

		switch sub.Channel {
		case types.KLineChannel:
			loadedIntervals[types.Interval(sub.Options.Interval)] = struct{}{}

		default:
			return fmt.Errorf("stream channel %s is not supported in backtest", sub.Channel)
		}
	}

	var symbols []string
	for symbol := range loadedSymbols {
		symbols = append(symbols, symbol)
	}

	var intervals []types.Interval
	for interval := range loadedIntervals {
		intervals = append(intervals, interval)
	}

	log.Infof("used symbols: %v and intervals: %v", symbols, intervals)

	go func() {
		s.EmitConnect()

		klineC, errC := s.exchange.srv.QueryKLinesCh(s.exchange.startTime, s.exchange.endTime, s.exchange, symbols, intervals)
		for k := range klineC {
			if k.Interval == types.Interval1m {
				matching, ok := s.exchange.matchingBooks[k.Symbol]
				if !ok {
					log.Errorf("matching book of %s is not initialized", k.Symbol)
				}
				matching.processKLine(k)
			}

			s.EmitKLineClosed(k)
		}

		if err := <-errC; err != nil {
			log.WithError(err).Error("backtest data feed error")
		}

		if err := s.Close(); err != nil {
			log.WithError(err).Error("stream close error")
		}
	}()

	return nil
}

func (s *Stream) SetPublicOnly() {
	return
}

func (s *Stream) Close() error {
	close(s.exchange.doneC)
	return nil
}
