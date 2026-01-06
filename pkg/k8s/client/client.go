package client

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	clientOnce   sync.Once
	cachedClient *kubernetes.Clientset
	cachedConfig *rest.Config
	clientErr    error
)

// GetKubeClient returns a singleton Kubernetes client, creating it on first call.
// Subsequent calls return the cached client for connection reuse and reduced overhead.
// This prevents connection exhaustion and reduces load on the Kubernetes API server.
//
// The client automatically discovers configuration from:
//   - KUBECONFIG environment variable
//   - ~/.kube/config (default location)
//   - In-cluster service account (when running as Kubernetes Pod)
//
// For custom kubeconfig paths, use BuildKubeClient directly.
func GetKubeClient() (*kubernetes.Clientset, *rest.Config, error) {
	clientOnce.Do(func() {
		cachedClient, cachedConfig, clientErr = BuildKubeClient("")
	})
	return cachedClient, cachedConfig, clientErr
}

// BuildKubeClient creates a Kubernetes client from the given kubeconfig file.
//
// This function is exported to allow direct client creation with a specific
// kubeconfig path, bypassing the singleton cache. Use GetKubeClient for most
// cases; only use BuildKubeClient when you need explicit control over the
// kubeconfig source (e.g., multi-cluster operations, testing with different configs).
//
// Parameters:
//   - kubeconfig: Path to kubeconfig file. If empty, uses automatic discovery:
//     1. KUBECONFIG environment variable
//     2. ~/.kube/config (if it exists)
//     3. In-cluster configuration (service account)
//
// Returns:
//   - *kubernetes.Clientset: The Kubernetes client
//   - *rest.Config: The rest configuration used to create the client
//   - error: Any error encountered during client creation
//
// Example with custom kubeconfig:
//
//	clientset, config, err := client.BuildKubeClient("/path/to/custom/kubeconfig")
//	if err != nil {
//	    return fmt.Errorf("failed to build client: %w", err)
//	}
func BuildKubeClient(kubeconfig string) (*kubernetes.Clientset, *rest.Config, error) {
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")

		if kubeconfig == "" {
			kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
			if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
				kubeconfig = ""
			}
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build kube config: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return client, config, nil
}
