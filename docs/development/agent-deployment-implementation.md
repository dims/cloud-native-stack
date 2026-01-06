# Agent Deployment Implementation

## Overview

This document describes the implementation of programmatic agent deployment for the `eidos snapshot` command, allowing users to deploy Kubernetes Jobs directly from the CLI without requiring `kubectl`.

## Architecture

### Components

#### 1. pkg/k8s/agent Package (New)

**Purpose**: Programmatic deployment and management of snapshot agent Jobs and RBAC resources.

**Files**:
- `types.go` - Core types (Config, Deployer, CleanupOptions)
- `deployer.go` - Main orchestration (Deploy, WaitForCompletion, GetSnapshot, Cleanup)
- `rbac.go` - RBAC resource management (5 resource types)
- `job.go` - Job lifecycle management
- `wait.go` - Job monitoring and snapshot retrieval
- `deployer_test.go` - Comprehensive test suite (5 test suites, all passing)
- `doc.go` - Package documentation with examples

**Key Design Decisions**:
1. **Smart Reconciliation**: RBAC resources are idempotent (create if missing, reuse if exists), Jobs are always deleted and recreated
2. **Type Safety**: Uses `kubernetes.Interface` for compatibility with fake clients in tests
3. **Selective Cleanup**: Cleanup can optionally preserve RBAC resources for reuse (via `--cleanup-rbac` flag)
4. **Watch-Based Monitoring**: Uses `watch.Watch` API for efficient Job completion monitoring
5. **Error Handling**: Proper error wrapping with context, handles AlreadyExists and NotFound gracefully

#### 2. pkg/k8s/client Enhancements

**Added**:
- `Interface` type alias (`= kubernetes.Interface`) for mock/fake client compatibility
- `GetKubeClientWithConfig(kubeconfig string)` for custom kubeconfig paths
- Updated `GetKubeClient()` return type to `Interface`

**Purpose**: Support agent deployment with custom kubeconfig paths while maintaining singleton pattern.

#### 3. pkg/serializer Enhancements

**Added**:
- `WriteToFile(path string, data []byte)` for writing raw bytes to files

**Purpose**: Enable agent to write retrieved snapshots to output files.

#### 4. pkg/cli/snapshot.go Enhancements

**Dual-Mode Operation**:
1. **Local Mode** (default): Original behavior - capture snapshot on local machine
2. **Agent Deployment Mode** (`--deploy-agent`): Deploy Job to Kubernetes, retrieve snapshot

**New Flags** (11 total):
- `--deploy-agent` - Enable agent deployment mode
- `--kubeconfig` - Override kubeconfig path
- `--namespace` - Deployment namespace (default: gpu-operator)
- `--image` - Agent container image (default: ghcr.io/nvidia/eidos:latest)
- `--job-name` - Override Job name (default: eidos)
- `--service-account-name` - Override ServiceAccount name (default: eidos)
- `--node-selector` - Node selector for scheduling (format: key=value, repeatable)
- `--toleration` - Toleration for scheduling (format: key=value:effect, repeatable)
- `--timeout` - Wait timeout (default: 5m)
- `--cleanup-rbac` - Remove RBAC on cleanup (default: keep for reuse)

**Agent Deployment Workflow**:
1. Create Kubernetes client (respects `--kubeconfig`)
2. Build agent.Config from CLI flags
3. Deploy RBAC and Job using agent.Deployer
4. Wait for Job completion (with timeout)
5. Retrieve snapshot from ConfigMap
6. Write to output (stdout, file, or ConfigMap)
7. Cleanup Job (optionally remove RBAC)

**Helper Functions**:
- `parseNodeSelectors([]string)` - Parse "key=value" format into map
- `parseTolerations([]string)` - Parse "key=value:effect" into []corev1.Toleration

## Usage Examples

### Local Snapshot (Original Behavior)
```bash
# Capture snapshot on local machine
eidos snapshot --output snapshot.yaml

# Same as before - no changes to local mode
eidos snapshot --format json > snapshot.json
```

### Agent Deployment Mode
```bash
# Basic agent deployment
eidos snapshot --deploy-agent

# With custom kubeconfig
eidos snapshot --deploy-agent --kubeconfig ~/.kube/prod-cluster

# Custom namespace and image
eidos snapshot --deploy-agent \
  --namespace gpu-operator \
  --image ghcr.io/nvidia/eidos:v0.8.0

# With node selector (target specific nodes)
eidos snapshot --deploy-agent \
  --node-selector accelerator=nvidia-h100 \
  --node-selector zone=us-west1-a

# With tolerations (schedule on tainted nodes)
eidos snapshot --deploy-agent \
  --toleration nvidia.com/gpu=present:NoSchedule

# Full example with all options
eidos snapshot --deploy-agent \
  --kubeconfig ~/.kube/config \
  --namespace gpu-operator \
  --image ghcr.io/nvidia/eidos:latest \
  --job-name snapshot-gpu-nodes \
  --service-account-name eidos \
  --node-selector accelerator=nvidia-h100 \
  --toleration nvidia.com/gpu:NoSchedule \
  --timeout 10m \
  --output cm://gpu-operator/eidos-snapshot \
  --cleanup-rbac
```

