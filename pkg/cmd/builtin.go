package cmd

// import built-in strategies
import (
	_ "github.com/ycdesu/spreaddog/pkg/strategy/bollgrid"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/buyandhold"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/flashcrash"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/gap"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/grid"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/mirrormaker"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/pricealert"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/spreadmonitor"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/support"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/swing"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/trailingstop"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/xmaker"
	_ "github.com/ycdesu/spreaddog/pkg/strategy/xpuremaker"
)
