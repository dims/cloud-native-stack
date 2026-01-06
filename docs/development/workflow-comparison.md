# Workflow Comparison: kubectl vs CLI Agent Deployment

## Overview

This document compares the old kubectl-based workflow with the new CLI-based agent deployment workflow.

## Quick Comparison

| Aspect | kubectl Workflow | CLI Workflow |
|--------|------------------|--------------|
| Commands | 5+ manual steps | 1 command |
| Dependencies | kubectl, yaml files | eidos CLI only |
| Complexity | Medium | Low |
| Automation | Manual | Automatic |
| Error Handling | Manual checking | Built-in |
| ConfigMap Output | Manual parsing | Direct support |
| Reusability | RBAC recreation | RBAC reuse |
| Flexibility | Edit YAML | CLI flags |

## Old Workflow (kubectl)

### Step-by-Step Process

#### 1. Deploy RBAC Dependencies
```bash
kubectl apply -f deployments/eidos-agent/1-deps.yaml
```

**Creates**:
- ServiceAccount: eidos
- Role: eidos (ConfigMap permissions)
- RoleBinding: eidos
- ClusterRole: eidos (cluster-wide read)
- ClusterRoleBinding: eidos

**Issues**:
- Requires YAML files locally
- No validation before deployment
- Manual error checking

#### 2. Deploy Job
```bash
kubectl apply -f deployments/eidos-agent/2-job.yaml
```

**Creates**:
- Job: eidos (snapshot agent)

**Issues**:
- Job reuse creates stale state
- No automatic cleanup
- Manual Job name collision handling

#### 3. Wait for Completion
```bash
kubectl wait --for=condition=complete job/eidos -n gpu-operator --timeout=5m
```

**Output**:
```
job.batch/eidos condition met
```

**Issues**:
- Manual timeout handling
- No progress indicator
- No automatic retry

#### 4. Retrieve Snapshot
```bash
kubectl get configmap eidos-snapshot -n gpu-operator \
  -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml
```

**Issues**:
- Complex jsonpath syntax
- No error handling
- Manual file writing

#### 5. Cleanup
```bash
# Delete Job
kubectl delete job/eidos -n gpu-operator

# Optionally delete RBAC
kubectl delete -f deployments/eidos-agent/1-deps.yaml
```

**Issues**:
- Manual cleanup
- RBAC deletion unnecessary for reuse
- No selective cleanup

### Full Example
```bash
# Complete kubectl workflow
kubectl apply -f deployments/eidos-agent/1-deps.yaml
kubectl apply -f deployments/eidos-agent/2-job.yaml
kubectl wait --for=condition=complete job/eidos -n gpu-operator --timeout=5m
kubectl get configmap eidos-snapshot -n gpu-operator \
  -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml
kubectl delete job/eidos -n gpu-operator

# If Job failed, need to check logs manually
kubectl logs job/eidos -n gpu-operator
```

### Pain Points

1. **Multi-Step Process**: 5+ commands to execute
2. **Error Handling**: Manual checking at each step
3. **YAML Dependency**: Requires local YAML files
4. **State Management**: Manual Job cleanup, stale state issues
5. **ConfigMap Parsing**: Complex jsonpath syntax
6. **No Reusability**: RBAC recreated each time
7. **No Validation**: No pre-deployment checks
8. **Limited Flexibility**: Must edit YAML for customization

## New Workflow (CLI Agent Deployment)

### Step-by-Step Process

#### Single Command
```bash
eidos snapshot --deploy-agent --output snapshot.yaml
```

**What Happens Automatically**:
1. ✅ Validates Kubernetes connectivity
2. ✅ Creates RBAC resources (idempotent)
3. ✅ Deletes old Job (if exists)
4. ✅ Creates fresh Job
5. ✅ Waits for completion (with timeout)
6. ✅ Retrieves snapshot from ConfigMap
7. ✅ Writes to output file
8. ✅ Cleans up Job (keeps RBAC)

**Output**:
```
Deploying agent to namespace gpu-operator...
RBAC resources created (idempotent)
Job eidos created
Waiting for Job completion (timeout: 5m)...
Job completed successfully
Snapshot retrieved from ConfigMap
Written to snapshot.yaml
Cleanup completed (RBAC preserved for reuse)
```

### Customization Examples

