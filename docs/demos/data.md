# Recipe Data Architecture Demo

This demo walks through the recipe metadata system, showing how multi-level inheritance, criteria matching, and component configuration work together.

## Prerequisites

Install CLI:

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/main/install | bash -s --
```

Test CLI:

```shell
cnsctl -h
```

## Explore Recipe Data

View embedded recipe files structure:

```shell
tree -L 1 pkg/recipe/data/
```

Expected structure:

```
pkg/recipe/data/
├── base.yaml
├── components
├── eks-training.yaml
├── eks.yaml
├── gb200-eks-training.yaml
├── gb200-eks-ubuntu-training.yaml
├── h100-ubuntu-inference.yaml
└── README.md
```

## Multi-Level Inheritance

View base recipe (foundation for all recipes):

```shell
cat pkg/recipe/data/base.yaml | yq .
```

View EKS recipe (inherits from base):

```shell
cat pkg/recipe/data/eks.yaml | yq .
```

View EKS training recipe (inherits from eks):

```shell
cat pkg/recipe/data/eks-training.yaml | yq .
```

View GB200 EKS training recipe (inherits from eks-training):

```shell
cat pkg/recipe/data/gb200-eks-training.yaml | yq .
```

View leaf recipe (inherits from gb200-eks-training):

```shell
cat pkg/recipe/data/gb200-eks-ubuntu-training.yaml | yq .
```

### Inheritance Chain

```
base.yaml
    │
    └── eks.yaml (service: eks)
            │
            └── eks-training.yaml (service: eks, intent: training)
                    │
                    └── gb200-eks-training.yaml (service: eks, accelerator: gb200, intent: training)
                            │
                            └── gb200-eks-ubuntu-training.yaml (full criteria)
```

## Criteria Matching

### Broad Query (matches multiple overlays)

```shell
cnsctl recipe --service eks | yq .metadata
```

This matches:

```yaml
  appliedOverlays:
    - base
    - eks
```

### More Specific Query

```shell
cnsctl recipe \
    --service eks \
    --intent training \
    | yq .metadata
```

This matches:

```yaml
  appliedOverlays:
    - base
    - eks
    - eks-training
```

### Fully Specific Query

```shell
cnsctl recipe \
    --service eks \
    --accelerator gb200 \
    --intent training \
    | yq .metadata
```

This matches:

```yaml
  appliedOverlays:
    - base
    - eks
    - eks-training
    - gb200-eks-training
```

### Full Criteria Query (with OS)

```shell
cnsctl recipe \
    --service eks \
    --accelerator gb200 \
    --intent training \
    --os ubuntu \
    | yq .metadata
```

This matches all 5 levels:

```yaml
  appliedOverlays:
    - base
    - eks
    - eks-training
    - gb200-eks-training
    - gb200-eks-ubuntu-training
```

## Component Configuration

### View Base Component Values

GPU Operator base values:

```shell
cat pkg/recipe/data/components/gpu-operator/values.yaml | yq .
```

Training-optimized values:

```shell
cat pkg/recipe/data/components/gpu-operator/values-eks-training.yaml | yq .
```

### Value Merge Precedence

Values are merged in order (later = higher priority):

```
Base ValuesFile → Overlay ValuesFile → Overlay Overrides → CLI --set flags
```

Example with CLI overrides:

```shell
cnsctl recipe \
  --service eks \
  --accelerator h100 \
  --intent training \
  --format yaml | yq .componentRefs
```

## Constraints

View constraints for a recipe:

```shell
cnsctl recipe \
  --service eks \
  --accelerator gb200 \
  --intent training \
  --format yaml | yq .constraints
```

Constraint format: `{MeasurementType}.{Subtype}.{Key}`

Examples:
- `K8s.server.version` - Kubernetes version
- `OS.release.ID` - Operating system ID
- `GPU.smi.driver_version` - GPU driver version

## Deployment Order

View computed deployment order (topological sort based on dependencies):

```shell
cnsctl recipe \
  --service eks \
  --accelerator h100 \
  --intent training \
  --format yaml | yq .deploymentOrder
```

Expected order respects `dependencyRefs`:
1. `cert-manager` (no dependencies)
2. `gpu-operator` (depends on cert-manager)
3. Other components...

## API Access

Same recipe via API:

```shell
curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=gb200&intent=training" | jq .
```

View applied overlays:

```shell
curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=gb200&intent=training" | jq .metadata.appliedOverlays
```

## Validation Tests

Run recipe data validation tests:

```shell
go test -v ./pkg/recipe/... -run TestAllMetadataFilesParseCorrectly
```

Check inheritance references:

```shell
go test -v ./pkg/recipe/... -run TestAllBaseReferencesPointToExistingRecipes
```

Check criteria enums:

```shell
go test -v ./pkg/recipe/... -run TestAllOverlayCriteriaUseValidEnums
```

## Links

### Documentation
- [Data Architecture](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/architecture/data.md) - Full architecture documentation
- [Recipe Development Guide](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/integration/recipe-development.md) - Adding/modifying recipes
- [CLI Reference](https://github.com/mchmarny/cloud-native-stack/blob/main/docs/user-guide/cli-reference.md) - Recipe command options

### Source Code
- [Recipe Data Files](https://github.com/mchmarny/cloud-native-stack/tree/main/pkg/recipe/data) - YAML recipe definitions
- [Metadata Store](https://github.com/mchmarny/cloud-native-stack/blob/main/pkg/recipe/metadata_store.go) - Inheritance resolution
- [Criteria Matching](https://github.com/mchmarny/cloud-native-stack/blob/main/pkg/recipe/criteria.go) - Matching algorithm
