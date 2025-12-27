/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

func TestBuildQueryFromCmd(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
		errMsg    string
		validate  func(*testing.T, *recipe.Query)
	}{
		{
			name: "valid os",
			args: []string{"cmd", "--os", "ubuntu"},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.Os != recipe.OSUbuntu {
					t.Errorf("Os = %v, want %v", q.Os, recipe.OSUbuntu)
				}
			},
		},
		{
			name:      "invalid os",
			args:      []string{"cmd", "--os", "invalid-os"},
			wantError: true,
			errMsg:    "supported values",
		},
		{
			name: "valid os version",
			args: []string{"cmd", "--osv", "22.04"},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.OsVersion == nil {
					t.Error("OsVersion should not be nil")
					return
				}
				if q.OsVersion.Major != 22 || q.OsVersion.Minor != 4 {
					t.Errorf("OsVersion = %v, want 22.4", q.OsVersion)
				}
			},
		},
		{
			name:      "invalid os version",
			args:      []string{"cmd", "--osv", "invalid"},
			wantError: true,
			errMsg:    "invalid os version",
		},
		{
			name:      "negative os version",
			args:      []string{"cmd", "--osv", "-1.0"},
			wantError: true,
			errMsg:    "cannot contain negative numbers",
		},
		{
			name: "valid kernel version",
			args: []string{"cmd", "--kernel", "5.15.0-1028-aws"},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.Kernel == nil {
					t.Error("Kernel should not be nil")
					return
				}
				if q.Kernel.Major != 5 || q.Kernel.Minor != 15 || q.Kernel.Patch != 0 {
					t.Errorf("Kernel = %v, want 5.15.0", q.Kernel)
				}
			},
		},
		{
			name:      "invalid kernel version",
			args:      []string{"cmd", "--kernel", "invalid"},
			wantError: true,
			errMsg:    "invalid kernel version",
		},
		{
			name:      "negative kernel version",
			args:      []string{"cmd", "--kernel", "5.-1.0"},
			wantError: true,
			errMsg:    "cannot contain negative numbers",
		},
		{
			name: "valid service",
			args: []string{"cmd", "--service", "eks"},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.Service != recipe.ServiceEKS {
					t.Errorf("Service = %v, want %v", q.Service, recipe.ServiceEKS)
				}
			},
		},
		{
			name:      "invalid service",
			args:      []string{"cmd", "--service", "invalid-service"},
			wantError: true,
			errMsg:    "supported values",
		},
		{
			name: "valid k8s version",
			args: []string{"cmd", "--k8s", "v1.28.0-eks-3025e55"},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.K8s == nil {
					t.Error("K8s should not be nil")
					return
				}
				if q.K8s.Major != 1 || q.K8s.Minor != 28 || q.K8s.Patch != 0 {
					t.Errorf("K8s = %v, want 1.28.0", q.K8s)
				}
			},
		},
		{
			name:      "invalid k8s version",
			args:      []string{"cmd", "--k8s", "invalid"},
			wantError: true,
			errMsg:    "invalid kubernetes version",
		},
		{
			name:      "negative k8s version",
			args:      []string{"cmd", "--k8s", "1.-28.0"},
			wantError: true,
			errMsg:    "cannot contain negative numbers",
		},
		{
			name: "valid gpu",
			args: []string{"cmd", "--gpu", "h100"},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.GPU != recipe.GPUH100 {
					t.Errorf("GPU = %v, want %v", q.GPU, recipe.GPUH100)
				}
			},
		},
		{
			name:      "invalid gpu",
			args:      []string{"cmd", "--gpu", "invalid-gpu"},
			wantError: true,
			errMsg:    "supported values",
		},
		{
			name: "valid intent",
			args: []string{"cmd", "--intent", "training"},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.Intent != recipe.IntentTraining {
					t.Errorf("Intent = %v, want %v", q.Intent, recipe.IntentTraining)
				}
			},
		},
		{
			name:      "invalid intent",
			args:      []string{"cmd", "--intent", "invalid-intent"},
			wantError: true,
			errMsg:    "supported values",
		},
		{
			name: "context flag true",
			args: []string{"cmd", "--context"},
			validate: func(t *testing.T, q *recipe.Query) {
				if !q.IncludeContext {
					t.Error("IncludeContext should be true")
				}
			},
		},
		{
			name: "complete query",
			args: []string{
				"cmd",
				"--os", "ubuntu",
				"--osv", "22.04",
				"--kernel", "5.15.0",
				"--service", "eks",
				"--k8s", "v1.28.0",
				"--gpu", "h100",
				"--intent", "training",
				"--context",
			},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.Os != recipe.OSUbuntu {
					t.Errorf("Os = %v, want %v", q.Os, recipe.OSUbuntu)
				}
				if q.Service != recipe.ServiceEKS {
					t.Errorf("Service = %v, want %v", q.Service, recipe.ServiceEKS)
				}
				if q.GPU != recipe.GPUH100 {
					t.Errorf("GPU = %v, want %v", q.GPU, recipe.GPUH100)
				}
				if q.Intent != recipe.IntentTraining {
					t.Errorf("Intent = %v, want %v", q.Intent, recipe.IntentTraining)
				}
				if !q.IncludeContext {
					t.Error("IncludeContext should be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedQuery *recipe.Query
			var capturedErr error

			testCmd := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "os"},
					&cli.StringFlag{Name: "osv"},
					&cli.StringFlag{Name: "kernel"},
					&cli.StringFlag{Name: "service"},
					&cli.StringFlag{Name: "k8s"},
					&cli.StringFlag{Name: "gpu"},
					&cli.StringFlag{Name: "intent"},
					&cli.BoolFlag{Name: "context"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					capturedQuery, capturedErr = buildQueryFromCmd(cmd)
					return capturedErr
				},
			}

			err := testCmd.Run(context.Background(), tt.args)

			if tt.wantError {
				if err == nil && capturedErr == nil {
					t.Error("expected error but got nil")
					return
				}
				errToCheck := err
				if capturedErr != nil {
					errToCheck = capturedErr
				}
				if tt.errMsg != "" && !strings.Contains(errToCheck.Error(), tt.errMsg) {
					t.Errorf("error = %v, want error containing %v", errToCheck, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if capturedErr != nil {
				t.Errorf("unexpected captured error: %v", capturedErr)
				return
			}

			if capturedQuery == nil {
				t.Error("expected non-nil query")
				return
			}

			if tt.validate != nil {
				tt.validate(t, capturedQuery)
			}
		})
	}
}