#### Custom Kubeconfig
```bash
eidos snapshot --deploy-agent --kubeconfig ~/.kube/prod-cluster
```

#### Custom Namespace and Image
```bash
eidos snapshot --deploy-agent \
  --namespace my-gpu-ns \
  --image ghcr.io/nvidia/eidos:v0.8.0
```

#### Node Targeting
```bash
eidos snapshot --deploy-agent \
  --node-selector accelerator=nvidia-h100 \
  --node-selector zone=us-west1-a
```

#### Toleration for Tainted Nodes
```bash
eidos snapshot --deploy-agent \
  --toleration nvidia.com/gpu=present:NoSchedule
```

#### Extended Timeout
```bash
eidos snapshot --deploy-agent --timeout 10m
```

#### ConfigMap Output
```bash
eidos snapshot --deploy-agent \
  --output cm://gpu-operator/my-snapshot
```

#### Full Cleanup (Remove RBAC)
```bash
eidos snapshot --deploy-agent --cleanup-rbac
```

### Full Example with All Options
```bash
eidos snapshot --deploy-agent \
  --kubeconfig ~/.kube/config \
  --namespace gpu-operator \
  --image ghcr.io/nvidia/eidos:latest \
  --job-name snapshot-gpu-nodes \
  --service-account-name eidos \
  --node-selector accelerator=nvidia-h100 \
  --node-selector zone=us-west1-a \
  --toleration nvidia.com/gpu:NoSchedule \
  --timeout 10m \
  --output cm://gpu-operator/eidos-snapshot \
  --cleanup-rbac
```

### Advantages

1. **Single Command**: One command vs 5+ manual steps
2. **Automatic Error Handling**: Built-in retry, timeout, validation
3. **No YAML Files**: Everything configured via flags
4. **Smart State Management**: Automatic Job recreation, RBAC reuse
5. **Direct ConfigMap Support**: Native `cm://` URI handling
6. **Reusability**: RBAC preserved by default for subsequent runs
7. **Validation**: Pre-deployment connectivity and permission checks
8. **Flexibility**: 11 configuration flags for customization
9. **Integration**: Works seamlessly with recipe and bundle generation

## Migration Path

### Updating Existing Scripts

#### Before (kubectl)
```bash
#!/bin/bash
set -e

# Deploy agent
kubectl apply -f deployments/eidos-agent/1-deps.yaml
kubectl apply -f deployments/eidos-agent/2-job.yaml

# Wait
kubectl wait --for=condition=complete job/eidos -n gpu-operator --timeout=5m

# Get snapshot
kubectl get configmap eidos-snapshot -n gpu-operator \
  -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml

# Generate recipe
eidos recipe --snapshot snapshot.yaml --intent training --output recipe.yaml

# Cleanup
kubectl delete job/eidos -n gpu-operator
```

#### After (CLI)
```bash
#!/bin/bash
set -e

# Deploy agent and capture snapshot
eidos snapshot --deploy-agent --output snapshot.yaml

# Generate recipe (can also read from ConfigMap directly)
eidos recipe --snapshot snapshot.yaml --intent training --output recipe.yaml

# Or use ConfigMap URIs (no files needed)
eidos snapshot --deploy-agent --output cm://gpu-operator/eidos-snapshot
eidos recipe --snapshot cm://gpu-operator/eidos-snapshot \
  --intent training --output recipe.yaml
```

### Updating CI/CD Pipelines

#### Before (GitHub Actions)
```yaml
- name: Deploy agent dependencies
  run: |
    kubectl apply -f deployments/eidos-agent/1-deps.yaml
    
- name: Deploy agent job
  run: |
    kubectl apply -f deployments/eidos-agent/2-job.yaml
    
- name: Wait for completion
  run: |
    kubectl wait --for=condition=complete job/eidos -n gpu-operator --timeout=5m
    
- name: Get snapshot
  run: |
    kubectl get configmap eidos-snapshot -n gpu-operator \
      -o jsonpath='{.data.snapshot\.yaml}' > snapshot.yaml
    
- name: Cleanup
  run: |
    kubectl delete job/eidos -n gpu-operator
```

#### After (GitHub Actions)
```yaml
- name: Capture snapshot
  run: |
    eidos snapshot --deploy-agent --output snapshot.yaml
```

## Performance Comparison

