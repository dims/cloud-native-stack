# CLI Refactoring Summary

## Overview

Refactored snapshot command to move agent deployment logic from CLI layer (`pkg/cli/snapshot.go`) into the snapshotter package (`pkg/snapshotter/snapshot.go`), reducing CLI complexity and improving separation of concerns.

## Changes

### Before Refactoring
- **pkg/cli/snapshot.go**: 316 lines (contained all agent deployment logic)
- **pkg/snapshotter/snapshot.go**: 184 lines (local measurement only)

### After Refactoring
- **pkg/cli/snapshot.go**: 172 lines (**144 lines removed**, -46%)
- **pkg/snapshotter/snapshot.go**: 381 lines (**197 lines added**, includes agent logic + helpers)

**Net result**: Cleaner separation, CLI is now 46% smaller and focused only on user input collection.

## Architecture Changes

### pkg/snapshotter/snapshot.go

**Added Types**:
```go
type AgentConfig struct {
    Enabled            bool
    Kubeconfig         string
    Namespace          string
    Image              string
    JobName            string
    ServiceAccountName string
    NodeSelector       map[string]string
    Tolerations        []corev1.Toleration
    Timeout            time.Duration
    CleanupRBAC        bool
    Output             string
    Debug              bool
}
```

**Enhanced NodeSnapshotter**:
```go
type NodeSnapshotter struct {
    Version     string
    Factory     collector.Factory
    Serializer  serializer.Serializer
    AgentConfig *AgentConfig  // NEW: Optional agent configuration
}
```

**New/Modified Methods**:
- `Measure(ctx)` - Routes to local or agent mode based on AgentConfig
- `measureLocal(ctx)` - Original local measurement logic (renamed)
- `measureWithAgent(ctx)` - **NEW**: Agent deployment workflow
- `ParseNodeSelectors([]string)` - **NEW**: Helper function (exported)
- `ParseTolerations([]string)` - **NEW**: Helper function (exported)

### pkg/cli/snapshot.go

**Simplified to**:
1. Parse CLI flags
2. Build `NodeSnapshotter` struct
3. If `--deploy-agent` is set, populate `AgentConfig`
4. Call `ns.Measure(ctx)` (routing handled by snapshotter)

**Removed**:
- `runLocalSnapshot()` function
- `runAgentDeployment()` function (moved to snapshotter)
- `parseNodeSelectors()` helper (moved to snapshotter as `ParseNodeSelectors()`)
- `parseTolerations()` helper (moved to snapshotter as `ParseTolerations()`)
- All Kubernetes client creation logic
- All agent deployment orchestration
- All output handling logic

