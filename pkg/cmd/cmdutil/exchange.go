package cmdutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/ycdesu/spreaddog/pkg/exchange/binance"
	"github.com/ycdesu/spreaddog/pkg/exchange/ftx"
	"github.com/ycdesu/spreaddog/pkg/exchange/max"
	"github.com/ycdesu/spreaddog/pkg/types"
)

func NewExchangeStandard(n types.ExchangeName, key, secret, subAccount string) (types.Exchange, error) {
	switch n {

	case types.ExchangeFTX:
		return ftx.NewExchange(key, secret, subAccount), nil

	case types.ExchangeBinance:
		return binance.New(key, secret), nil

	case types.ExchangeMax:
		return max.New(key, secret), nil

	default:
		return nil, fmt.Errorf("unsupported exchange: %v", n)

	}
}

func NewExchangeWithEnvVarPrefix(n types.ExchangeName, varPrefix string) (types.Exchange, error) {
	if len(varPrefix) == 0 {
		varPrefix = n.String()
	}

	varPrefix = strings.ToUpper(varPrefix)

	key := os.Getenv(varPrefix + "_API_KEY")
	secret := os.Getenv(varPrefix + "_API_SECRET")
	if len(key) == 0 || len(secret) == 0 {
		return nil, fmt.Errorf("can not initialize exchange %s: empty key or secret, env var prefix: %s", n, varPrefix)
	}

	subAccount := os.Getenv(varPrefix + "_SUBACCOUNT")
	return NewExchangeStandard(n, key, secret, subAccount)
}

// NewExchange constructor exchange object from viper config.
func NewExchange(n types.ExchangeName) (types.Exchange, error) {
	return NewExchangeWithEnvVarPrefix(n, "")
}
