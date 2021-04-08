package service

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ycdesu/spreaddog/pkg/types"
)

type WithdrawService struct {
	DB *sqlx.DB
}

// Sync syncs the withdraw records into db
func (s *WithdrawService) Sync(ctx context.Context, ex types.Exchange) error {
	txnIDs := map[string]struct{}{}

	// query descending
	records, err := s.QueryLast(ex.Name(), 10)
	if err != nil {
		return err
	}

	for _, record := range records {
		txnIDs[record.TransactionID] = struct{}{}
	}

	transferApi, ok := ex.(types.ExchangeTransferService)
	if !ok {
		return ErrNotImplemented
	}

	since := time.Time{}
	if len(records) > 0 {
		since = records[len(records)-1].ApplyTime.Time()
	}

	// asset "" means all assets
	withdraws, err := transferApi.QueryWithdrawHistory(ctx, "", since, time.Now())
	if err != nil {
		return err
	}

	for _, withdraw := range withdraws {
		if _, exists := txnIDs[withdraw.TransactionID]; exists {
			continue
		}

		if err := s.Insert(withdraw); err != nil {
			return err
		}
	}

	return nil
}

func (s *WithdrawService) QueryLast(ex types.ExchangeName, limit int) ([]types.Withdraw, error) {
	sql := "SELECT * FROM `withdraws` WHERE `exchange` = :exchange ORDER BY `time` DESC LIMIT :limit"
	rows, err := s.DB.NamedQuery(sql, map[string]interface{}{
		"exchange": ex,
		"limit":    limit,
	})
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	return s.scanRows(rows)
}

func (s *WithdrawService) Query(exchangeName types.ExchangeName) ([]types.Withdraw, error) {
	args := map[string]interface{}{
		"exchange": exchangeName,
	}
	sql := "SELECT * FROM `withdraws` WHERE `exchange` = :exchange ORDER BY `time` ASC"
	rows, err := s.DB.NamedQuery(sql, args)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return s.scanRows(rows)
}

func (s *WithdrawService) scanRows(rows *sqlx.Rows) (withdraws []types.Withdraw, err error) {
	for rows.Next() {
		var withdraw types.Withdraw
		if err := rows.StructScan(&withdraw); err != nil {
			return withdraws, err
		}

		withdraws = append(withdraws, withdraw)
	}

	return withdraws, rows.Err()
}

func (s *WithdrawService) Insert(withdrawal types.Withdraw) error {
	sql := `INSERT INTO withdraws (exchange, asset, network, address, amount, txn_id, txn_fee, time)
			VALUES (:exchange, :asset, :network, :address, :amount, :txn_id, :txn_fee, :time)`
	_, err := s.DB.NamedExec(sql, withdrawal)
	return err
}
