# Cloud Native Stack

Cloud Native Stack (CNS) provides validated configurations for deploying GPU-accelerated Kubernetes infrastructure. The project generates deployment artifacts from system configuration snapshots, ensuring consistent deployments for managed Kubernetes offerings like Amazon EKS, Google GKE, Azure AKS, Oracle OKE, as well as self-managed Kubernetes clusters.

## What Cloud Native Stack Does

CNS captures system state, generates configuration recipes based on hardware and software parameters, and produces deployment bundles. Configurations are derived from field deployments and tested against H100, GB200, A100, and other NVIDIA GPU hardware.

**Key characteristics:**

- **Prescriptive configurations**: Specific component versions and settings validated for GPU workloads
- **Deterministic generation**: Same inputs produce identical outputs across environments  
- **Hardware-aware**: Recipes adapt to GPU type, OS version, Kubernetes distribution, and workload intent (training or inference)

**Three-stage workflow:**

1. **Snapshot**: Capture system configuration (operating system, kernel version, Kubernetes cluster state, GPU hardware)
2. **Recipe**: Generate configuration recommendations based on captured state or query parameters
3. **Bundle**: Create deployment artifacts including Helm values, Kubernetes manifests, and installation scripts

## Components

- **CLI (`eidos`)**: Command-line tool supporting all three workflow stages (snapshot, recipe, bundle)
- **API Server**: HTTP REST API for recipe generation (query mode only). Production deployment: https://cns.dgxc.io
- **Agent**: Kubernetes Job that captures cluster snapshots and writes output to ConfigMaps

## Documentation

The documentation is organized by persona to help you find what you need quickly. Whether you're deploying GPU infrastructure, contributing code to the CNS project, or integrating CNS into your product or service, start with the section that matches your role.

**Note**: Documentation for the previous version (manual installation guides, playbooks, and platform-specific optimizations) is located in [docs/v1](docs/v1).

### For Users

You are responsible for deploying and operating GPU-accelerated Kubernetes clusters. You need practical guides to get CNS running and validated configurations for your specific hardware and workload requirements.

Get started with installing and using Cloud Native Stack:

- **[Installation Guide](docs/user-guide/installation.md)** – Install the eidos CLI (automated script, manual, or build from source)
- **[CLI Reference](docs/user-guide/cli-reference.md)** – Complete command reference with examples
- **[Agent Deployment](docs/user-guide/agent-deployment.md)** – Deploy the Kubernetes agent to get automated configuration snapshots

### For Developers

You're a software engineer looking to contribute code, extend functionality, or understand CNS internals. You need development setup instructions, architecture documentation, and guidelines for adding new features like bundlers or collectors.

Learn how to contribute and understand the architecture:

- **[Contributing Guide](CONTRIBUTING.md)** – Development setup, testing, and PR process
- **[Architecture Overview](docs/architecture/README.md)** – System design and components
- **[Bundler Development](docs/architecture/bundler-development.md)** – How to create new bundlers
- **[Data Architecture](docs/architecture/data.md)** – Recipe data model and query matching

### For Integrators

You are integrating CNS into CI/CD pipelines, GitOps workflows, or existing product or service. You need API documentation, data schemas, and patterns for programmatic interaction with CNS components.

Integrate Cloud Native Stack into your infrastructure automation:

- **[API Reference](docs/integration/api-reference.md)** – REST API endpoints and usage examples
- **[Data Flow](docs/integration/data-flow.md)** – Understanding snapshots, recipes, and bundles
- **[Automation Guide](docs/integration/automation.md)** – CI/CD integration patterns
- **[Kubernetes Deployment](docs/integration/kubernetes-deployment.md)** – Self-hosted API server setup

## Project Resources

- **[Roadmap](ROADMAP.md)** – Feature priorities and development timeline
- **[Transition](docs/MIGRATION.md)** - Migration to CLI/API-based bundle generation
- **[Security](SECURITY.md)** - Security-related resources 
- **[Releases](https://github.com/NVIDIA/cloud-native-stack/releases)** - Binaries, SBOMs, and other artifacts
- **[Issues](https://github.com/NVIDIA/cloud-native-stack/issues)** - Bugs, feature requests, and questions
