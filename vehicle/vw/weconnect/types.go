package weconnect

import "time"

// VehiclesResponse is the /vehicle/v2/vehicles response
type VehiclesResponse struct {
	Data []Vehicle `json:"data"`
}

// Vehicle describes a vehicle linked to the account
type Vehicle struct {
	VIN      string `json:"vin"`
	Nickname string `json:"nickname"`
	Model    string `json:"model"`
}

// StatusResponse is the relevant subset of the /selectivestatus response
type StatusResponse struct {
	Charging struct {
		BatteryStatus    timedValue[batteryStatus]    `json:"batteryStatus"`
		ChargingStatus   timedValue[chargingStatus]   `json:"chargingStatus"`
		PlugStatus       timedValue[plugStatus]       `json:"plugStatus"`
		ChargingSettings timedValue[chargingSettings] `json:"chargingSettings"`
	} `json:"charging"`
	Measurements struct {
		OdometerStatus timedValue[odometerStatus] `json:"odometerStatus"`
		RangeStatus    timedValue[rangeStatus]    `json:"rangeStatus"`
	} `json:"measurements"`
}

// timedValue wraps the {"value": {...}} envelope used throughout the api
type timedValue[T any] struct {
	Value T `json:"value"`
}

type batteryStatus struct {
	CurrentSocPct           int       `json:"currentSOC_pct"`
	CruisingRangeElectricKm int       `json:"cruisingRangeElectric_km"`
	CarCapturedTimestamp    time.Time `json:"carCapturedTimestamp"`
}

type chargingStatus struct {
	ChargingState                      string    `json:"chargingState"`
	ChargePowerKw                      float64   `json:"chargePower_kW"`
	RemainingChargingTimeToCompleteMin int       `json:"remainingChargingTimeToComplete_min"`
	CarCapturedTimestamp               time.Time `json:"carCapturedTimestamp"`
}

type plugStatus struct {
	PlugConnectionState string `json:"plugConnectionState"`
}

type chargingSettings struct {
	TargetSocPct int `json:"targetSOC_pct"`
}

type odometerStatus struct {
	Odometer int `json:"odometer"`
}

type rangeStatus struct {
	ElectricRange int `json:"electricRange"`
}
