package service

import "strings"

func NormalizeAdminAccountStatusInput(status string) string {
	normalized := strings.TrimSpace(strings.ToLower(status))
	switch normalized {
	case "":
		return ""
	case StatusActive:
		return StatusActive
	case "inactive", StatusDisabled:
		return StatusDisabled
	case StatusError:
		return StatusError
	default:
		return strings.TrimSpace(status)
	}
}

func PresentAdminAccountStatus(status string) string {
	normalized := strings.TrimSpace(strings.ToLower(status))
	switch normalized {
	case StatusDisabled, "inactive":
		return "inactive"
	case StatusActive:
		return StatusActive
	case StatusError:
		return StatusError
	default:
		return strings.TrimSpace(status)
	}
}
