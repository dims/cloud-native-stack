# Skyhook Operator Bundle

```shell
Generated from Cloud Native Stack Recipe
Timestamp: 2026-01-11T16:17:47Z
Bundler Version: v0.12.4-next
Recipe Version: v0.12.4-next
```

## Overview

This bundle contains configuration and deployment artifacts for the NVIDIA Skyhook Operator, which provides automated node optimization for GPU workloads in Kubernetes clusters.

## Contents

- `values.yaml` - Helm chart values for Skyhook Operator deployment
- `manifests/` - Skyhook custom resources for node tuning (if customization specified)
- `scripts/install.sh` - Automated installation script
- `scripts/uninstall.sh` - Cleanup script
- `checksums.txt` - SHA256 checksums for file integrity verification

## Configuration

All configuration options are defined in `values.yaml`. Key settings include:

- Operator version and image repositories
- Resource requests and limits
- Node selectors and tolerations

Review and customize `values.yaml` for your environment.

## Installation

### Prerequisites

- Kubernetes cluster (1.29+)
- kubectl configured with cluster access
- Helm 3.x installed
- Cluster admin permissions

### Verify File Integrity

Before installation, verify the checksums:

```bash
sha256sum -c checksums.txt
```

### Quick Install

Run the automated installation script:

```bash
chmod +x scripts/install.sh
./scripts/install.sh
```

### Manual Installation

1. **Create namespace**:
   ```bash
   kubectl create namespace skyhook
   ```

2. **Add Helm repository**:
   ```bash
   helm repo add skyhook oci://ghcr.io/nvidia/skyhook
   helm repo update
   ```

3. **Install operator**:
   ```bash
   helm upgrade --install skyhook skyhook/skyhook \
     --namespace skyhook \
     --create-namespace \
     --values values.yaml \
     --wait
   ```

4. **Apply Skyhook customization** (if manifests/ directory exists):
   ```bash
   if [ -d manifests ]; then
     kubectl apply -f manifests/
   fi
   ```

## Verification

Check operator deployment:

```bash
# Check operator pods
kubectl get pods -n skyhook

# Check Skyhook resources
kubectl get skyhook -A

# View operator logs
kubectl logs -n skyhook -l control-plane=controller-manager -f
```

Verify node tuning:

```bash
# Check node labels
kubectl get nodes --show-labels

# Check node conditions
kubectl describe node <node-name>
```

## Customization

### Modifying Node Selectors

Edit `manifests/skyhook.yaml` to change which nodes are tuned:

```yaml
nodeSelectors:
  matchExpressions:
    - key: your-node-selector-key
      operator: In
      values:
        - your-node-group
```

### Adjusting Resource Limits

Edit `values.yaml` to modify operator resource allocation:

```yaml
controllerManager:
  manager:
    resources:
      limits:
        cpu: 500m
        memory: 512Mi
```

### Custom Tuning Configuration

Modify the configMap section in `manifests/skyhook.yaml` to customize:
- GRUB boot parameters
- Sysctl kernel settings
- Containerd service configuration

## Uninstallation

Run the cleanup script:

```bash
chmod +x scripts/uninstall.sh
./scripts/uninstall.sh
```

Or manually:

```bash
# Delete Skyhook customization (if manifests exist)
if [ -d manifests ]; then
  kubectl delete -f manifests/
fi

# Uninstall Helm release
helm uninstall skyhook -n skyhook

# Delete namespace (optional)
kubectl delete namespace skyhook
```

## Troubleshooting

### Operator Not Starting

Check operator logs:
```bash
kubectl logs -n skyhook deployment/skyhook-operator-controller-manager
```

### Node Not Being Tuned

1. Check node selectors match your nodes:
   ```bash
   kubectl get nodes --show-labels
   ```

2. Check Skyhook status:
   ```bash
   kubectl get skyhook -A
   kubectl describe skyhook <name>
   ```

3. Check agent pods on nodes:
   ```bash
   kubectl get pods -A -o wide | grep skyhook-agent
   ```

### Tuning Not Applied

Check configMap:
```bash
kubectl get configmap -n skyhook | grep skyhook
kubectl describe configmap <configmap-name> -n skyhook
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/NVIDIA/skyhook/issues
- Documentation: https://github.com/NVIDIA/skyhook