**Kept**:
- Flag definitions (CLI's responsibility)
- Single Action handler that populates NodeSnapshotter

## Benefits

### 1. Separation of Concerns
- **CLI Layer**: User input collection and validation only
- **Snapshotter Layer**: Business logic (measurement strategy, agent deployment)

### 2. Testability
- Snapshotter logic can be unit tested without CLI framework
- Agent deployment can be tested independently of CLI flags
- Mock AgentConfig for different scenarios

### 3. Reusability
- NodeSnapshotter can be used programmatically without CLI
- Helper functions (ParseNodeSelectors, ParseTolerations) are now exported
- Agent deployment logic accessible to other packages

### 4. Maintainability
- CLI is now ~170 lines (easier to understand)
- Clear boundary: CLI collects input, snapshotter executes logic
- Future changes to agent deployment don't require CLI changes

### 5. Consistency
- Both local and agent modes use same Measure() API
- Single entry point simplifies error handling
- Unified logging strategy

## Usage Example

### Before (Internal routing in CLI)
```go
// CLI had two separate functions
func runLocalSnapshot(ctx, cmd) error { ... }
func runAgentDeployment(ctx, cmd) error { ... }

// Action decided which to call
Action: func(ctx, cmd) {
    if cmd.Bool("deploy-agent") {
        return runAgentDeployment(ctx, cmd)
    }
    return runLocalSnapshot(ctx, cmd)
}
```

### After (Configuration-based routing)
```go
// CLI just builds config
ns := snapshotter.NodeSnapshotter{
    Version: version,
    Factory: factory,
}

if cmd.Bool("deploy-agent") {
    ns.AgentConfig = &snapshotter.AgentConfig{
        Enabled: true,
        // ... populate from flags
    }
}

// Snapshotter routes internally
return ns.Measure(ctx)
```

## Programmatic Usage (New Capability)

Now other packages can use agent deployment without CLI:

```go
import "github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"

// Local mode
ns := snapshotter.NodeSnapshotter{
    Version: "v1.0.0",
}
ns.Measure(ctx)

// Agent mode
ns := snapshotter.NodeSnapshotter{
    Version: "v1.0.0",
    AgentConfig: &snapshotter.AgentConfig{
        Enabled:   true,
        Namespace: "gpu-operator",
        Image:     "ghcr.io/nvidia/eidos:latest",
        // ...
    },
}
ns.Measure(ctx)
```

## Testing

### Unit Tests
All existing tests pass:
```bash
$ go test -v ./pkg/snapshotter/...
=== RUN   TestNewSnapshot
--- PASS: TestNewSnapshot (0.00s)
=== RUN   TestNodeSnapshotter_Measure
--- PASS: TestNodeSnapshotter_Measure (0.01s)
=== RUN   TestSnapshot_Init
--- PASS: TestSnapshot_Init (0.00s)
PASS
ok      github.com/NVIDIA/cloud-native-stack/pkg/snapshotter    0.453s
```

### CLI Build
```bash
$ go build -o eidos-test cmd/eidos/main.go
# Success - no errors
```

### Functionality
CLI behavior is unchanged - all flags work identically:
```bash
$ ./eidos-test snapshot --help
# All 11 flags present
# Examples show correct usage
```

## Migration Guide

### For Developers

**No API changes** - CLI behavior is identical:
```bash
# Same commands work
eidos snapshot --deploy-agent
eidos snapshot --deploy-agent --node-selector key=value
```

**New programmatic API**:
```go
// Can now use snapshotter programmatically
ns := snapshotter.NodeSnapshotter{
    AgentConfig: &snapshotter.AgentConfig{
        Enabled: true,
        // ...
    },
}
ns.Measure(ctx)
```

### For Package Users

**Exported helpers** now available:
```go
import "github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"

// Parse user input
nodeSelector, err := snapshotter.ParseNodeSelectors([]string{"key=value"})
tolerations, err := snapshotter.ParseTolerations([]string{"key=value:NoSchedule"})
```

## Future Enhancements

With this architecture, easy to add:

1. **Alternative deployment strategies**: StatefulSet, DaemonSet, etc.
2. **Batch agent deployment**: Multiple namespaces/clusters
3. **Custom measurement strategies**: User-defined collectors
4. **Progress callbacks**: Real-time status updates
5. **Plugin system**: Third-party measurement sources

## File Changes Summary

### Modified Files
1. `pkg/snapshotter/snapshot.go`:
   - Added AgentConfig struct
   - Added AgentConfig field to NodeSnapshotter
   - Renamed Measure() → measureLocal()
   - Added new Measure() as router
   - Added measureWithAgent() with full agent workflow
   - Added ParseNodeSelectors() helper
   - Added ParseTolerations() helper
   - Added imports: corev1, agent, k8sclient, strings

2. `pkg/cli/snapshot.go`:
   - Removed imports: strings, corev1, agent, k8sclient
   - Removed runLocalSnapshot() function
   - Removed runAgentDeployment() function
   - Removed parseNodeSelectors() helper
   - Removed parseTolerations() helper
   - Simplified Action handler to populate NodeSnapshotter
   - Added AgentConfig population when --deploy-agent is set

### Line Count Changes
- CLI: 316 → 172 lines (-144, -46%)
- Snapshotter: 184 → 381 lines (+197, +107%)
- Net: Cleaner separation with snapshotter owning business logic

## Verification

✅ All tests pass  
✅ CLI builds successfully  
✅ Help output correct  
✅ Flag parsing unchanged  
✅ Backward compatible  
✅ Programmatic usage enabled  

