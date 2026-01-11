package gpuoperator

import (
	_ "embed"
)

//go:embed templates/values.yaml.tmpl
var valuesTemplate string

//go:embed templates/clusterpolicy.yaml.tmpl
var clusterPolicyTemplate string

//go:embed templates/install.sh.tmpl
var installScriptTemplate string

//go:embed templates/uninstall.sh.tmpl
var uninstallScriptTemplate string

//go:embed templates/README.md.tmpl
var readmeTemplate string

//go:embed templates/kernel-module-params.yaml.tmpl
var kernelModuleParamsTemplate string

//go:embed templates/dcgm-exporter.yaml.tmpl
var dcgmExporterTemplate string

// GetTemplate returns the named template content.
func GetTemplate(name string) (string, bool) {
	templates := map[string]string{
		"values.yaml":          valuesTemplate,
		"clusterpolicy":        clusterPolicyTemplate,
		"install.sh":           installScriptTemplate,
		"uninstall.sh":         uninstallScriptTemplate,
		"README.md":            readmeTemplate,
		"kernel-module-params": kernelModuleParamsTemplate,
		"dcgm-exporter":        dcgmExporterTemplate,
	}

	tmpl, ok := templates[name]
	return tmpl, ok
}
