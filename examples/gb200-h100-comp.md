# H100 vs GB200 Node Snapshot Comparison Report

## Files Compared

| System | Source | Node                      |
|--------|--------|---------------------------|
| H100   | AWS    | `ip-10-0-233-106.ec2.internal` |
| GB200  | AWS    | `ip-10-0-205-119.ec2.internal` |

Both snapshots use `snapshot.dgxc.io/v1`, version `v0.5.6`.

> Meaningful config and capability diffs only. Ignores order, timestamps, and other expected runtime noise.

⸻

## 1. High-Level Summary

| Category | Classification | Notes |
|----------|----------------|-------|
| Kernel & Boot | Different | Same kernel family (6.8 AWS), different patch level and flags |
| CPU Architecture | Different | H100 is x86_64; GB200 is ARM64 |
| Crypto Acceleration | Different | Architecture-specific crypto modules |
| NUMA / Memory Policy | Different | Explicit NUMA tuning only on GB200 |
| Kubernetes | Both Present | Different versions: H100 (v1.30.14-eks), GB200 (v1.33.5-eks) |
| Container Runtime | Equivalent | containerd configuration aligned |
| GPU Stack | Equivalent | NVIDIA + GDR present on both |
| Networking / RDMA | Equivalent | EFA + RDMA stacks aligned |
| Docker | Equivalent (disabled) | Inactive on both |
| Kubelet systemd unit | Equivalent (inactive) | Inactive on both |


⸻

## 2. Kernel & Boot Configuration (Grub)

### Kernel Version

| System | Kernel Version |
|--------|----------------|
| H100 | 6.8.0-1024-aws |
| GB200 | 6.8.0-1028-aws |

**Classification:** Patch-level skew only; same kernel line.

### Boot Flags – Real Differences

| Flag | H100 | GB200 |
|------|------|-------|
| init_on_alloc | not set | 0 |
| numa_balancing | default | disable |
| hugepages | 5128 | 5128 |
| hugepagesz | 2M | 2M |
| nokaslr | enabled | enabled |

**Interpretation:** GB200 explicitly disables NUMA auto-balancing and init-on-alloc, indicating tighter control over memory placement and determinism. H100 relies on kernel defaults.

⸻

## 3. CPU Architecture & Crypto Stack

### Architecture Evidence

**H100 (x86_64-oriented modules):**
- aesni_intel
- sha256_ssse3
- ghash_clmulni_intel

**GB200 (ARM64-oriented modules):**
- aes_ce, sha*_ce
- sm3, sm4
- polyval_ce

**Classification:** Fundamental architectural difference. Expected and correct for GB200-class systems.

⸻

## 4. Kernel Module Inventory (KMod)

### GPU / NVIDIA Stack

| Module | H100 | GB200 |
|--------|------|-------|
| nvidia | ✓ | ✓ |
| nvidia_uvm | ✓ | ✓ |
| nvidia_modeset | ✓ | ✓ |
| gdrdrv | ✓ | ✓ |
| ecc | ✓ | ✓ |

**Assessment:** No gap. GPU driver and GDR plumbing are aligned.

### Networking & RDMA

Both snapshots include:
- efa
- ib_core, ib_uverbs
- rdma_cm, iw_cm
- rpcrdma, sunrpc

**Assessment:** No meaningful difference. RDMA and EFA parity is good.

### Filesystem / Storage Stack

- **H100:** Includes full Lustre client stack (lustre, lmv, mdc, osc, ptlrpc, etc.)
- **GB200:** Lustre modules not present

**Classification:** True functional gap. H100 nodes are Lustre-capable; GB200 nodes are not configured with Lustre support.

⸻

## 5. Kubernetes Presence

| Aspect | H100 | GB200 |
|--------|------|-------|
| Kubernetes metadata | Present | Present |
| Reported version | v1.30.14-eks-3025e55 | v1.33.5-eks-3025e55 |
| Build date | 2025-11-11T03:22:24Z | 2025-11-11T03:21:21Z |
| Go version | go1.24.9 | go1.24.6 |
| Platform | linux/amd64 | linux/amd64 |

**Classification:** Both nodes have Kubernetes installed, but GB200 is running a newer version (v1.33.5 vs v1.30.14). Both are EKS builds from similar timeframes with minor Go version differences.

⸻

## 6. systemd: containerd

### containerd.service

- Active and enabled on both nodes
- Identical drop-ins (999-tuning-tuning.conf)
- Same cgroup delegation, limits, restart policy

Observed differences are limited to runtime counters:
- CPUUsageNSec
- MemoryCurrent
- TasksCurrent

**Classification:** Runtime variance only. No configuration drift.

⸻

## 7. Docker & Kubelet Units

| Unit | H100 | GB200 |
|------|------|-------|
| docker.service | inactive / not-found | inactive / not-found |
| kubelet.service | inactive | inactive |

**Assessment:** No difference.


