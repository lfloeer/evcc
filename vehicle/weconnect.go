package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vw/weconnect"
)

// WeConnect is an api.Vehicle implementation for VW We Connect cars.
//
// It authenticates via the VW group device authorization flow (see
// vehicle/vw/weconnect) and reads vehicle data from the CARIAD backend.
type WeConnect struct {
	*embed
	*weconnect.Provider
}

func init() {
	registry.Add("weconnect", NewWeConnectFromConfig)
}

// NewWeConnectFromConfig creates a new vehicle
func NewWeConnectFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &WeConnect{
		embed: &cc.embed,
	}

	log := util.NewLogger("weconnect").Redact(cc.User, cc.Password, cc.VIN)

	ts, err := weconnect.NewIdentity(log, cc.User, cc.Password).Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	api := weconnect.NewAPI(log, ts)
	api.Client.Timeout = cc.Timeout

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v weconnect.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)
	if err != nil {
		return nil, err
	}

	if vehicle.Nickname != "" {
		v.fromVehicle(vehicle.Nickname, 0)
	} else {
		v.fromVehicle(vehicle.Model, 0)
	}

	v.Provider = weconnect.NewProvider(api, vehicle.VIN, cc.Cache)

	return v, nil
}
