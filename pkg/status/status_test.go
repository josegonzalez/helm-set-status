package status

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    release.Status
		expectError bool
	}{
		{
			name:        "unknown status",
			input:       "unknown",
			expected:    release.StatusUnknown,
			expectError: false,
		},
		{
			name:        "deployed status",
			input:       "deployed",
			expected:    release.StatusDeployed,
			expectError: false,
		},
		{
			name:        "superseded status",
			input:       "superseded",
			expected:    release.StatusSuperseded,
			expectError: false,
		},
		{
			name:        "failed status",
			input:       "failed",
			expected:    release.StatusFailed,
			expectError: false,
		},
		{
			name:        "uninstalling status",
			input:       "uninstalling",
			expected:    release.StatusUninstalling,
			expectError: false,
		},
		{
			name:        "pending-install status",
			input:       "pending-install",
			expected:    release.StatusPendingInstall,
			expectError: false,
		},
		{
			name:        "pending-upgrade status",
			input:       "pending-upgrade",
			expected:    release.StatusPendingUpgrade,
			expectError: false,
		},
		{
			name:        "pending-rollback status",
			input:       "pending-rollback",
			expected:    release.StatusPendingRollback,
			expectError: false,
		},
		{
			name:        "invalid status",
			input:       "invalid",
			expected:    release.StatusUnknown,
			expectError: true,
		},
		{
			name:        "empty status",
			input:       "",
			expected:    release.StatusUnknown,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseStatus(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid status")
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidStatusesString(t *testing.T) {
	result := ValidStatusesString()
	assert.Contains(t, result, "unknown")
	assert.Contains(t, result, "deployed")
	assert.Contains(t, result, "superseded")
	assert.Contains(t, result, "failed")
	assert.Contains(t, result, "uninstalling")
	assert.Contains(t, result, "pending-install")
	assert.Contains(t, result, "pending-upgrade")
	assert.Contains(t, result, "pending-rollback")
	assert.Contains(t, result, ", ")
}

func TestSetStatus(t *testing.T) {
	t.Run("successfully sets status of latest revision", func(t *testing.T) {
		// Create in-memory storage
		mem := driver.NewMemory()
		store := storage.Init(mem)

		// Create test release
		rel := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   1,
			Info: &release.Info{
				Status: release.StatusDeployed,
			},
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		}
		err := store.Create(rel)
		require.NoError(t, err)

		// Create configuration with in-memory storage
		cfg := &action.Configuration{Releases: store}

		// Test status change to failed (revision 0 = latest)
		err = SetStatus(cfg, "test-release", release.StatusFailed, 0, nil)
		require.NoError(t, err)

		// Verify status was updated
		updated, err := store.Last("test-release")
		require.NoError(t, err)
		assert.Equal(t, release.StatusFailed, updated.Info.Status)
		assert.Contains(t, updated.Info.Description, "failed")
	})

	t.Run("sets status of specific revision", func(t *testing.T) {
		mem := driver.NewMemory()
		store := storage.Init(mem)

		// Create multiple revisions
		rel1 := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   1,
			Info: &release.Info{
				Status: release.StatusSuperseded,
			},
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		}
		rel2 := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   2,
			Info: &release.Info{
				Status: release.StatusDeployed,
			},
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		}
		err := store.Create(rel1)
		require.NoError(t, err)
		err = store.Create(rel2)
		require.NoError(t, err)

		cfg := &action.Configuration{Releases: store}

		// Update revision 1 specifically
		err = SetStatus(cfg, "test-release", release.StatusFailed, 1, nil)
		require.NoError(t, err)

		// Verify revision 1 was updated
		updated1, err := store.Get("test-release", 1)
		require.NoError(t, err)
		assert.Equal(t, release.StatusFailed, updated1.Info.Status)

		// Verify revision 2 was NOT updated
		updated2, err := store.Get("test-release", 2)
		require.NoError(t, err)
		assert.Equal(t, release.StatusDeployed, updated2.Info.Status)
	})

	t.Run("sets status to all valid statuses", func(t *testing.T) {
		statuses := []release.Status{
			release.StatusUnknown,
			release.StatusDeployed,
			release.StatusSuperseded,
			release.StatusFailed,
			release.StatusUninstalling,
			release.StatusPendingInstall,
			release.StatusPendingUpgrade,
			release.StatusPendingRollback,
		}

		for _, targetStatus := range statuses {
			t.Run(targetStatus.String(), func(t *testing.T) {
				// Create fresh storage for each test
				mem := driver.NewMemory()
				store := storage.Init(mem)

				rel := &release.Release{
					Name:      "test-release",
					Namespace: "default",
					Version:   1,
					Info: &release.Info{
						Status: release.StatusDeployed,
					},
					Chart: &chart.Chart{
						Metadata: &chart.Metadata{
							Name:    "test-chart",
							Version: "1.0.0",
						},
					},
				}
				err := store.Create(rel)
				require.NoError(t, err)

				cfg := &action.Configuration{Releases: store}

				err = SetStatus(cfg, "test-release", targetStatus, 0, nil)
				require.NoError(t, err)

				updated, err := store.Last("test-release")
				require.NoError(t, err)
				assert.Equal(t, targetStatus, updated.Info.Status)
			})
		}
	})

	t.Run("returns ReleaseNotFoundError for non-existent release", func(t *testing.T) {
		mem := driver.NewMemory()
		store := storage.Init(mem)

		cfg := &action.Configuration{Releases: store}

		err := SetStatus(cfg, "non-existent", release.StatusFailed, 0, nil)
		assert.Error(t, err)

		var notFoundErr *ReleaseNotFoundError
		assert.True(t, errors.As(err, &notFoundErr), "error should be *ReleaseNotFoundError")
		assert.Equal(t, "non-existent", notFoundErr.ReleaseName)
	})

	t.Run("fails for non-existent revision", func(t *testing.T) {
		mem := driver.NewMemory()
		store := storage.Init(mem)

		rel := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   1,
			Info: &release.Info{
				Status: release.StatusDeployed,
			},
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		}
		err := store.Create(rel)
		require.NoError(t, err)

		cfg := &action.Configuration{Releases: store}

		// Try to update non-existent revision 5
		err = SetStatus(cfg, "test-release", release.StatusFailed, 5, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get release")
		assert.Contains(t, err.Error(), "revision 5")
	})
}