### kubectl Workflow
```
RBAC deployment:       ~2s
Job deployment:        ~1s
Waiting for Job:       ~30s (varies)
ConfigMap retrieval:   ~1s
Manual cleanup:        ~2s
----------------------------------------
Total:                 ~36s + manual time
```

### CLI Workflow
```
RBAC deployment:       ~2s (first run), ~0s (subsequent)
Job deployment:        ~2s (delete + create)
Waiting for Job:       ~30s (varies)
ConfigMap retrieval:   ~1s (automatic)
Automatic cleanup:     ~1s
----------------------------------------
Total:                 ~35s (first run), ~33s (subsequent)
Saved time:            ~3-5s + human time
```

**Key Improvements**:
- RBAC reuse saves ~2s on subsequent runs
- Eliminates manual steps (saves minutes)
- Automatic error handling (saves debug time)

## Error Handling Comparison

### kubectl Workflow

#### Job Failure
```bash
$ kubectl wait --for=condition=complete job/eidos -n gpu-operator
error: timed out waiting for the condition

# Manual investigation required
$ kubectl logs job/eidos -n gpu-operator
# ... examine logs manually
```

#### RBAC Issues
```bash
$ kubectl apply -f deployments/eidos-agent/1-deps.yaml
Error from server (Forbidden): ...

# Manual debugging required
$ kubectl auth can-i create configmaps --as=system:serviceaccount:gpu-operator:eidos
```

#### ConfigMap Missing
```bash
$ kubectl get configmap eidos-snapshot -n gpu-operator
Error from server (NotFound): configmaps "eidos-snapshot" not found

# Manual investigation required
```

### CLI Workflow

#### Job Failure
```bash
$ eidos snapshot --deploy-agent
Error: Job failed after 2m30s
Job logs:
  Error: failed to collect GPU info: nvidia-smi not found
  
Troubleshooting:
  1. Verify GPU nodes have NVIDIA driver installed
  2. Check node selector matches GPU nodes
  3. Verify image has nvidia-smi in PATH
```

#### RBAC Issues
```bash
$ eidos snapshot --deploy-agent
Error: Failed to create Role in namespace gpu-operator
Reason: Forbidden: User "test@example.com" cannot create resource "roles"

Troubleshooting:
  1. Verify kubeconfig context has required permissions
  2. Check RBAC policies allow Role creation
  3. Try with cluster-admin privileges for initial setup
```

#### ConfigMap Missing
```bash
$ eidos snapshot --deploy-agent
Error: Job completed but ConfigMap not found
Expected: cm://gpu-operator/eidos-snapshot

Troubleshooting:
  1. Check Job logs for errors
  2. Verify ServiceAccount has ConfigMap create permissions
  3. Ensure Job command includes correct ConfigMap URI
```

## Backward Compatibility

### kubectl Workflow Still Works
The old kubectl workflow continues to work:
```bash
# Still functional
kubectl apply -f deployments/eidos-agent/1-deps.yaml
kubectl apply -f deployments/eidos-agent/2-job.yaml
```

### Gradual Migration
Teams can migrate incrementally:
1. Keep existing kubectl scripts
2. Test CLI workflow in dev/staging
3. Migrate production once validated
4. Eventually deprecate YAML files (v1.0.0+)

## Recommendations

### For New Users
✅ **Use CLI workflow** (`--deploy-agent`)
- Simpler
- Better error handling
- Integrated with other commands

### For Existing Users
⚠️ **Migrate to CLI gradually**
- Test in development first
- Update CI/CD pipelines
- Document custom configurations
- Keep YAML files until migration complete

### For Automation
✅ **Use CLI workflow** with flags
- Easier to parameterize
- No file dependencies
- Better for GitOps (no manifest updates)

### For Debugging
✅ **CLI provides better diagnostics**
- Structured error messages
- Automatic log retrieval
- Troubleshooting hints

## Conclusion

The CLI agent deployment workflow provides:
- **Simplicity**: 1 command vs 5+ manual steps
- **Automation**: No manual waiting or parsing
- **Reliability**: Built-in error handling and retry
- **Flexibility**: 11 configuration flags
- **Performance**: RBAC reuse for subsequent runs
- **Integration**: Seamless ConfigMap URI support

**Migration is recommended for all users**, with gradual rollout for existing production deployments.

