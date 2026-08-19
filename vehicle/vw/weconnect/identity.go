package weconnect

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag"
	"golang.org/x/net/publicsuffix"
)

// Identity implements the VW We Connect device authorization login and the
// resulting refreshing token source.
type Identity struct {
	*request.Helper
	log            *util.Logger
	user, password string
}

// NewIdentity creates a We Connect identity for the given credentials.
func NewIdentity(log *util.Logger, user, password string) *Identity {
	return &Identity{
		Helper:   request.NewHelper(log),
		log:      log,
		user:     user,
		password: password,
	}
}

// deviceAuthResponse is the device authorization grant response
type deviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	Interval                int    `json:"interval"`
	ExpiresIn               int    `json:"expires_in"`
}

// Login runs the device authorization flow and returns a refreshing token source.
func (v *Identity) Login() (vag.TokenSource, error) {
	token, err := v.login()
	if err != nil {
		return nil, err
	}
	return vag.RefreshTokenSource(token, v.refresh), nil
}

func (v *Identity) login() (*vag.Token, error) {
	// 1. start the device authorization grant
	device, err := v.startDeviceFlow()
	if err != nil {
		return nil, fmt.Errorf("device authorization: %w", err)
	}

	verificationURI := device.VerificationURIComplete
	if verificationURI == "" {
		verificationURI = device.VerificationURI
	}
	if verificationURI == "" || device.DeviceCode == "" {
		return nil, errors.New("device authorization response incomplete")
	}

	// 2. drive the browser confirmation headlessly
	if err := v.browserRoute(verificationURI); err != nil {
		return nil, fmt.Errorf("confirmation: %w", err)
	}

	// 3. poll for the issued token
	interval := device.Interval
	if interval < 1 {
		interval = 5
	}
	expires := device.ExpiresIn
	if expires < 30 {
		expires = 330
	}

	return v.pollToken(device.DeviceCode, interval, expires)
}

func (v *Identity) startDeviceFlow() (deviceAuthResponse, error) {
	var res deviceAuthResponse

	data := url.Values{
		"client_id": {DeviceFlowClientID},
		"scope":     {ClientScope},
	}

	req, err := request.New(http.MethodPost, DeviceAuthorizationURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		"Accept":       request.JSONContent,
	})
	if err != nil {
		return res, err
	}

	err = v.DoJSON(req, &res)
	return res, err
}

// browserRoute performs the headless browser confirmation: it opens the
// verification page and walks the IDKit identifier/password/confirmation stages.
func (v *Identity) browserRoute(verificationURI string) error {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return err
	}

	v.Client.Jar = jar
	v.Client.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		// follow the identity redirect chain, stop at app schemes
		if req.URL.Scheme != "https" {
			return http.ErrUseLastResponse
		}
		return nil
	}
	defer func() {
		v.Client.Jar = nil
		v.Client.CheckRedirect = nil
	}()

	page, err := v.getPage(verificationURI)
	if err != nil {
		return err
	}

	// sticky client id carried across pages
	clientID := page.clientID
	if clientID == "" {
		clientID = DeviceFlowClientID
	}

	if page.stage == stageIdentifier {
		if page, err = v.postIdentifier(clientID, page); err != nil {
			return err
		}
		if page.stage != stagePassword {
			return fmt.Errorf("unexpected stage after identifier: %q", page.stage)
		}

		if page, err = v.postPassword(clientID, page); err != nil {
			return err
		}
		if page.stage != stageConfirm {
			return fmt.Errorf("unexpected stage after password: %q", page.stage)
		}
	}

	if page.stage != stageConfirm {
		return fmt.Errorf("unexpected initial stage: %q", page.stage)
	}

	return v.postConfirm(page)
}

func (v *Identity) getPage(uri string) (idkPage, error) {
	body, err := v.GetBody(uri)
	if err != nil {
		return idkPage{}, err
	}
	return parseIDKitPage(body)
}

