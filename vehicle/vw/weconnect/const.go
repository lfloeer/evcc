package weconnect

// VW We Connect device authorization flow and BFF endpoints.
//
// The login follows the OAuth 2.0 device authorization grant (RFC 8628) against
// the VW group identity service, driving the browser confirmation headlessly.
// This mirrors the approach of the volkswagencarnet project (v5.5.1, vw_login).
const (
	// IdentityBase is the VW group identity service
	IdentityBase = "https://identity.vwgroup.io"

	// DeviceAuthorizationURL starts the device authorization grant
	DeviceAuthorizationURL = IdentityBase + "/oidc/v1/device_authorization"
	// TokenURL issues and refreshes tokens
	TokenURL = IdentityBase + "/oidc/v1/token"

	// LoginIdentifierURL is the e-mail (identifier) submission endpoint
	LoginIdentifierURL = IdentityBase + "/signin-service/v1/%s/login/identifier"
	// LoginAuthenticateURL is the password submission endpoint
	LoginAuthenticateURL = IdentityBase + "/signin-service/v1/%s/login/authenticate"
	// CodeConfirmationURL confirms the device user code
	CodeConfirmationURL = IdentityBase + "/signin-service/v1/device/%s/%s"

	// DeviceFlowClientID is the We Connect device flow client
	DeviceFlowClientID = "650d46ca-2475-4384-85c2-6af3bf3d52f1@apps_vw-dilab_com"
	// ClientScope are the OIDC scopes requested for the device flow
	ClientScope = "openid profile badge cars dealers vin offline_access"

	// DeviceCodeGrant is the device authorization grant type
	DeviceCodeGrant = "urn:ietf:params:oauth:grant-type:device_code"

	// BaseAPI is the CARIAD backend-for-frontend serving We Connect vehicle data
	BaseAPI = "https://emea.bff.cariad.digital"

	// UserAgent identifies the We Connect app
	UserAgent = "Volkswagen/3.61.0-android/14"
	// AndroidPackage is the We Connect Android package name
	AndroidPackage = "com.volkswagen.weconnect"
)
