package main

import (
	"bytes"
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

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	assert.Equal(t, "helm-set-status RELEASE STATUS", cmd.Use)
	assert.Equal(t, "Set the status of a Helm release", cmd.Short)
	assert.Contains(t, cmd.Long, "Valid status values")
	assert.Contains(t, cmd.Long, "--revision")
	assert.Contains(t, cmd.Long, "--from")
	assert.Equal(t, version, cmd.Version)

	// Verify --revision flag exists
	revFlag := cmd.Flags().Lookup("revision")
	assert.NotNil(t, revFlag)
	assert.Equal(t, "0", revFlag.DefValue)

	// Verify --from flag exists
	fromFlag := cmd.Flags().Lookup("from")
	assert.NotNil(t, fromFlag)
	assert.Equal(t, "[]", fromFlag.DefValue)

	// Verify --no-fail flag exists
	noFailFlag := cmd.Flags().Lookup("no-fail")
	assert.NotNil(t, noFailFlag)
	assert.Equal(t, "false", noFailFlag.DefValue)
}

func TestRunWithConfigFactory_Success(t *testing.T) {
	// Create in-memory storage with a test release
	mem := driver.NewMemory()
	store := storage.Init(mem)

	rel := &release.Release{
		Name:      "my-release",
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

	// Create configuration factory
	configFactory := func() (*action.Configuration, error) {
		return &action.Configuration{Releases: store}, nil
	}

	// Create command
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Run with revision 0 (latest)
	err = runWithConfigFactory(cmd, []string{"my-release", "failed"}, 0, nil, false, configFactory)
	require.NoError(t, err)

	// Verify output
	assert.Contains(t, buf.String(), "my-release")
	assert.Contains(t, buf.String(), "failed")

	// Verify status was changed
	updated, err := store.Last("my-release")
	require.NoError(t, err)
	assert.Equal(t, release.StatusFailed, updated.Info.Status)
}

func TestRunWithConfigFactory_SpecificRevision(t *testing.T) {
	mem := driver.NewMemory()
	store := storage.Init(mem)

	// Create multiple revisions
	rel1 := &release.Release{
		Name:      "my-release",
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
		Name:      "my-release",
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

	configFactory := func() (*action.Configuration, error) {
		return &action.Configuration{Releases: store}, nil
	}

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Update revision 1 specifically
	err = runWithConfigFactory(cmd, []string{"my-release", "failed"}, 1, nil, false, configFactory)
	require.NoError(t, err)

	// Verify output mentions revision
	assert.Contains(t, buf.String(), "my-release")
	assert.Contains(t, buf.String(), "revision 1")
	assert.Contains(t, buf.String(), "failed")

	// Verify only revision 1 was changed
	updated1, err := store.Get("my-release", 1)
	require.NoError(t, err)
	assert.Equal(t, release.StatusFailed, updated1.Info.Status)

	updated2, err := store.Get("my-release", 2)
	require.NoError(t, err)
	assert.Equal(t, release.StatusDeployed, updated2.Info.Status)
}

func TestRunWithConfigFactory_InvalidStatus(t *testing.T) {
	// Configuration factory won't be called for invalid status
	configFactory := func() (*action.Configuration, error) {
		return nil, errors.New("should not be called")
	}

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := runWithConfigFactory(cmd, []string{"my-release", "invalid-status"}, 0, nil, false, configFactory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
	assert.Contains(t, err.Error(), "Valid statuses")
}

func TestRunWithConfigFactory_ConfigError(t *testing.T) {
	configFactory := func() (*action.Configuration, error) {
		return nil, errors.New("config creation failed")
	}

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := runWithConfigFactory(cmd, []string{"my-release", "failed"}, 0, nil, false, configFactory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create configuration")
}

func TestRunWithConfigFactory_ReleaseNotFound(t *testing.T) {
	// Create empty storage
	mem := driver.NewMemory()
	store := storage.Init(mem)

	configFactory := func() (*action.Configuration, error) {
		return &action.Configuration{Releases: store}, nil
	}

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := runWithConfigFactory(cmd, []string{"non-existent", "failed"}, 0, nil, false, configFactory)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Warning")
	assert.Contains(t, buf.String(), "non-existent")
}

func TestRun_UsesConfigurationFactory(t *testing.T) {
	// Save original factory
	originalFactory := ConfigurationFactory
	defer func() { ConfigurationFactory = originalFactory }()

	// Set up test factory
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

	ConfigurationFactory = func() (*action.Configuration, error) {
		return &action.Configuration{Releases: store}, nil
	}

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = run(cmd, []string{"test-release", "deployed"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "test-release")
}

func TestRunWithConfigFactory_AllStatuses(t *testing.T) {
	statuses := []string{
		"unknown",
		"deployed",
		"superseded",
		"failed",
		"uninstalling",
		"pending-install",
		"pending-upgrade",
		"pending-rollback",
	}

	for _, statusStr := range statuses {
		t.Run(statusStr, func(t *testing.T) {
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

			configFactory := func() (*action.Configuration, error) {
				return &action.Configuration{Releases: store}, nil
			}

			cmd := newRootCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			err = runWithConfigFactory(cmd, []string{"test-release", statusStr}, 0, nil, false, configFactory)
			require.NoError(t, err)
			assert.Contains(t, buf.String(), statusStr)
		})
	}
}

func TestRunWithConfigFactory_FromFlag(t *testing.T) {
	t.Run("succeeds with matching from status", func(t *testing.T) {
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

		configFactory := func() (*action.Configuration, error) {
			return &action.Configuration{Releases: store}, nil
		}

		cmd := newRootCmd()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Should succeed because current status (pending-upgrade) is in allowed list
		err = runWithConfigFactory(cmd, []string{"test-release", "deployed"}, 0, []string{"pending-upgrade", "pending-rollback"}, false, configFactory)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "test-release")
		assert.Contains(t, buf.String(), "deployed")
	})

	t.Run("fails with non-matching from status", func(t *testing.T) {
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

		configFactory := func() (*action.Configuration, error) {
			return &action.Configuration{Releases: store}, nil
		}

		cmd := newRootCmd()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Should fail because current status (deployed) is NOT in allowed list
		err = runWithConfigFactory(cmd, []string{"test-release", "failed"}, 0, []string{"pending-upgrade", "pending-rollback"}, false, configFactory)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "current status")
		assert.Contains(t, err.Error(), "not in allowed list")
	})

	t.Run("fails with invalid from status", func(t *testing.T) {
		configFactory := func() (*action.Configuration, error) {
			return nil, errors.New("should not be called")
		}

		cmd := newRootCmd()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := runWithConfigFactory(cmd, []string{"test-release", "deployed"}, 0, []string{"invalid-status"}, false, configFactory)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --from status")
		assert.Contains(t, err.Error(), "Valid statuses")
	})

	t.Run("works with multiple from statuses", func(t *testing.T) {
		mem := driver.NewMemory()
		store := storage.Init(mem)

		rel := &release.Release{
			Name:      "test-release",
			Namespace: "default",
			Version:   1,
			Info: &release.Info{
				Status: release.StatusPendingRollback,
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

		configFactory := func() (*action.Configuration, error) {
			return &action.Configuration{Releases: store}, nil
		}

		cmd := newRootCmd()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Should succeed because pending-rollback is in the list of allowed statuses
		err = runWithConfigFactory(cmd, []string{"test-release", "deployed"}, 0, []string{"pending-install", "pending-upgrade", "pending-rollback"}, false, configFactory)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "test-release")
	})
}

func TestRunWithConfigFactory_NoFailFlag(t *testing.T) {
	t.Run("with matching precondition succeeds normally", func(t *testing.T) {
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

		configFactory := func() (*action.Configuration, error) {
			return &action.Configuration{Releases: store}, nil
		}

		cmd := newRootCmd()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Should succeed because current status matches precondition
		err = runWithConfigFactory(cmd, []string{"test-release", "deployed"}, 0, []string{"pending-upgrade"}, true, configFactory)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "test-release")
		assert.Contains(t, buf.String(), "deployed")

		// Verify status was changed
		updated, err := store.Last("test-release")
		require.NoError(t, err)
		assert.Equal(t, release.StatusDeployed, updated.Info.Status)
	})

	t.Run("with non-matching precondition exits 0 and prints skip message", func(t *testing.T) {
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

		configFactory := func() (*action.Configuration, error) {
			return &action.Configuration{Releases: store}, nil
		}

		cmd := newRootCmd()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Should not fail because --no-fail is set, even though precondition doesn't match
		err = runWithConfigFactory(cmd, []string{"test-release", "failed"}, 0, []string{"pending-upgrade"}, true, configFactory)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Skipped")
		assert.Contains(t, buf.String(), "current status")
		assert.Contains(t, buf.String(), "not in allowed list")

		// Verify status was NOT changed
		unchanged, err := store.Last("test-release")
		require.NoError(t, err)
		assert.Equal(t, release.StatusDeployed, unchanged.Info.Status)
	})

	t.Run("without no-fail flag and non-matching precondition exits 1", func(t *testing.T) {
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

		configFactory := func() (*action.Configuration, error) {
			return &action.Configuration{Releases: store}, nil
		}

		cmd := newRootCmd()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Should fail because --no-fail is not set
		err = runWithConfigFactory(cmd, []string{"test-release", "failed"}, 0, []string{"pending-upgrade"}, false, configFactory)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "current status")
		assert.Contains(t, err.Error(), "not in allowed list")

		// Verify status was NOT changed
		unchanged, err := store.Last("test-release")
		require.NoError(t, err)
		assert.Equal(t, release.StatusDeployed, unchanged.Info.Status)
	})
}
