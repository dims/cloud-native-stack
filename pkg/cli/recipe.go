package cli

import (
	"fmt"
	"sync"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	ver "github.com/NVIDIA/cloud-native-stack/pkg/version"

	"github.com/spf13/cobra"
)

var (
	// Flags for recipe query parameters
	recOs        string
	recOsVersion string
	recKernel    string
	recService   string
	recK8s       string
	recGPU       string
	recIntent    string
	recContext   bool

	mu sync.RWMutex
)

// recipeCmd represents the recipe command
var recipeCmd = &cobra.Command{
	Use:     "recipe",
	Aliases: []string{"rec"},
	GroupID: "functional",
	Short:   "Generate configuration recipe for a given environment",
	Long: `Generate configuration recipe based on specified environment parameters including:
  - Operating system and version
  - Kernel version
  - Managed service context
  - Kubernetes cluster version
  - GPU type
  - Workload intent

The recipe can be output in JSON, YAML, or table format.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Parse output format
		outFormat := serializer.Format(format)
		if outFormat.IsUnknown() {
			return fmt.Errorf("unknown output format: %q", outFormat)
		}

		q, err := buildQueryFromFlags()
		if err != nil {
			return fmt.Errorf("error parsing recipe input parameter: %w", err)
		}

		rec, err := recipe.BuildRecipe(q)
		if err != nil {
			return fmt.Errorf("error building recipe: %w", err)
		}

		return serializer.NewFileWriterOrStdout(outFormat, output).Serialize(rec)
	},
}

func init() {
	rootCmd.AddCommand(recipeCmd)

	// Define flags for recipe query parameters
	recipeCmd.Flags().StringVarP(&recOs, "os", "", "", "Operating system family (e.g., ubuntu, cos)")
	recipeCmd.Flags().StringVarP(&recOsVersion, "osv", "", "", "Operating system version (e.g., 22.04)")
	recipeCmd.Flags().StringVarP(&recKernel, "kernel", "", "", "Running kernel version (e.g., 5.15.0)")
	recipeCmd.Flags().StringVarP(&recService, "service", "", "", "Managed service context (e.g., eks, gke, or self-managed)")
	recipeCmd.Flags().StringVarP(&recK8s, "k8s", "", "", "Kubernetes cluster version (e.g., v1.25.4)")
	recipeCmd.Flags().StringVarP(&recGPU, "gpu", "", "", "GPU type (e.g., H100, GB200)")
	recipeCmd.Flags().StringVarP(&recIntent, "intent", "", "", "Workload intent (e.g., training or inference)")
	recipeCmd.Flags().BoolVarP(&recContext, "context", "", false, "Include context metadata in the response")

	// Define output format flag specific to recipe command
	recipeCmd.Flags().StringVarP(&output, "output", "", "", "output file path (default: stdout)")
	recipeCmd.Flags().StringVarP(&format, "format", "", "json", "output format (json, yaml, table)")
}

// buildQueryFromFlags constructs a recipe.Query from CLI flags.
func buildQueryFromFlags() (*recipe.Query, error) {
	mu.Lock()
	defer mu.Unlock()

	q := &recipe.Query{}
	var err error

	if recOs != "" {
		q.Os = recipe.OsFamily(recOs)
		if !q.Os.IsValid() {
			return nil, fmt.Errorf("os: %q, supported values: %v", recOs, recipe.SupportedOSFamilies())
		}
	}
	if recOsVersion != "" {
		q.OsVersion, err = ver.ParseVersion(recOsVersion)
		if err != nil {
			return nil, fmt.Errorf("osv: %q: %w", recOsVersion, err)
		}
	}
	if recKernel != "" {
		q.Kernel, err = ver.ParseVersion(recKernel)
		if err != nil {
			return nil, fmt.Errorf("kernel: %q: %w", recKernel, err)
		}
	}
	if recService != "" {
		q.Service = recipe.ServiceType(recService)
		if !q.Service.IsValid() {
			return nil, fmt.Errorf("service: %q, supported values: %v", recService, recipe.SupportedServiceTypes())
		}
	}

	if recK8s != "" {
		q.K8s, err = ver.ParseVersion(recK8s)
		if err != nil {
			return nil, fmt.Errorf("k8s: %q: %w", recK8s, err)
		}
	}
	if recGPU != "" {
		q.GPU = recipe.GPUType(recGPU)
		if !q.GPU.IsValid() {
			return nil, fmt.Errorf("gpu: %q, supported values: %v", recGPU, recipe.SupportedGPUTypes())
		}
	}
	if recIntent != "" {
		q.Intent = recipe.IntentType(recIntent)
		if !q.Intent.IsValid() {
			return nil, fmt.Errorf("intent: %q, supported values: %v", recIntent, recipe.SupportedIntentTypes())
		}
	}

	q.IncludeContext = recContext

	return q, nil
}
