package status

import (
	"fmt"

	"helm.sh/helm/v3/pkg/release"
)

// ValidStatuses lists all valid Helm release status values.
var ValidStatuses = []string{
	"unknown",
	"deployed",
	"superseded",
	"failed",
	"uninstalling",
	"pending-install",
	"pending-upgrade",
	"pending-rollback",
}

// ParseStatus converts a string to a release.Status.
// Returns an error if the status string is not valid.
func ParseStatus(s string) (release.Status, error) {
	switch s {
	case "unknown":
		return release.StatusUnknown, nil
	case "deployed":
		return release.StatusDeployed, nil
	case "superseded":
		return release.StatusSuperseded, nil
	case "failed":
		return release.StatusFailed, nil
	case "uninstalling":
		return release.StatusUninstalling, nil
	case "pending-install":
		return release.StatusPendingInstall, nil
	case "pending-upgrade":
		return release.StatusPendingUpgrade, nil
	case "pending-rollback":
		return release.StatusPendingRollback, nil
	default:
		return release.StatusUnknown, fmt.Errorf("invalid status: %s", s)
	}
}

// ValidStatusesString returns a comma-separated string of valid status values.
func ValidStatusesString() string {
	result := ""
	for i, s := range ValidStatuses {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