func TestValidStatuses(t *testing.T) {
	// Verify ValidStatuses contains all expected values
	expected := []string{
		"unknown",
		"deployed",
		"superseded",
		"failed",
		"uninstalling",
		"pending-install",
		"pending-upgrade",
		"pending-rollback",
	}

	assert.Equal(t, expected, ValidStatuses)
	assert.Len(t, ValidStatuses, 8)
}

// failingUpdateDriver wraps a memory driver but fails on Update
type failingUpdateDriver struct {
	*driver.Memory
}

func (f *failingUpdateDriver) Update(key string, rls *release.Release) error {
	return driver.ErrReleaseNotFound
}

func TestSetStatus_UpdateError(t *testing.T) {
	// Create a memory driver and wrap it
	mem := driver.NewMemory()

	// Create a release directly in the memory driver
	rel := &release.Release{
		Name:      "test-release",
		Namespace: "default",
		Version:   1,
		Info: &release.Info{
			Status: release.StatusDeployed,
		},
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name:    "test-chart",
				Version: "1.0.0",
			},
		},
	}
	err := mem.Create("test-release.v1", rel)
	require.NoError(t, err)

	// Create a failing driver and initialize storage with it
	failingDriver := &failingUpdateDriver{Memory: mem}
	store := storage.Init(failingDriver)

	cfg := &action.Configuration{Releases: store}

	err = SetStatus(cfg, "test-release", release.StatusFailed, 0, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update release")
}

