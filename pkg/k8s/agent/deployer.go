package agent

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
)

// Deploy deploys the agent with all required resources (RBAC + Job).
// This is the main entry point that orchestrates the deployment.
func (d *Deployer) Deploy(ctx context.Context) error {
	// Step 1: Ensure RBAC resources (idempotent - reuses if already exists)
	if err := d.ensureServiceAccount(ctx); err != nil {
		return fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	if err := d.ensureRole(ctx); err != nil {
		return fmt.Errorf("failed to create Role: %w", err)
	}

	if err := d.ensureRoleBinding(ctx); err != nil {
		return fmt.Errorf("failed to create RoleBinding: %w", err)
	}

	if err := d.ensureClusterRole(ctx); err != nil {
		return fmt.Errorf("failed to create ClusterRole: %w", err)
	}

	if err := d.ensureClusterRoleBinding(ctx); err != nil {
		return fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
	}

	// Step 2: Ensure Job (delete existing + recreate)
	if err := d.ensureJob(ctx); err != nil {
		return fmt.Errorf("failed to create Job: %w", err)
	}

	return nil
}

// WaitForCompletion waits for the agent Job to complete successfully.
// Returns error if the Job fails or times out.
func (d *Deployer) WaitForCompletion(ctx context.Context, timeout time.Duration) error {
	return d.waitForJobCompletion(ctx, timeout)
}

// GetSnapshot retrieves the snapshot data from the ConfigMap created by the agent.
// Returns the snapshot YAML content.
func (d *Deployer) GetSnapshot(ctx context.Context) ([]byte, error) {
	return d.getSnapshotFromConfigMap(ctx)
}

// Cleanup removes the agent Job and optionally the RBAC resources.
// By default, RBAC resources are kept for reuse in future deployments.
func (d *Deployer) Cleanup(ctx context.Context, opts CleanupOptions) error {
	// Always delete the Job
	if err := d.deleteJob(ctx); err != nil {
		return fmt.Errorf("failed to delete Job: %w", err)
	}

	// Optionally delete RBAC resources
	if opts.RemoveRBAC {
		if err := d.deleteServiceAccount(ctx); err != nil {
			return fmt.Errorf("failed to delete ServiceAccount: %w", err)
		}

		if err := d.deleteRole(ctx); err != nil {
			return fmt.Errorf("failed to delete Role: %w", err)
		}

		if err := d.deleteRoleBinding(ctx); err != nil {
			return fmt.Errorf("failed to delete RoleBinding: %w", err)
		}

		if err := d.deleteClusterRole(ctx); err != nil {
			return fmt.Errorf("failed to delete ClusterRole: %w", err)
		}

		if err := d.deleteClusterRoleBinding(ctx); err != nil {
			return fmt.Errorf("failed to delete ClusterRoleBinding: %w", err)
		}
	}

	return nil
}

// ignoreAlreadyExists returns nil if the error is "already exists", otherwise returns the error.
// Used to make resource creation idempotent.
func ignoreAlreadyExists(err error) error {
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// ignoreNotFound returns nil if the error is "not found", otherwise returns the error.
// Used to make resource deletion idempotent.
func ignoreNotFound(err error) error {
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}
