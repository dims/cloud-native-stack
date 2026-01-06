# eidos demo

## Snapshot 

Using CLI:

``` shell
eidos snapshot \
    --deploy-agent \
    --namespace gpu-operator \
    --image ghcr.io/mchmarny/eidos:latest \
    --toleration dedicated=user-workload:NoSchedule \
    --toleration dedicated=user-workload:NoExecute \
    --node-selector nodeGroup=customer-gpu \
    --cleanup-rbac
```

Outputs: 

```json
{"time":"2026-01-06T06:39:08.064636-08:00","level":"INFO","msg":"deploying agent","module":"eidos","version":"v0.8.12-next","namespace":"gpu-operator"}
{"time":"2026-01-06T06:39:09.4316-08:00","level":"INFO","msg":"agent deployed successfully","module":"eidos","version":"v0.8.12-next"}
{"time":"2026-01-06T06:39:09.431686-08:00","level":"INFO","msg":"waiting for Job completion","module":"eidos","version":"v0.8.12-next","job":"eidos","timeout":300000000000}
{"time":"2026-01-06T06:39:15.679265-08:00","level":"INFO","msg":"job completed successfully","module":"eidos","version":"v0.8.12-next"}
{"time":"2026-01-06T06:39:15.968405-08:00","level":"INFO","msg":"snapshot saved to ConfigMap","module":"eidos","version":"v0.8.12-next","uri":"cm://gpu-operator/eidos-snapshot"}
```

Also available as a proper k8s deployment: [deployments/eidos-agent](../../deployments/eidos-agent).

What it creates: 

```shell
kubectl -n gpu-operator get cm eidos-snapshot -o yaml 
```

## Recipe

```shell
eidos recipe \
    --snapshot cm://gpu-operator/eidos-snapshot \
    --intent training \
    --output recipe.yaml
```

Outputs: 

```json
{"time":"2026-01-06T06:48:47.287757-08:00","level":"INFO","msg":"loading snapshot from","module":"eidos","version":"v0.8.12-next","uri":"cm://gpu-operator/eidos-snapshot"}
{"time":"2026-01-06T06:48:49.079605-08:00","level":"INFO","msg":"recipe generation completed","module":"eidos","version":"v0.8.12-next","output":"recipe.yaml"}
```

Review the recipe:

```shell
open recipe.yaml
```

## Bundle

```shell
eidos bundle \
  --recipe recipe.yaml \
  --bundlers gpu-operator \
  --output ./bundles
```

Outputs: 

```json
{"time":"2026-01-06T06:55:48.783386-08:00","level":"INFO","msg":"generating bundle","module":"eidos","version":"v0.8.12-next","recipeFilePath":"recipe.yaml","outputDir":"./bundles","bundlerTypes":["gpu-operator"]}
{"time":"2026-01-06T06:55:48.82458-08:00","level":"INFO","msg":"bundle generation completed","module":"eidos","version":"v0.8.12-next","success":1,"errors":0,"duration_sec":0.03861875,"summary":"Generated 6 files (8.5 KB) in 39ms. Success: 1/1 bundlers."}
```

Review the created bundles:

```shell
open ./bundles
```
