package lunar

import (
	"context"
	"github.com/mlsorensen/goscale"
	"time"
)

func init() {
	// Register with a distinct name, "MOCK", so it can be requested specifically.
	goscale.Register("LUNAR", New)
}

// This line is the compile-time check. It will fail to compile if
// *MockScale ever stops satisfying the goscale.Scale interface.
var _ goscale.Scale = (*LunarScale)(nil)

type LunarScale struct{}

func New() goscale.Scale {
	return LunarScale{}
}

func (l LunarScale) Connect(ctx context.Context) (<-chan goscale.WeightUpdate, error) {
	//TODO implement me
	panic("implement me")
}

func (l LunarScale) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (l LunarScale) Tare(ctx context.Context, blocking bool) error {
	//TODO implement me
	panic("implement me")
}

func (l LunarScale) SetSleepTimeout(ctx context.Context, d time.Duration) error {
	//TODO implement me
	panic("implement me")
}

func (l LunarScale) ReadBatteryChargePercent(ctx context.Context) (uint8, error) {
	//TODO implement me
	panic("implement me")
}
