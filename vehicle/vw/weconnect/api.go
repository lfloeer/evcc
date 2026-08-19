package weconnect

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// API is the We Connect CARIAD backend-for-frontend client
type API struct {
	*request.Helper
}

// NewAPI creates a We Connect api client authenticated with the given token source
func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   v.Client.Transport,
	}

	return v
}

// headers returns the default request headers expected by the api
func headers() map[string]string {
	return map[string]string{
		"Accept":                 request.JSONContent,
		"User-Agent":             UserAgent,
		"x-android-package-name": AndroidPackage,
	}
}

func (v *API) getJSON(uri string, res any) error {
	req, err := request.New(http.MethodGet, uri, nil, headers())
	if err != nil {
		return err
	}
	return v.DoJSON(req, res)
}

// Vehicles returns the vehicles linked to the account
func (v *API) Vehicles() ([]Vehicle, error) {
	var res VehiclesResponse
	uri := fmt.Sprintf("%s/vehicle/v2/vehicles", BaseAPI)
	err := v.getJSON(uri, &res)
	return res.Data, err
}

// Status returns the selective status for the given vehicle
func (v *API) Status(vin string) (StatusResponse, error) {
	var res StatusResponse
	jobs := strings.Join([]string{"charging", "measurements"}, ",")
	uri := fmt.Sprintf("%s/vehicle/v1/vehicles/%s/selectivestatus?jobs=%s", BaseAPI, vin, jobs)
	err := v.getJSON(uri, &res)
	return res, err
}