// postForm submits an IDKit form and returns the resulting page.
func (v *Identity) postForm(uri string, data url.Values) (idkPage, error) {
	resp, err := v.PostForm(uri, data)
	if err != nil {
		return idkPage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return idkPage{}, errors.New(resp.Status)
	}

	if err := credentialsError(resp.Request.URL); err != nil {
		return idkPage{}, err
	}

	body, err := request.ReadBody(resp)
	if err != nil {
		return idkPage{}, err
	}

	page, err := parseIDKitPage(body)
	if err == nil && page.errorMsg != "" {
		err = errors.New(page.errorMsg)
	}
	return page, err
}

func (v *Identity) postIdentifier(clientID string, page idkPage) (idkPage, error) {
	uri := fmt.Sprintf(LoginIdentifierURL, clientID)
	return v.postForm(uri, url.Values{
		"_csrf":      {page.csrfToken},
		"relayState": {page.relayState},
		"hmac":       {page.hmac},
		"email":      {v.user},
	})
}

func (v *Identity) postPassword(clientID string, page idkPage) (idkPage, error) {
	uri := fmt.Sprintf(LoginAuthenticateURL, clientID)
	return v.postForm(uri, url.Values{
		"_csrf":      {page.csrfToken},
		"relayState": {page.relayState},
		"hmac":       {page.hmac},
		"email":      {v.user},
		"password":   {v.password},
	})
}

func (v *Identity) postConfirm(page idkPage) error {
	if page.userCode == "" {
		return errors.New("confirmation page missing user code")
	}

	q := url.Values{
		"relayState": {page.relayState},
		"user_id":    {page.userID},
		"hmac":       {page.hmac},
	}
	uri := fmt.Sprintf(CodeConfirmationURL, page.clientID, page.userCode) + "?" + q.Encode()

	resp, err := v.PostForm(uri, url.Values{
		"_csrf":                {page.csrfToken},
		"client_identity_name": {page.clientIdentityName},
		"allow":                {""},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return errors.New(resp.Status)
	}

	return credentialsError(resp.Request.URL)
}

// credentialsError reports an error= parameter in the given url as a login error.
func credentialsError(u *url.URL) error {
	if u == nil {
		return nil
	}
	if e := u.Query().Get("error"); e != "" {
		return fmt.Errorf("login rejected: %s", e)
	}
	return nil
}

// pollToken polls the token endpoint until the device authorization is confirmed.
func (v *Identity) pollToken(deviceCode string, interval, expiresIn int) (*vag.Token, error) {
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	wait := time.Duration(interval) * time.Second

	for time.Now().Before(deadline) {
		time.Sleep(wait)

		data := url.Values{
			"grant_type":  {DeviceCodeGrant},
			"device_code": {deviceCode},
			"client_id":   {DeviceFlowClientID},
		}

		resp, err := v.PostForm(TokenURL, data)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			var token vag.Token
			err := json.NewDecoder(resp.Body).Decode(&token)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}
			if err := token.Error(); err != nil {
				return nil, err
			}
			return &token, nil
		}

		// transient server errors - keep polling
		if resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			continue
		}

		var res struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()

		switch res.Error {
		case "authorization_pending":
			// keep polling
		case "slow_down":
			wait += 5 * time.Second
		case "":
			return nil, fmt.Errorf("token polling failed: %s", resp.Status)
		default:
			return nil, fmt.Errorf("token polling failed: %s", res.Error)
		}
	}

	return nil, errors.New("token polling timed out")
}

// refresh exchanges the refresh token for a new access token.
func (v *Identity) refresh(token *vag.Token) (*vag.Token, error) {
	if token == nil || token.RefreshToken == "" {
		return nil, errors.New("missing refresh token")
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"client_id":     {DeviceFlowClientID},
	}

	resp, err := v.PostForm(TokenURL, data)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, request.NewStatusError(resp)
	}

	var res vag.Token
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if err := res.Error(); err != nil {
		return nil, err
	}

	return &res, nil
}
