# Refactoring

This document outlines the full scope of the refactoring effort for this project. 

## cli

Change the input flags for `recipe` command (pkg/cli/recipe.go) to: 

* `service` - e.g. eks, gke, oke, ake
* `fabric` - e.g efa, ib
* `accelerator` (currently called: `gpu`) - e.g. h100, gb200
* `intent` - e.g training, inference
* `worker` - (worker node OS) - e.g. ubuntu, rhel, cos, awslinux
* `system` - (system node OS) - e.g. ubuntu, rhel, cos, awslinux
* `nodes`: number of worker nodes, 0-N, e.g. 100

Each one of these is optional, if not provided, default to `any`.
Remove `context`, the context will be included by default.
Continue to support `snapshot`.

## recipe

Refactor `pkg/recipe/data/data-v1.yaml` file to 

1) Single base recipe file: `pkg/recipe/data/base.yaml`:

```yaml
kind: recipeMetadata
apiVersion: cns.nvidia.com/v1alpha1
metadata:
  name: base
spec:
  
  # Basic assumptions, will create warning in pre-flight but not block deployment
  constraints:
    - name: k8s
      value: ">= 1.30"
  
  componentRefs:
    - name: cert-manager
      type: Helm
      source: https://charts.jetstack.io
      version: v1.19.2
      valuesFile: cert-manager-values.yaml

    - name: gpu-operator
      type: Helm
      source: https://helm.ngc.nvidia.com/nvidia
      version: v25.3.3
      valuesFile: gpu-operator-values.yaml
      dependencyRefs:
        - cert-manager
        - dra-driver
      
    - name: dra-driver
      type: Kustomize  # Static (already hydrated YAMLs are just Kustomize sources sans patches)
      source: https://raw.githubusercontent.com/NVIDIA/k8s-dra-driver-gpu/refs/heads/main/deployments/controller.yaml
      tag: v25.8.1
      patches:
        - dra-driver-patch1.yaml
        - dra-driver-patch2.yaml
      
    - name: nvsentinel
      type: Helm
      source: oci://ghcr.io/nvidia/nvsentinel
      version: v0.6.0
      # valuesFile: all defaults
      dependencyRefs:
        - cert-manager
```


2) N recipe overlays:

GB200 Training on EKS
File: `pkg/recipe/data/gb200-eks-training.yaml`

```yaml
kind: recipeMetadata
apiVersion: cns.nvidia.com/v1alpha1
metadata:
  name: gb200-eks-training
spec:
  
  criteria:
    service: eks
    accelerator: gb200
    intent: training

  # Basic assumptions, will create warning in pre-flight but not block deployment
  constraints:
    - name: k8s
      value: ">= 1.32.4"
    - name: worker-os
      value: "ubuntu"
    - name: worker-os-version
      value: "24.04"
    - name: kernel
      value: ">= 6.8"
  
  componentRefs:
    - name: gpu-operator
      type: Helm
      source: https://helm.ngc.nvidia.com/nvidia
      version: v25.3.3
      valuesFile: gpu-operator-eks-gb200-training.yaml
```

H100 Inference on Vanilla (upstream) K8s 
File: `pkg/recipe/data/h100-training.yaml`

```yaml
kind: recipeMetadata
apiVersion: cns.nvidia.com/v1alpha1
metadata:
  name: h100-training
spec:
  
  criteria:
    accelerator: h100
    intent: training

  # Basic assumptions, will create warning in pre-flight but not block deployment
  constraints:
    - name: container-toolkit
      value: ">= 1.17"

  componentRefs:
    - name: network-operator
      type: Helm
      source: https://helm.ngc.nvidia.com/nvidia
      version: v25.4.0
      valuesFile: network-operator-h100-ib-training.yaml
```

When user runs the `eidos recipe` command, their input (flags) or the snapshot are used to create a `criteria` object. 
That criteria object is then passed into the recipe engine where the base recipe is loaded (`pkg/recipe/data/base.yaml`), and then overlaid based on user input. For example: 

Command: `eidos recipe --service eks --accelerator gb200 --intent training` would result in base (`pkg/recipe/data/base.yaml`) plus `pkg/recipe/data/gb200-eks-training.yaml`. Just like currently implemented, user's input (i.e. criteria) could match N criteria in the recipeMetadata files. The recipe engine will have a config that will drive the order in which the recipe overlays are being applied (the last one wins). 

The response would be serialized `recipeMetadata` with the compost list of components and constraints, as well as the criteria that were used to generated.
 