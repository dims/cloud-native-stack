package agent

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testName = "eidos"

func TestDeployer_EnsureRBAC(t *testing.T) {
	clientset := fake.NewClientset()
	config := Config{
		Namespace:          "test-namespace",
		ServiceAccountName: testName,
		JobName:            testName,
		Image:              "ghcr.io/nvidia/eidos:latest",
		Output:             "cm://test-namespace/eidos-snapshot",
	}
	deployer := NewDeployer(clientset, config)
	ctx := context.Background()

	// Test ServiceAccount creation
	t.Run("create ServiceAccount", func(t *testing.T) {
		if err := deployer.ensureServiceAccount(ctx); err != nil {
			t.Fatalf("failed to create ServiceAccount: %v", err)
		}

		sa, err := clientset.CoreV1().ServiceAccounts(config.Namespace).
			Get(ctx, testName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("ServiceAccount not found: %v", err)
		}
		if sa.Name != testName {
			t.Errorf("expected SA name %q, got %q", testName, sa.Name)
		}
	})

	// Test Role creation
	t.Run("create Role", func(t *testing.T) {
		if err := deployer.ensureRole(ctx); err != nil {
			t.Fatalf("failed to create Role: %v", err)
		}

		role, err := clientset.RbacV1().Roles(config.Namespace).
			Get(ctx, testName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Role not found: %v", err)
		}

		// Verify policy rules
		if len(role.Rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(role.Rules))
		}

		// Check ConfigMap rule
		rule0 := role.Rules[0]
		if len(rule0.Resources) != 1 || rule0.Resources[0] != "configmaps" {
			t.Errorf("expected configmaps in first rule, got %v", rule0.Resources)
		}
		if !containsVerb(rule0.Verbs, "create") || !containsVerb(rule0.Verbs, "update") {
			t.Errorf("expected create/update verbs, got %v", rule0.Verbs)
		}
	})

	// Test RoleBinding creation
	t.Run("create RoleBinding", func(t *testing.T) {
		if err := deployer.ensureRoleBinding(ctx); err != nil {
			t.Fatalf("failed to create RoleBinding: %v", err)
		}

		rb, err := clientset.RbacV1().RoleBindings(config.Namespace).
			Get(ctx, testName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("RoleBinding not found: %v", err)
		}

		// Verify subjects
		if len(rb.Subjects) != 1 {
			t.Errorf("expected 1 subject, got %d", len(rb.Subjects))
		}
		if rb.Subjects[0].Name != testName {
			t.Errorf("expected subject name 'eidos', got %q", rb.Subjects[0].Name)
		}

		// Verify roleRef
		if rb.RoleRef.Name != testName {
			t.Errorf("expected roleRef name 'eidos', got %q", rb.RoleRef.Name)
		}
	})

	// Test ClusterRole creation
	t.Run("create ClusterRole", func(t *testing.T) {
		if err := deployer.ensureClusterRole(ctx); err != nil {
			t.Fatalf("failed to create ClusterRole: %v", err)
		}

		cr, err := clientset.RbacV1().ClusterRoles().
			Get(ctx, "eidos-node-reader", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("ClusterRole not found: %v", err)
		}

		// Verify policy rules
		if len(cr.Rules) != 4 {
			t.Errorf("expected 4 rules, got %d", len(cr.Rules))
		}
	})

	// Test ClusterRoleBinding creation
	t.Run("create ClusterRoleBinding", func(t *testing.T) {
		if err := deployer.ensureClusterRoleBinding(ctx); err != nil {
			t.Fatalf("failed to create ClusterRoleBinding: %v", err)
		}

		crb, err := clientset.RbacV1().ClusterRoleBindings().
			Get(ctx, "eidos-node-reader", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("ClusterRoleBinding not found: %v", err)
		}

		// Verify subjects
		if len(crb.Subjects) != 1 {
			t.Errorf("expected 1 subject, got %d", len(crb.Subjects))
		}

		// Verify roleRef
		if crb.RoleRef.Name != "eidos-node-reader" {
			t.Errorf("expected roleRef name 'eidos-node-reader', got %q", crb.RoleRef.Name)
		}
	})
}

