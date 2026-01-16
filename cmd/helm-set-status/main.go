package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/josegonzalez/helm-set-status/pkg/status"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
)

var version = "dev"

// ConfigurationFactory creates Helm action configurations.
// This can be overridden for testing.
var ConfigurationFactory = status.NewConfiguration

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

var revision int
var fromStatuses []string
var noFail bool

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm-set-status RELEASE STATUS",
		Short: "Set the status of a Helm release",
		Long: `Set the status of a Helm release to any valid Helm status value.

Valid status values:
  unknown, deployed, superseded, failed,
  uninstalling, pending-install, pending-upgrade, pending-rollback

By default, the latest revision is updated. Use --revision to update a specific revision.
Use --from to only change status if the current status matches one of the specified values.`,
		Args:    cobra.ExactArgs(2),
		Version: version,
		RunE:    run,
	}

	cmd.Flags().IntVar(&revision, "revision", 0, "update a specific revision (default: latest)")
	cmd.Flags().StringSliceVar(&fromStatuses, "from", nil, "only change status if current status is one of these values (can specify multiple)")
	cmd.Flags().BoolVar(&noFail, "no-fail", false, "exit 0 instead of 1 when --from precondition is not met")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	rev, _ := cmd.Flags().GetInt("revision")
	from, _ := cmd.Flags().GetStringSlice("from")
	nf, _ := cmd.Flags().GetBool("no-fail")
	return runWithConfigFactory(cmd, args, rev, from, nf, ConfigurationFactory)
}

func runWithConfigFactory(cmd *cobra.Command, args []string, rev int, fromStatuses []string, noFail bool, configFactory func() (*action.Configuration, error)) error {
	releaseName := args[0]
	statusStr := args[1]

	// Parse and validate status
	targetStatus, err := status.ParseStatus(statusStr)
	if err != nil {
		return fmt.Errorf("%w\nValid statuses: %s", err, status.ValidStatusesString())
	}

	// Parse and validate --from statuses
	var allowedFromStatuses []release.Status
	for _, s := range fromStatuses {
		parsed, err := status.ParseStatus(s)
		if err != nil {
			return fmt.Errorf("invalid --from status %q: %w\nValid statuses: %s", s, err, status.ValidStatusesString())
		}
		allowedFromStatuses = append(allowedFromStatuses, parsed)
	}

	// Create Helm configuration
	cfg, err := configFactory()
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}

	// Set the status
	if err := status.SetStatus(cfg, releaseName, targetStatus, rev, allowedFromStatuses); err != nil {
		var precondErr *status.PreconditionError
		if errors.As(err, &precondErr) && noFail {
			fmt.Fprintf(cmd.OutOrStdout(), "Skipped: %s\n", err)
			return nil
		}
		return err
	}

	if rev > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Release %q revision %d status set to %q\n", releaseName, rev, statusStr)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Release %q status set to %q\n", releaseName, statusStr)
	}
	return nil
}
