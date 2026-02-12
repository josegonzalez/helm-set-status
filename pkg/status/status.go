package status

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	helmtime "helm.sh/helm/v3/pkg/time"
)

// ReleaseNotFoundError is returned when a release is not found in storage.
type ReleaseNotFoundError struct {
	ReleaseName string
}

func (e *ReleaseNotFoundError) Error() string {
	return fmt.Sprintf("release %q not found", e.ReleaseName)
}

// PreconditionError is returned when a precondition check fails.
type PreconditionError struct {
	CurrentStatus   release.Status
	AllowedStatuses []release.Status
}

func (e *PreconditionError) Error() string {
	return fmt.Sprintf("current status %q is not in allowed list: %v",
		e.CurrentStatus, statusListToStrings(e.AllowedStatuses))
}

// statusListToStrings converts a slice of release.Status to a slice of strings.
func statusListToStrings(statuses []release.Status) []string {
	result := make([]string, len(statuses))
	for i, s := range statuses {
		result[i] = s.String()
	}
	return result
}

// SetStatus sets the status of a Helm release.
// If revision is 0, it updates the latest release.
// If revision is > 0, it updates that specific revision.
// If allowedFromStatuses is non-empty, the status change only proceeds if
// the current release status is in the allowed list.
func SetStatus(cfg *action.Configuration, releaseName string, status release.Status, revision int, allowedFromStatuses []release.Status) error {
	var rel *release.Release
	var err error

	if revision > 0 {
		// Get specific revision
		rel, err = cfg.Releases.Get(releaseName, revision)
		if err != nil {
			return fmt.Errorf("failed to get release %s revision %d: %w", releaseName, revision, err)
		}
	} else {
		// Get latest release from storage
		rel, err = cfg.Releases.Last(releaseName)
		if err != nil {
			return &ReleaseNotFoundError{ReleaseName: releaseName}
		}
	}

	// Check precondition if allowedFromStatuses is specified
	if len(allowedFromStatuses) > 0 {
		currentStatus := rel.Info.Status
		allowed := false
		for _, s := range allowedFromStatuses {
			if currentStatus == s {
				allowed = true
				break
			}
		}
		if !allowed {
			return &PreconditionError{
				CurrentStatus:   currentStatus,
				AllowedStatuses: allowedFromStatuses,
			}
		}
	}

	// Update status
	rel.Info.Status = status
	rel.Info.Description = fmt.Sprintf("status set to %s", status.String())
	rel.Info.LastDeployed = helmtime.Now()

	// Persist back to storage
	if err := cfg.Releases.Update(rel); err != nil {
		return fmt.Errorf("failed to update release %s: %w", releaseName, err)
	}

	return nil
}