func TestDeployer_EnsureRBAC_Idempotent(t *testing.T) {
	clientset := fake.NewClientset()
	config := Config{
		Namespace:          "test-namespace",
		ServiceAccountName: testName,
		JobName:            testName,
		Image:              "ghcr.io/nvidia/eidos:latest",
		Output:             "cm://test-namespace/eidos-snapshot",
	}
	deployer := NewDeployer(clientset, config)
	ctx := context.Background()

	// Create resources twice - second call should be idempotent
	if err := deployer.ensureServiceAccount(ctx); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	if err := deployer.ensureServiceAccount(ctx); err != nil {
		t.Fatalf("second create failed (not idempotent): %v", err)
	}

	// Verify only one ServiceAccount exists
	saList, err := clientset.CoreV1().ServiceAccounts(config.Namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list ServiceAccounts: %v", err)
	}
	if len(saList.Items) != 1 {
		t.Errorf("expected 1 ServiceAccount, got %d", len(saList.Items))
	}
}

func TestDeployer_EnsureJob(t *testing.T) {
	clientset := fake.NewClientset()
	config := Config{
		Namespace:          "test-namespace",
		ServiceAccountName: testName,
		JobName:            testName,
		Image:              "ghcr.io/nvidia/eidos:latest",
		Output:             "cm://test-namespace/eidos-snapshot",
		NodeSelector: map[string]string{
			"nodeGroup": "customer-gpu",
		},
		Tolerations: []corev1.Toleration{
			{
				Key:      "dedicated",
				Operator: corev1.TolerationOpEqual,
				Value:    "user-workload",
				Effect:   corev1.TaintEffectNoSchedule,
			},
		},
	}
	deployer := NewDeployer(clientset, config)
	ctx := context.Background()

	t.Run("create Job", func(t *testing.T) {
		if err := deployer.ensureJob(ctx); err != nil {
			t.Fatalf("failed to create Job: %v", err)
		}

		job, err := clientset.BatchV1().Jobs(config.Namespace).
			Get(ctx, config.JobName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Job not found: %v", err)
		}

		// Verify Job spec
		if job.Spec.Template.Spec.ServiceAccountName != config.ServiceAccountName {
			t.Errorf("expected ServiceAccountName %q, got %q",
				config.ServiceAccountName, job.Spec.Template.Spec.ServiceAccountName)
		}

		// Verify host settings
		if !job.Spec.Template.Spec.HostPID {
			t.Error("expected HostPID to be true")
		}
		if !job.Spec.Template.Spec.HostNetwork {
			t.Error("expected HostNetwork to be true")
		}
		if !job.Spec.Template.Spec.HostIPC {
			t.Error("expected HostIPC to be true")
		}

		// Verify node selector
		if job.Spec.Template.Spec.NodeSelector["nodeGroup"] != "customer-gpu" {
			t.Errorf("expected nodeGroup=customer-gpu, got %v", job.Spec.Template.Spec.NodeSelector)
		}

		// Verify tolerations
		if len(job.Spec.Template.Spec.Tolerations) != 1 {
			t.Errorf("expected 1 toleration, got %d", len(job.Spec.Template.Spec.Tolerations))
		}

		// Verify container
		if len(job.Spec.Template.Spec.Containers) != 1 {
			t.Fatalf("expected 1 container, got %d", len(job.Spec.Template.Spec.Containers))
		}
		container := job.Spec.Template.Spec.Containers[0]
		if container.Image != config.Image {
			t.Errorf("expected image %q, got %q", config.Image, container.Image)
		}

		// Verify volumes
		if len(job.Spec.Template.Spec.Volumes) != 2 {
			t.Errorf("expected 2 volumes, got %d", len(job.Spec.Template.Spec.Volumes))
		}
	})

	t.Run("recreate Job deletes old one", func(t *testing.T) {
		// Create Job first time
		if err := deployer.ensureJob(ctx); err != nil {
			t.Fatalf("first create failed: %v", err)
		}

		// Create Job second time - should delete and recreate
		if err := deployer.ensureJob(ctx); err != nil {
			t.Fatalf("second create failed: %v", err)
		}

		// Verify Job still exists (fake client doesn't support watch/wait,
		// but we can verify the Job exists)
		_, err := clientset.BatchV1().Jobs(config.Namespace).
			Get(ctx, config.JobName, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Job should exist after recreate: %v", err)
		}
	})
}

