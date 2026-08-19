package weconnect

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// stage identifies the IDKit login page as reported in templateModel.template.
// The values match the VW identity kit (see volkswagencarnet vw_login/_idkit.py).
type stage string

const (
	stageIdentifier stage = "loginIdentifier"     // e-mail entry
	stagePassword   stage = "loginAuthenticate"   // password entry
	stageConfirm    stage = "codeConfirmation"    // device code confirmation
	stageSuccess    stage = "verificationSuccess" // login completed
)

// idkPage holds the fields extracted from a single IDKit page's window._IDK
// object. Not every field is present on every page; the flow carries the sticky
// ones (client id, user code, ...) across pages.
type idkPage struct {
	stage              stage
	csrfToken          string
	hmac               string
	relayState         string
	clientID           string
	clientIdentityName string
	userCode           string
	userID             string
	email              string
	errorMsg           string
}

// templateModel mirrors the relevant subset of window._IDK.templateModel.
type templateModel struct {
	Template               string `json:"template"`
	Hmac                   string `json:"hmac"`
	RelayState             string `json:"relayState"`
	PostAction             string `json:"postAction"`
	IdentifierURL          string `json:"identifierUrl"`
	URL                    string `json:"url"`
	ClientIdentityName     string `json:"clientIdentityName"`
	ClientLegalEntityModel struct {
		ClientID string `json:"clientId"`
	} `json:"clientLegalEntityModel"`
	EmailPasswordForm struct {
		Email string `json:"email"`
	} `json:"emailPasswordForm"`
	Error errorField `json:"error"`
}

// errorField decodes templateModel.error which may be null, a plain string or a
// nested object depending on the page.
type errorField string

func (e *errorField) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	switch {
	case s == "" || s == "null":
		*e = ""
	case strings.HasPrefix(s, `"`):
		*e = errorField(strings.Trim(s, `"`))
	default:
		// object form - keep the raw text as a best-effort message
		*e = errorField(s)
	}
	return nil
}

var csrfRe = regexp.MustCompile(`csrf_token['"]?\s*:\s*['"]([^'"]+)['"]`)

// parseIDKitPage extracts the login state from an IDKit HTML page. The relevant
// data is embedded in a window._IDK JavaScript object; its templateModel member
// is compact JSON that we scan out and decode.
func parseIDKitPage(body []byte) (idkPage, error) {
	var res idkPage

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return res, err
	}

	var script string
	doc.Find("script").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if strings.Contains(s.Text(), "window._IDK") {
			script = s.Text()
			return false
		}
		return true
	})

	if script == "" {
		return res, errors.New("window._IDK not found")
	}

	tm, err := extractTemplateModel(script)
	if err != nil {
		return res, err
	}

	res.stage = stage(tm.Template)
	res.hmac = tm.Hmac
	res.relayState = tm.RelayState
	res.clientID = tm.ClientLegalEntityModel.ClientID
	res.clientIdentityName = tm.ClientIdentityName
	res.email = tm.EmailPasswordForm.Email
	res.errorMsg = string(tm.Error)

	if m := csrfRe.FindStringSubmatch(script); m != nil {
		res.csrfToken = m[1]
	}

	// the confirmation page carries client id / user code / user id in the device url
	if tm.URL != "" {
		clientID, userCode, q := parseDeviceURL(tm.URL)
		if clientID != "" {
			res.clientID = clientID
		}
		res.userCode = userCode
		if res.userID == "" {
			res.userID = q.Get("user_id")
		}
		if res.relayState == "" {
			res.relayState = q.Get("relayState")
		}
		if res.hmac == "" {
			res.hmac = q.Get("hmac")
		}
	}

	return res, nil
}

// extractTemplateModel scans the window._IDK script for the templateModel member
// and decodes its (compact JSON) object.
func extractTemplateModel(script string) (templateModel, error) {
	var tm templateModel

	idx := strings.Index(script, "templateModel")
	if idx < 0 {
		return tm, errors.New("templateModel not found")
	}

	obj, err := extractJSONObject(script, idx)
	if err != nil {
		return tm, err
	}

	if err := unmarshalRelaxed(obj, &tm); err != nil {
		return tm, fmt.Errorf("templateModel: %w", err)
	}

	return tm, nil
}

// parseDeviceURL splits a device confirmation url of the form
// device/{client_id}/{user_code}?relayState=...&user_id=...&hmac=... into its
// client id, user code and remaining query parameters.
func parseDeviceURL(raw string) (clientID, userCode string, q url.Values) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", url.Values{}
	}

	parts := strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
	if len(parts) >= 3 && parts[len(parts)-3] == "device" {
		clientID = parts[len(parts)-2]
		userCode = parts[len(parts)-1]
	}

	return clientID, userCode, u.Query()
}
