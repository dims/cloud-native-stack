# Cloud Native Stack: Migration from Documentation-Driven to CLI Bundle Generation Approach

## Overview

This document provides a comprehensive comparison between the traditional documentation-driven installation approach and the new CLI-based bundle generation approach for deploying NVIDIA Cloud Native Stack components.

---

## PREVIOUS APPROACH: Documentation-Driven Installation

### Structure

- **docs/install-guides/**: 14+ platform/version-specific markdown guides (e.g., Ubuntu-24-04_Server_x86-arm64_v16.0.md)
- **docs/playbooks/**: Ansible automation with version-specific YAML files (cns_values_14.0.yaml, 15.0, 16.0, etc.)
- **docs/optimizations/**: Hardware-specific tuning guides (GB200-NVL72.md)
- **docs/troubleshooting/**: Issue resolution guides

### Characteristics

#### 1. Manual, Step-by-Step Instructions

Each install guide contains ~1,144 lines covering:
- Install OS (Ubuntu 24.04)
- Install container runtime (Containerd 2.1.3 OR CRI-O 1.33.2)
- Install Kubernetes (1.33.2)
- Install Helm (3.18.3)
- Install Network Operator (optional)
- Install GPU Operator with specific flags:

```bash
helm install --version 25.3.4 --create-namespace \
  --namespace nvidia-gpu-operator nvidia/gpu-operator \
  --set driver.version=580.82.07 \
  --set driver.rdma.enabled=true \
  --set gds.enabled=true \
  --wait --generate-name
```

#### 2. Ansible Playbook Approach

**Version Configuration Files:**
- `cns_values_16.0.yaml` - 141 lines of configuration
- **95+ configuration parameters** including:
  - Component versions (containerd, k8s, gpu_operator, network_operator, etc.)
  - GPU Operator settings (driver_version, enable_mig, enable_gds, enable_cdi, etc.)
  - NGC registry credentials
  - Network operator settings (enable_rdma, deploy_ofed)
  - Storage, monitoring, KServe, LeaderWorkerSet options

**Supporting Playbooks:**
- Pre-requisite playbooks (prerequisites.yaml, k8s-install.yaml)
- Operator-specific playbooks (gpu_operator.yaml with 259 lines mapping 18 GPU Operator releases)

#### 3. Version Matrix Maintenance

**Complex Version Tracking:**
- Component Matrix tables tracking 13+ components across 3 CNS versions
- `gpu_operator.yaml`: Maps component versions for 18+ GPU Operator releases (v25.3.4 → v23.9.1)
- Release lifecycle management (GA, Maintenance, EOL)

Example from gpu_operator.yaml:
```yaml
release_25_3_4:
  gpu_operator_version: v25.3.4
  gpu_driver_version: 580.82.07
  driver_manager_version: 0.8.0
  container_toolkit: v1.17.8
  device_plugin: v0.17.3
  dcgm_exporter_version: 4.2.3-4.1.3
  nfd_version: v0.17.2
  gfd_version: v0.17.1
  mig_manager_version: v0.12.2
  dcgm_version: 4.2.3-1
  validator_version: v25.3.4
  gds_driver: 2.20.5
```

#### 4. Workflow

```
User reads docs → Follows manual steps → Copies commands → 
Adjusts for environment → Executes → Troubleshoots → Repeats
```

#### 5. Challenges

- ❌ **Documentation Drift**: 14 install guides × 3 versions × updates = high maintenance burden
- ❌ **Copy-Paste Errors**: Users must manually type/copy commands with specific flags
- ❌ **Version Mismatches**: Easy to mix incompatible component versions
- ❌ **Platform Variations**: Different guides for Ubuntu 22.04 vs 24.04, x86 vs ARM, Developer vs Production
- ❌ **Configuration Complexity**: 95+ Ansible variables to understand and configure
- ❌ **No Verification**: No built-in way to validate configuration before deployment
- ❌ **Update Lag**: Documentation updates lag behind new releases
- ❌ **Testing Difficulty**: Cannot easily test documentation accuracy in CI/CD

---

## NEW APPROACH: CLI Bundle Generation

### Structure

**Implementation:**
- **pkg/bundler/X/**: Go-based bundler implementation
  - `bundler.go`: Core logic
  - `helm.go`: Helm values generation
  - `manifests.go`, `scripts.go`: Manifest and script generation
  - `templates/`: 5 Go templates (values.yaml.tmpl, clusterpolicy.yaml.tmpl, install.sh.tmpl, etc.)

### Characteristics

#### 1. Recipe-Driven Generation

**Three-Step Workflow:**
```
eidos snapshot → eidos recipe → eidos bundle
```

- **Snapshot**: Captures actual system state (OS, GPU, K8s, SystemD services)
- **Recipe**: Generates optimized recipes based on workload intent (training/inference)
- **Bundle**: Creates deployment-ready bundles tailored to environment

#### 2. Bundle Output Structure

Example based on the GPU Operator:

```
gpu-operator/
├── values.yaml              # Generated Helm configuration
├── manifests/
│   └── clusterpolicy.yaml   # ClusterPolicy manifest
├── scripts/
│   ├── install.sh           # Automated installation (114 lines)
│   └── uninstall.sh         # Cleanup script
├── README.md                # Generated documentation (170 lines)
└── checksums.txt            # SHA256 verification
```

#### 3. Template-Based Generation

**values.yaml.tmpl**: Generates Helm values from recipe measurements
- Extracts driver_version, enable_gds, mig_strategy from recipe
- Applies optimizations based on GPU type (H100, GB200)
- Includes namespace, labels, annotations

**install.sh.tmpl**: Generates executable script with:
- Prerequisite checks (kubectl, helm, cluster connectivity)
- Namespace creation
- Helm repo setup
- GPU Operator installation with `--values values.yaml`
- Verification steps (pod readiness, ClusterPolicy checks)
- Color-coded logging (info, warn, error)

#### 4. Data Extraction from Recipe

```go
// helm.go extracts from recipe measurements:
- Type: K8s → gpu_operator_version, container_toolkit_version
- Type: GPU → driver_version, enable_gds, mig_strategy
- Type: OS → platform-specific optimizations
- Type: SystemD → service configurations
```

#### 5. Workflow Comparison

**End-to-End Process:**
```
System → Snapshot → Recipe (with intent) → Bundle → Deploy
```

**Step-by-Step:**
1. **Snapshot**: Captures 4 measurement types (SystemD, OS, K8s, GPU)
2. **Recipe**: Matches rules based on os/gpu/intent, returns optimized config
3. **Bundle**: Generates deployment artifacts in seconds
4. **Deploy**: Execute `./scripts/install.sh` or use Helm directly

**Example Commands:**
```bash
# Step 1: Capture system snapshot
eidos snapshot --output snapshot.yaml

# Step 2: Generate optimized recipe for training workloads
eidos recipe --snapshot snapshot.yaml --intent training --output recipe.yaml

# Step 3: Create deployment bundle
eidos bundle --recipe recipe.yaml --output ./bundles

# Step 4: Deploy GPU Operator
cd bundles/gpu-operator
chmod +x scripts/install.sh
./scripts/install.sh
```

#### 6. Advantages

- ✅ **Single Source of Truth**: Recipe data (data-v1.yaml) drives all bundles
- ✅ **Version Correctness**: Recipe engine ensures compatible component versions
- ✅ **Environment-Specific**: Bundle matches actual system capabilities
- ✅ **Reproducible**: Checksums ensure file integrity
- ✅ **Self-Documenting**: Generated README includes prerequisites and instructions
- ✅ **Automated Verification**: Install script includes health checks
- ✅ **Extensible**: Add new bundlers (network-operator) via registry pattern
- ✅ **Testable**: Bundle generation can be tested in CI/CD
- ✅ **Fast Updates**: Change recipe data → regenerate bundles instantly
- ✅ **Error Prevention**: Generated code reduces human errors
- ✅ **Parallel Execution**: Multiple bundlers run concurrently

---

## Key Differences

| Aspect | Documentation-Driven | CLI Bundle Generation |
|--------|---------------------|----------------------|
| **Configuration Source** | Human-written markdown + Ansible YAML | Machine-generated from recipes |
| **Version Management** | Manual updates across 14+ guides | Centralized recipe data (data-v1.yaml) |
| **Customization** | Edit 95+ Ansible variables | Specify intent + GPU type |
| **Validation** | Manual verification post-install | Built into install scripts |
| **Maintenance** | Update docs for each CNS version | Update recipe rules once |
| **Error Prevention** | Copy-paste errors common | Generated code reduces errors |
| **Platform Support** | Separate guides per platform | Single workflow adapts to platform |
| **Testing** | Manual testing of docs | Automated bundle generation testing |
| **User Experience** | Read → understand → execute → debug | Snapshot → recipe → bundle → deploy |
| **Time to Deploy** | Hours (reading + execution) | Minutes (automated workflow) |
| **Version Compatibility** | User must manually verify | Recipe engine ensures compatibility |
| **Documentation Updates** | Must update 14+ files per release | Update recipe data once |
| **Reproducibility** | Depends on user following steps | Checksums verify bundle integrity |
| **Extensibility** | Add new playbooks and docs | Implement bundler interface |

---

## Migration Path Analysis

### What's Currently Covered

**CLI Bundle Approach (Implemented):**
- ✅ GPU Operator deployment
- ✅ Helm values generation
- ✅ ClusterPolicy manifests
- ✅ Installation/uninstallation scripts
- ✅ README documentation
- ✅ Checksum verification
- ✅ Intent-based optimization (training/inference)

### What's Missing in New Approach

#### 1. Network Operator Bundle
**Status**: Not yet implemented  
**Required**: `pkg/bundler/networkoperator/` similar to gpuoperator

**Would Include:**
- Templates for RDMA, SR-IOV, OFED configurations
- Network definition manifests
- IPAM configuration
- Mellanox/ConnectX NIC setup

#### 2. Full Stack Installation
**Status**: Still in documentation/playbooks  
**Not Covered by Bundles:**
- Container runtime installation (Containerd/CRI-O)
- Kubernetes cluster setup (kubeadm, MicroK8s)
- Helm installation
- Base system prerequisites

**Reasoning**: These are foundational infrastructure components that bundles layer on top of.

#### 3. Platform-Specific Optimizations
**Current Location**: docs/optimizations/GB200-NVL72.md

**Example GB200 Optimizations:**
```bash
# Boot parameters
init_on_alloc=0 
numa_balancing=disable 
iommu.passthrough=1
```

**Potential**: Could be embedded in recipe overlays for GB200 GPU type and automatically included in generated bundles.

#### 4. Add-On Services
**Status**: In playbooks but not bundlers

**Not Yet Bundled:**
- **KServe** (Istio, Knative, CertManager)
- **Monitoring** (Prometheus, Grafana, Elastic)
- **Storage** (NFS, Local Path Provisioner)
- **LoadBalancer** (MetalLB)
- **LeaderWorkerSet**
- **NIM Operator**
- **Nsight Operator**

**Potential**: Each could have dedicated bundler implementation.

#### 5. Troubleshooting Automation
**Current Location**: docs/troubleshooting/

**Potential Enhancements:**
- Add validation/diagnostic commands to bundles
- Include common issue detection in install scripts
- Generate troubleshooting checklists based on detected environment

### Migration Priority

**Phase 1: Core Operators (Current)**
- ✅ GPU Operator bundler (completed)
- 🔄 Network Operator bundler (next priority)

**Phase 2: Add-On Services**
- Monitoring stack bundler (Prometheus/Grafana)
- Storage bundler (NFS/Local Path)
- KServe bundler
- LoadBalancer bundler (MetalLB)

**Phase 3: Platform Optimizations**
- Embed GB200 optimizations in recipes
- H100/A100-specific tuning
- AWS/Azure/GKE platform-specific configurations

**Phase 4: Integration**
- Full-stack bundle orchestration
- Multi-bundler dependency management
- End-to-end deployment workflows

---

## Hybrid Strategy

### Keep Documentation For

**Essential Documentation:**
1. **Prerequisites**
   - OS installation procedures
   - Hardware setup and verification
   - Network configuration
   - BIOS/firmware settings

2. **Kubernetes Cluster Bootstrapping**
   - Control plane setup
   - Worker node joining
   - Network plugin selection
   - Storage class configuration

3. **Conceptual Architecture**
   - System design and component relationships
   - Best practices and recommendations
   - Security considerations
   - Performance tuning principles

4. **Troubleshooting**
   - Common issues and resolutions
   - Diagnostic procedures
   - Known limitations
   - Support escalation paths

### Migrate to Bundles

**Operator and Service Deployments:**
1. **GPU Operator Deployment** ✅ Completed
   - Helm values generation
   - ClusterPolicy configuration
   - Driver installation
   - Device plugin setup

2. **Network Operator Deployment** 🎯 Next Priority
   - RDMA configuration
   - SR-IOV setup
   - OFED driver deployment
   - Network definitions

3. **Add-On Services** 📋 Future
   - Monitoring stack deployment
   - Storage provisioners
   - KServe deployment
   - LoadBalancer configuration
   - Platform-specific optimizations

### Update Documentation Strategy

**New Documentation Approach:**
1. **Reference CLI Workflow**
   - Replace manual Helm commands with bundle generation workflow
   - Update examples to show: snapshot → recipe → bundle → deploy
   - Keep Ansible playbooks for full-stack automation scenarios

2. **Maintain Ansible Playbooks**
   - Use for complete infrastructure provisioning
   - Keep for environments requiring full automation
   - Position as complementary to CLI bundles

3. **Bundle Approach for Operators**
   - Primary recommendation for operator deployments
   - Document bundle customization procedures
   - Show advanced use cases
