package weconnect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const identifierPage = `<html><head>
<meta name="_csrf" content="ignored">
</head><body>
<script>
window._IDK = {
	templateModel: {"clientLegalEntityModel":{"clientId":"650d46ca-2475-4384-85c2-6af3bf3d52f1@apps_vw-dilab_com","clientAppName":"WeConnect"},"template":"loginIdentifier","hmac":"abc123hmac","useClientRendering":true,"emailPasswordForm":{"email":"user@example.com","password":null},"error":null,"relayState":"relay-state-1","postAction":"login/identifier","identifierUrl":"login/identifier"},
	currentLocale: 'de-DE',
	csrf_parameterName: '_csrf',
	csrf_token: 'csrf-token-identifier',
	baseUrl: 'https://identity.vwgroup.io'
}
</script>
</body></html>`

const passwordPage = `<html><body>
<script>
window._IDK = {
	templateModel: {"template":"loginAuthenticate","hmac":"pwhmac456","emailPasswordForm":{"email":"user@example.com","password":null},"error":null,"relayState":"relay-state-2","postAction":"login/authenticate","identifierUrl":"login/identifier"},
	csrf_parameterName: '_csrf',
	csrf_token: 'csrf-token-password'
}
</script>
</body></html>`

const confirmPage = `<html><body>
<script>
window._IDK = {
	templateModel: {"clientLegalEntityModel":{"clientId":"650d46ca-2475-4384-85c2-6af3bf3d52f1@apps_vw-dilab_com"},"template":"codeConfirmation","clientIdentityName":"WeConnect ID","hmac":"confirmhmac789","relayState":"relay-state-3","url":"device/650d46ca-2475-4384-85c2-6af3bf3d52f1@apps_vw-dilab_com/ABCD-1234?relayState=relay-state-3&user_id=user-id-99&hmac=confirmhmac789"},
	csrf_parameterName: '_csrf',
	csrf_token: 'csrf-token-confirm'
}
</script>
</body></html>`

func TestParseIdentifierPage(t *testing.T) {
	p, err := parseIDKitPage([]byte(identifierPage))
	require.NoError(t, err)

	require.Equal(t, stageIdentifier, p.stage)
	require.Equal(t, "csrf-token-identifier", p.csrfToken)
	require.Equal(t, "abc123hmac", p.hmac)
	require.Equal(t, "relay-state-1", p.relayState)
	require.Equal(t, "650d46ca-2475-4384-85c2-6af3bf3d52f1@apps_vw-dilab_com", p.clientID)
	require.Equal(t, "user@example.com", p.email)
	require.Empty(t, p.errorMsg)
}

func TestParsePasswordPage(t *testing.T) {
	p, err := parseIDKitPage([]byte(passwordPage))
	require.NoError(t, err)

	require.Equal(t, stagePassword, p.stage)
	require.Equal(t, "csrf-token-password", p.csrfToken)
	require.Equal(t, "pwhmac456", p.hmac)
	require.Equal(t, "relay-state-2", p.relayState)
}

func TestParseConfirmPage(t *testing.T) {
	p, err := parseIDKitPage([]byte(confirmPage))
	require.NoError(t, err)

	require.Equal(t, stageConfirm, p.stage)
	require.Equal(t, "csrf-token-confirm", p.csrfToken)
	require.Equal(t, "confirmhmac789", p.hmac)
	require.Equal(t, "relay-state-3", p.relayState)
	require.Equal(t, "WeConnect ID", p.clientIdentityName)
	require.Equal(t, "650d46ca-2475-4384-85c2-6af3bf3d52f1@apps_vw-dilab_com", p.clientID)
	require.Equal(t, "ABCD-1234", p.userCode)
	require.Equal(t, "user-id-99", p.userID)
}

func TestParseCredentialsError(t *testing.T) {
	const errPage = `<html><body><script>
window._IDK = {
	templateModel: {"template":"loginAuthenticate","hmac":"h","relayState":"r","error":"login.errors.password_invalid"},
	csrf_token: 'c'
}
</script></body></html>`

	p, err := parseIDKitPage([]byte(errPage))
	require.NoError(t, err)
	require.Equal(t, "login.errors.password_invalid", p.errorMsg)
}

func TestParseDeviceURL(t *testing.T) {
	clientID, userCode, q := parseDeviceURL("device/foo@apps_vw-dilab_com/WXYZ-9876?relayState=rs&user_id=uid&hmac=hm")
	require.Equal(t, "foo@apps_vw-dilab_com", clientID)
	require.Equal(t, "WXYZ-9876", userCode)
	require.Equal(t, "uid", q.Get("user_id"))
	require.Equal(t, "rs", q.Get("relayState"))
	require.Equal(t, "hm", q.Get("hmac"))
}