func TestDeployer_Deploy(t *testing.T) {
	clientset := fake.NewClientset()
	config := Config{
		Namespace:          "test-namespace",
		ServiceAccountName: testName,
		JobName:            testName,
		Image:              "ghcr.io/nvidia/eidos:latest",
		Output:             "cm://test-namespace/eidos-snapshot",
	}
	deployer := NewDeployer(clientset, config)
	ctx := context.Background()

	// Deploy should create all resources
	if err := deployer.Deploy(ctx); err != nil {
		t.Fatalf("Deploy() failed: %v", err)
	}

	// Verify ServiceAccount
	_, err := clientset.CoreV1().ServiceAccounts(config.Namespace).
		Get(ctx, testName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("ServiceAccount not created: %v", err)
	}

	// Verify Role
	_, err = clientset.RbacV1().Roles(config.Namespace).
		Get(ctx, testName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Role not created: %v", err)
	}

	// Verify RoleBinding
	_, err = clientset.RbacV1().RoleBindings(config.Namespace).
		Get(ctx, testName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("RoleBinding not created: %v", err)
	}

	// Verify ClusterRole
	_, err = clientset.RbacV1().ClusterRoles().
		Get(ctx, "eidos-node-reader", metav1.GetOptions{})
	if err != nil {
		t.Errorf("ClusterRole not created: %v", err)
	}

	// Verify ClusterRoleBinding
	_, err = clientset.RbacV1().ClusterRoleBindings().
		Get(ctx, "eidos-node-reader", metav1.GetOptions{})
	if err != nil {
		t.Errorf("ClusterRoleBinding not created: %v", err)
	}

	// Verify Job
	_, err = clientset.BatchV1().Jobs(config.Namespace).
		Get(ctx, config.JobName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Job not created: %v", err)
	}
}

func TestDeployer_Cleanup(t *testing.T) {
	clientset := fake.NewClientset()
	config := Config{
		Namespace:          "test-namespace",
		ServiceAccountName: testName,
		JobName:            testName,
		Image:              "ghcr.io/nvidia/eidos:latest",
		Output:             "cm://test-namespace/eidos-snapshot",
	}
	deployer := NewDeployer(clientset, config)
	ctx := context.Background()

	// Deploy first
	if err := deployer.Deploy(ctx); err != nil {
		t.Fatalf("Deploy() failed: %v", err)
	}

	// Cleanup without removing RBAC
	if err := deployer.Cleanup(ctx, CleanupOptions{RemoveRBAC: false}); err != nil {
		t.Fatalf("Cleanup() failed: %v", err)
	}

	// Job should be deleted
	_, err := clientset.BatchV1().Jobs(config.Namespace).
		Get(ctx, config.JobName, metav1.GetOptions{})
	if err == nil {
		t.Errorf("Job should be deleted")
	}

	// ServiceAccount should still exist
	_, err = clientset.CoreV1().ServiceAccounts(config.Namespace).
		Get(ctx, testName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("ServiceAccount should still exist: %v", err)
	}

	// Cleanup with RBAC removal
	if cleanupErr := deployer.Cleanup(ctx, CleanupOptions{RemoveRBAC: true}); cleanupErr != nil {
		t.Fatalf("Cleanup() with RemoveRBAC failed: %v", cleanupErr)
	}

	// ServiceAccount should be deleted
	_, err = clientset.CoreV1().ServiceAccounts(config.Namespace).
		Get(ctx, testName, metav1.GetOptions{})
	if err == nil {
		t.Errorf("ServiceAccount should be deleted")
	}
}

// Helper function
func containsVerb(verbs []string, verb string) bool {
	for _, v := range verbs {
		if v == verb {
			return true
		}
	}
	return false
}
