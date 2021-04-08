package service

import (
	"context"
	"errors"
	"time"

	"github.com/ycdesu/spreaddog/pkg/types"
)

var ErrNotImplemented = errors.New("not implemented")
var ErrExchangeRewardServiceNotImplemented = errors.New("exchange does not implement ExchangeRewardService interface")

type SyncService struct {
	TradeService    *TradeService
	OrderService    *OrderService
	RewardService   *RewardService
	WithdrawService *WithdrawService
	DepositService  *DepositService
}

// SyncSessionSymbols syncs the trades from the given exchange session
func (s *SyncService) SyncSessionSymbols(ctx context.Context, exchange types.Exchange, startTime time.Time, symbols ...string) error {
	for _, symbol := range symbols {
		if err := s.TradeService.Sync(ctx, exchange, symbol); err != nil {
			return err
		}

		if err := s.OrderService.Sync(ctx, exchange, symbol, startTime); err != nil {
			return err
		}
	}

	if err := s.DepositService.Sync(ctx, exchange); err != nil {
		if err != ErrNotImplemented {
			return err
		}
	}

	if err := s.WithdrawService.Sync(ctx, exchange); err != nil {
		if err != ErrNotImplemented {
			return err
		}
	}

	if err := s.RewardService.Sync(ctx, exchange); err != nil {
		if err != ErrExchangeRewardServiceNotImplemented {
			return err
		}
	}

	return nil
}
