package weconnect

import (
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider implements the vehicle api on top of the We Connect status
type Provider struct {
	statusG func() (StatusResponse, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	return &Provider{
		statusG: util.Cached(func() (StatusResponse, error) {
			return api.Status(vin)
		}, cache),
	}
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return float64(res.Charging.BatteryStatus.Value.CurrentSocPct), nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if err != nil {
		return status, err
	}

	if strings.EqualFold(res.Charging.PlugStatus.Value.PlugConnectionState, "connected") {
		status = api.StatusB
	}

	if strings.EqualFold(res.Charging.ChargingStatus.Value.ChargingState, "charging") {
		status = api.StatusC
	}

	return status, nil
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if r := res.Measurements.RangeStatus.Value.ElectricRange; r > 0 {
		return int64(r), nil
	}
	if r := res.Charging.BatteryStatus.Value.CruisingRangeElectricKm; r > 0 {
		return int64(r), nil
	}

	return 0, api.ErrNotAvailable
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if o := res.Measurements.OdometerStatus.Value.Odometer; o > 0 {
		return float64(o), nil
	}

	return 0, api.ErrNotAvailable
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err != nil {
		return time.Time{}, err
	}

	cs := res.Charging.ChargingStatus.Value
	if cs.RemainingChargingTimeToCompleteMin > 0 {
		ts := cs.CarCapturedTimestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		return ts.Add(time.Duration(cs.RemainingChargingTimeToCompleteMin) * time.Minute), nil
	}

	return time.Time{}, api.ErrNotAvailable
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if t := res.Charging.ChargingSettings.Value.TargetSocPct; t > 0 {
		return int64(t), nil
	}

	return 0, api.ErrNotAvailable
}