func TestSetStatus_FromPrecondition(t *testing.T) {
	t.Run("succeeds when current status is in allowed list", func(t *testing.T) {
		mem := driver.NewMemory()
		store := storage.Init(mem)

		rel := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   1,
			Info: &release.Info{
				Status: release.StatusPendingUpgrade,
			},
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		}
		err := store.Create(rel)
		require.NoError(t, err)

		cfg := &action.Configuration{Releases: store}

		// Should succeed because current status (pending-upgrade) is in allowed list
		allowedFrom := []release.Status{release.StatusPendingUpgrade, release.StatusPendingRollback}
		err = SetStatus(cfg, "test-release", release.StatusDeployed, 0, allowedFrom)
		require.NoError(t, err)

		updated, err := store.Last("test-release")
		require.NoError(t, err)
		assert.Equal(t, release.StatusDeployed, updated.Info.Status)
	})

	t.Run("fails when current status is not in allowed list", func(t *testing.T) {
		mem := driver.NewMemory()
		store := storage.Init(mem)

		rel := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   1,
			Info: &release.Info{
				Status: release.StatusDeployed,
			},
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		}
		err := store.Create(rel)
		require.NoError(t, err)

		cfg := &action.Configuration{Releases: store}

		// Should fail because current status (deployed) is NOT in allowed list
		allowedFrom := []release.Status{release.StatusPendingUpgrade, release.StatusPendingRollback}
		err = SetStatus(cfg, "test-release", release.StatusFailed, 0, allowedFrom)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "current status")
		assert.Contains(t, err.Error(), "deployed")
		assert.Contains(t, err.Error(), "not in allowed list")

		// Verify status was NOT updated
		unchanged, err := store.Last("test-release")
		require.NoError(t, err)
		assert.Equal(t, release.StatusDeployed, unchanged.Info.Status)
	})

	t.Run("succeeds when no from statuses specified (empty list)", func(t *testing.T) {
		mem := driver.NewMemory()
		store := storage.Init(mem)

		rel := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   1,
			Info: &release.Info{
				Status: release.StatusDeployed,
			},
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		}
		err := store.Create(rel)
		require.NoError(t, err)

		cfg := &action.Configuration{Releases: store}

		// Should succeed with empty allowed list (any status can transition)
		err = SetStatus(cfg, "test-release", release.StatusFailed, 0, []release.Status{})
		require.NoError(t, err)

		updated, err := store.Last("test-release")
		require.NoError(t, err)
		assert.Equal(t, release.StatusFailed, updated.Info.Status)
	})

	t.Run("succeeds with single allowed status", func(t *testing.T) {
		mem := driver.NewMemory()
		store := storage.Init(mem)

		rel := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   1,
			Info: &release.Info{
				Status: release.StatusPendingInstall,
			},
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		}
		err := store.Create(rel)
		require.NoError(t, err)

		cfg := &action.Configuration{Releases: store}

		// Should succeed with single matching status
		allowedFrom := []release.Status{release.StatusPendingInstall}
		err = SetStatus(cfg, "test-release", release.StatusFailed, 0, allowedFrom)
		require.NoError(t, err)

		updated, err := store.Last("test-release")
		require.NoError(t, err)
		assert.Equal(t, release.StatusFailed, updated.Info.Status)
	})

	t.Run("returns PreconditionError type when precondition fails", func(t *testing.T) {
		mem := driver.NewMemory()
		store := storage.Init(mem)

		rel := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   1,
			Info: &release.Info{
				Status: release.StatusDeployed,
			},
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		}
		err := store.Create(rel)
		require.NoError(t, err)

		cfg := &action.Configuration{Releases: store}

		// Should return PreconditionError because current status (deployed) is NOT in allowed list
		allowedFrom := []release.Status{release.StatusPendingUpgrade, release.StatusPendingRollback}
		err = SetStatus(cfg, "test-release", release.StatusFailed, 0, allowedFrom)
		assert.Error(t, err)

		// Verify error type using errors.As
		var precondErr *PreconditionError
		assert.True(t, errors.As(err, &precondErr), "error should be *PreconditionError")
		assert.Equal(t, release.StatusDeployed, precondErr.CurrentStatus)
		assert.Equal(t, allowedFrom, precondErr.AllowedStatuses)
	})
}
