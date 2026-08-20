package domain

import "fmt"

type ErrorCode string

const (
	ErrNotAvailable         ErrorCode = "SERVICE_NOT_AVAILABLE"
	ErrUnexpectedUp         ErrorCode = "SECURITY_ALERT_UNEXPECTED_UP"
	ErrUnexpectedDown       ErrorCode = "SERVICE_UNEXPECTED_DOWN"
	ErrWrongRedirect        ErrorCode = "WRONG_REDIRECT_BEHAVIOR"
	ErrNoClients            ErrorCode = "NO_CLIENTS_AVAILABLE"
	ErrInvalidCertificate   ErrorCode = "INVALID_CERTIFICATE"
	ErrWebsocketUpgradeFail ErrorCode = "WEBSOCKET_UPGRADE_FAILED"
)

func RenderError(code ErrorCode, family string, details ...interface{}) string {
	prefix := ""
	if family != "" {
		prefix = fmt.Sprintf("[%s] ", family)
	}

	switch code {
	case ErrNotAvailable:
		return prefix + "Service is completely offline (neither reachable via IPv4 nor IPv6)"

	case ErrUnexpectedUp:
		// details[0] is the list of leaking interfaces
		return fmt.Sprintf("🚨 SECURITY ALERT: Service should be isolated, but responds via: %v", details[0])

	case ErrUnexpectedDown:
		// details[0] is the received HTTP status code
		return fmt.Sprintf("%sExpected status UP, but got HTTP code %v", prefix, details[0])

	case ErrWrongRedirect:
		// details[0] is the expectation (bool), details[1] is the actual status code
		expected := "HTTPS redirect"
		if len(details) > 0 && !details[0].(bool) {
			expected = "NO HTTPS redirect"
		}

		return fmt.Sprintf("%sExpected %s, but server returned HTTP code %v", prefix, expected, details[1])

	case ErrInvalidCertificate:
		// details[0] is the expected certificate status (e.g., "valid" or "invalid")
		return fmt.Sprintf("%sCertificate does not meet expectation (Expected: %v)", prefix, details[0])

	case ErrWebsocketUpgradeFail:
		// details[0] is the received HTTP status code
		return fmt.Sprintf(
			"%sHTTP connection established, but protocol upgrade failed (Status: %v). Check proxy configuration!",
			prefix,
			details[0],
		)

	case ErrNoClients:
		return "No active network clients available for the check"
	}

	return prefix + "Unknown network error"
}

func NewCheckResultFailure(checkName string, code ErrorCode, family string, details ...interface{}) CheckResult {
	return CheckResult{
		CheckName:    checkName,
		Success:      false,
		ErrorCode:    code,
		ErrorMessage: RenderError(code, family, details...),
	}
}

func NewCheckResultSuccess(checkName string) CheckResult {
	return CheckResult{
		CheckName: checkName,
		Success:   true,
	}
}