### Workflow Integration
```bash
# Step 1: Deploy agent and capture snapshot to ConfigMap
eidos snapshot --deploy-agent \
  --namespace gpu-operator \
  --output cm://gpu-operator/eidos-snapshot

# Step 2: Generate recipe from ConfigMap snapshot
eidos recipe \
  --snapshot cm://gpu-operator/eidos-snapshot \
  --intent training \
  --output recipe.yaml

# Step 3: Create deployment bundle
eidos bundle \
  --recipe recipe.yaml \
  --bundlers gpu-operator \
  --output ./bundles
```

## Testing

### Unit Tests
All tests passing (5 test suites):
```bash
$ go test -v ./pkg/k8s/agent/...

TestDeployer_EnsureRBAC (5 subtests)
  ✅ create_ServiceAccount
  ✅ create_Role
  ✅ create_RoleBinding
  ✅ create_ClusterRole
  ✅ create_ClusterRoleBinding

TestDeployer_EnsureRBAC_Idempotent
  ✅ RBAC resources are idempotent

TestDeployer_EnsureJob (2 subtests)
  ✅ create_Job
  ✅ recreate_Job_deletes_old_one

TestDeployer_Deploy
  ✅ Full deployment workflow

TestDeployer_Cleanup
  ✅ Cleanup with/without RBAC removal

PASS (0.559s)
```

### CLI Build Verification
```bash
$ go build -o eidos-test cmd/eidos/main.go
# Success - no errors

$ ./eidos-test snapshot --help
# All flags present and documented
```

## Implementation Details

### RBAC Resources
The agent deployment creates 5 RBAC resources:

1. **ServiceAccount** (`eidos`)
   - Namespace: gpu-operator (configurable)
   - Purpose: Identity for Job pods

2. **Role** (`eidos`)
   - Rules: configmaps (get, list, create, update)
   - Purpose: ConfigMap access for snapshot storage

3. **RoleBinding** (`eidos`)
   - Binds: ServiceAccount → Role
   - Purpose: Grant namespace-scoped ConfigMap permissions

4. **ClusterRole** (`eidos`)
   - Rules: nodes (get, list), pods (get, list)
   - Purpose: Cluster-wide resource read access

5. **ClusterRoleBinding** (`eidos`)
   - Binds: ServiceAccount → ClusterRole
   - Purpose: Grant cluster-scoped read permissions

### Job Specification
The Job matches the YAML in `deployments/eidos-agent/2-job.yaml`:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: eidos
  namespace: gpu-operator
spec:
  template:
    spec:
      serviceAccountName: eidos
      hostPID: true
      hostNetwork: true
      hostIPC: true
      restartPolicy: Never
      nodeSelector: {}  # Configurable via --node-selector
      tolerations: []   # Configurable via --toleration
      containers:
      - name: eidos
        image: ghcr.io/nvidia/eidos:latest  # Configurable via --image
        command: ["eidos", "snapshot", "-o", "cm://gpu-operator/eidos-snapshot"]
        securityContext:
          privileged: true
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
        volumeMounts:
        - name: host
          mountPath: /host
          readOnly: true
        - name: nvidia
          mountPath: /run/nvidia
          readOnly: true
      volumes:
      - name: host
        hostPath:
          path: /
          type: Directory
      - name: nvidia
        hostPath:
          path: /run/nvidia
          type: Directory
```

### Reconciliation Strategy

**RBAC Resources (Idempotent)**:
- Check if resource exists
- If exists: reuse (no-op)
- If missing: create
- Errors: ignore AlreadyExists

**Job (Always Recreate)**:
1. Delete existing Job (if present)
2. Wait for deletion to complete
3. Create fresh Job
4. Rationale: Ensures clean state for each snapshot

**Cleanup Behavior**:
- Default: Delete Job only, keep RBAC for next run
- With `--cleanup-rbac`: Delete Job + all RBAC resources
- Rationale: Avoid repeated RBAC recreation for frequent snapshots

### Error Handling

**Context Cancellation**:
- All operations respect context cancellation
- Timeouts propagate through errgroup
- Watch operations honor context deadline

**Kubernetes Errors**:
- `AlreadyExists`: Ignored during RBAC creation (idempotency)
- `NotFound`: Ignored during deletion (idempotency)
- Other errors: Wrapped with context and returned

**Job Failures**:
- JobFailed condition: Returns error with logs
- Timeout: Returns context.DeadlineExceeded
- Pod logs: Retrieved for debugging

## Migration from kubectl Workflow

### Before (Manual)
```bash
# Deploy dependencies
kubectl apply -f deployments/eidos-agent/1-deps.yaml

