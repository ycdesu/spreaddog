package cmd

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/ycdesu/spreaddog/pkg/bbgo"
	"github.com/ycdesu/spreaddog/pkg/cmd/cmdutil"
	"github.com/ycdesu/spreaddog/pkg/types"
)

// go run ./cmd/bbgo tradesCmd --session=ftx --symbol="BTC/USD"
var tradesCmd = &cobra.Command{
	Use:          "trades",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}

		if len(configFile) == 0 {
			return errors.New("--config option is required")
		}

		// if config file exists, use the config loaded from the config file.
		// otherwise, use a empty config object
		var userConfig *bbgo.Config
		if _, err := os.Stat(configFile); err == nil {
			// load successfully
			userConfig, err = bbgo.Load(configFile, false)
			if err != nil {
				return err
			}
		} else if os.IsNotExist(err) {
			// config file doesn't exist
			userConfig = &bbgo.Config{}
		} else {
			// other error
			return err
		}

		environ := bbgo.NewEnvironment()

		if err := environ.ConfigureExchangeSessions(userConfig); err != nil {
			return err
		}

		sessionName, err := cmd.Flags().GetString("session")
		if err != nil {
			return err
		}

		session, ok := environ.Session(sessionName)
		if !ok {
			return fmt.Errorf("session %s not found", sessionName)
		}

		symbol, err := cmd.Flags().GetString("symbol")
		if err != nil {
			return fmt.Errorf("can't get the symbol from flags: %w", err)
		}
		if symbol == "" {
			return fmt.Errorf("symbol is not found")
		}

		until := time.Now()
		since := until.Add(-3 * 24 * time.Hour)
		trades, err := session.Exchange.QueryTrades(ctx, symbol, &types.TradeQueryOptions{
			StartTime:   &since,
			EndTime:     &until,
			Limit:       100,
			LastTradeID: 0,
		})
		if err != nil {
			return err
		}

		log.Infof("%d trades", len(trades))
		for _, t := range trades {
			log.Infof("trade: %+v", t)
		}
		return nil
	},
}

// go run ./cmd/bbgo tradeupdate --session=ftx
var tradeUpdateCmd = &cobra.Command{
	Use: "tradeupdate",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}

		if len(configFile) == 0 {
			return errors.New("--config option is required")
		}

		// if config file exists, use the config loaded from the config file.
		// otherwise, use a empty config object
		var userConfig *bbgo.Config
		if _, err := os.Stat(configFile); err == nil {
			// load successfully
			userConfig, err = bbgo.Load(configFile, false)
			if err != nil {
				return err
			}
		} else if os.IsNotExist(err) {
			// config file doesn't exist
			userConfig = &bbgo.Config{}
		} else {
			// other error
			return err
		}

		environ := bbgo.NewEnvironment()

		if err := environ.ConfigureExchangeSessions(userConfig); err != nil {
			return err
		}

		sessionName, err := cmd.Flags().GetString("session")
		if err != nil {
			return err
		}

		session, ok := environ.Session(sessionName)
		if !ok {
			return fmt.Errorf("session %s not found", sessionName)
		}

		s := session.Exchange.NewStream()
		s.OnTradeUpdate(func(trade types.Trade) {
			log.Infof("trade update: %+v", trade)
		})

		log.Infof("connecting...")
		if err := s.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to %s", sessionName)
		}
		log.Infof("connected")

		cmdutil.WaitForSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
		return nil
	},
}

func init() {
	tradesCmd.Flags().String("session", "", "the exchange session name for querying balances")
	tradesCmd.Flags().String("symbol", "", "the trading pair, like btcusdt")

	tradeUpdateCmd.Flags().String("session", "", "the exchange session name for querying balances")

	RootCmd.AddCommand(tradesCmd)
	RootCmd.AddCommand(tradeUpdateCmd)
}