func TestRecipeCmd_CommandStructure(t *testing.T) {
	cmd := recipeCmd()

	if cmd.Name != "recipe" {
		t.Errorf("Name = %v, want recipe", cmd.Name)
	}

	if cmd.Usage == "" {
		t.Error("Usage should not be empty")
	}

	if cmd.Description == "" {
		t.Error("Description should not be empty")
	}

	requiredFlags := []string{"os", "osv", "kernel", "service", "k8s", "gpu", "intent", "context", "snapshot", "output", "format"}
	for _, flagName := range requiredFlags {
		found := false
		for _, flag := range cmd.Flags {
			if hasName(flag, flagName) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required flag %q not found", flagName)
		}
	}

	if cmd.Action == nil {
		t.Error("Action should not be nil")
	}
}

func TestSnapshotCmd_CommandStructure(t *testing.T) {
	cmd := snapshotCmd()

	if cmd.Name != "snapshot" {
		t.Errorf("Name = %v, want snapshot", cmd.Name)
	}

	if cmd.Usage == "" {
		t.Error("Usage should not be empty")
	}

	if cmd.Description == "" {
		t.Error("Description should not be empty")
	}

	requiredFlags := []string{"output", "format"}
	for _, flagName := range requiredFlags {
		found := false
		for _, flag := range cmd.Flags {
			if hasName(flag, flagName) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required flag %q not found", flagName)
		}
	}

	if cmd.Action == nil {
		t.Error("Action should not be nil")
	}
}

func TestCommandLister(_ *testing.T) {
	commandLister(context.Background(), nil)

	cmd := &cli.Command{Name: "test"}
	commandLister(context.Background(), cmd)

	rootCmd := &cli.Command{
		Name: "root",
		Commands: []*cli.Command{
			{Name: "visible1", Hidden: false},
			{Name: "hidden", Hidden: true},
			{Name: "visible2", Hidden: false},
		},
	}
	commandLister(context.Background(), rootCmd)
}

func hasName(flag cli.Flag, name string) bool {
	if flag == nil {
		return false
	}
	names := flag.Names()
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