# Deploy job
kubectl apply -f deployments/eidos-agent/2-job.yaml

# Wait for completion
kubectl wait --for=condition=complete job/eidos -n gpu-operator

# Get snapshot
kubectl get configmap eidos-snapshot -n gpu-operator -o yaml > snapshot.yaml

# Cleanup
kubectl delete job/eidos -n gpu-operator
```

### After (CLI)
```bash
# Single command
eidos snapshot --deploy-agent --output snapshot.yaml
```

### Benefits
1. **Simplicity**: Single command vs 5 manual steps
2. **Automation**: No manual waiting or ConfigMap parsing
3. **Error Handling**: Automatic retry and error reporting
4. **Flexibility**: Configurable via flags (namespace, image, node-selector, etc.)
5. **Integration**: Works with ConfigMap URIs for recipe generation

## Next Steps

### Completed ✅
- [x] Design and implement pkg/k8s/agent package
- [x] Add Interface type to pkg/k8s/client
- [x] Add GetKubeClientWithConfig() for custom kubeconfig paths
- [x] Add WriteToFile() to pkg/serializer
- [x] Integrate agent deployment into CLI snapshot command
- [x] Add --kubeconfig flag support
- [x] Add 11 configuration flags for agent deployment
- [x] Implement dual-mode operation (local vs agent)
- [x] Write comprehensive unit tests (all passing)
- [x] Build and verify CLI binary

### Pending
- [ ] Functional testing with real Kubernetes cluster
- [ ] Integration testing with GPU nodes
- [ ] Update user guide documentation (docs/user-guide/agent-deployment.md)
- [ ] Update architecture documentation (docs/architecture/cli.md)
- [ ] Update CLI reference with new flags
- [ ] Update tools/e2e script to use `--deploy-agent` flag
- [ ] Test edge cases (timeout, node selector, toleration)
- [ ] Test cleanup modes (with/without RBAC removal)
- [ ] Performance testing (large clusters, multiple snapshots)

## Testing Checklist

### Unit Tests ✅
- [x] RBAC resource creation (5 resources)
- [x] RBAC idempotency
- [x] Job creation
- [x] Job recreation (delete old, create new)
- [x] Full deployment workflow
- [x] Cleanup with/without RBAC removal
- [x] Fake client compatibility

### Integration Tests (Pending)
- [ ] Deploy to real cluster
- [ ] Verify RBAC resources created correctly
- [ ] Verify Job runs on GPU nodes
- [ ] Verify snapshot captured and stored in ConfigMap
- [ ] Verify snapshot retrieval
- [ ] Verify cleanup (Job only)
- [ ] Verify cleanup with RBAC removal
- [ ] Test custom kubeconfig path
- [ ] Test node selectors
- [ ] Test tolerations
- [ ] Test timeout scenarios
- [ ] Test concurrent snapshots (RBAC reuse)

### E2E Tests (Pending)
- [ ] Full workflow: snapshot → recipe → bundle
- [ ] ConfigMap-based workflow
- [ ] Multiple snapshots (RBAC reuse)
- [ ] Cleanup between runs

## Documentation Updates Required

1. **docs/user-guide/agent-deployment.md**
   - Usage examples
   - Flag descriptions
   - Troubleshooting

2. **docs/architecture/cli.md**
   - Agent deployment architecture
   - Component interaction diagram
   - Design decisions

3. **docs/integration/cli-reference.md**
   - New flags reference
   - Examples for each flag combination

4. **README.md**
   - Quick start with agent deployment
   - Replace kubectl examples

5. **CONTRIBUTING.md**
   - Testing agent deployment locally
   - E2E test updates

## Known Limitations

1. **Kubeconfig Context**: Uses active context from kubeconfig (no explicit context selection)
2. **Multiple Namespaces**: Single deployment per namespace (no multi-namespace support)
3. **Concurrent Jobs**: Job name collision if multiple snapshots run simultaneously (use --job-name to avoid)
4. **Image Pull Policy**: Defaults to IfNotPresent (no override flag)
5. **Resource Limits**: No flag to override Job resource requests/limits

## Future Enhancements

1. **Context Selection**: Add `--context` flag to select kubeconfig context
2. **Multi-Namespace**: Support deploying to multiple namespaces
3. **Job Name Generation**: Auto-generate unique Job names with timestamps
4. **Image Pull Policy**: Add flag to override (Always, IfNotPresent, Never)
5. **Resource Overrides**: Add flags for CPU/memory requests/limits
6. **Progress Indicator**: Show deployment progress (RBAC → Job → Wait → Retrieve)
7. **Watch Mode**: Continuous monitoring with real-time log streaming
8. **Parallel Deployment**: Deploy to multiple clusters simultaneously

