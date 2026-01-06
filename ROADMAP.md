# Cloud Native Stack Roadmap

> This roadmap reflects current thinking and direction. It is not a commitment to specific delivery dates.  
> Priorities may evolve based on validation results, partner feedback, and community input.

**Announcement Goal**  
The initial public announcement target for the next generation of Cloud Native Stack is **KubeCon + CloudNativeCon Europe (March 2026)**. Near-term and launch-blocking items are prioritized to support this milestone.

**Table of Contents**
- [High-Level Roadmap](#high-level-roadmap)
- [Scope & Principles](#scope--principles)
- [Near-Term (Next Release)](#near-term-next-release)
- [Launch Blockers](#launch-blockers)
- [Backlog](#backlog)
- [Revision History](#revision-history)

## High-Level Roadmap

```mermaid
flowchart LR
    A[Near-Term Foundations] --> B[Launch Blockers]
    B --> C[Core Platform Capabilities]
    C --> D[Bundlers & Artifacts]
    D --> E[Automation & Policy]
    E --> F[APIs & Integrations]
    F --> G[Extensibility & Advanced Features]

    A:::now
    B:::now
    C:::next
    D:::next
    E:::next
    F:::later
    G:::later

    classDef now fill:#e6f3ff,stroke:#1f77b4,stroke-width:2px;
    classDef next fill:#eef7ea,stroke:#2ca02c;
    classDef later fill:#f8f8f8,stroke:#999;
```

## Scope & Principles

**Project Scope**  
Cloud Native Stack generates validated configurations for GPU-accelerated Kubernetes deployments.  
Configurations are tested against NVIDIA GPU platforms (H100, GB200, A100, and others) and are intended to support both managed Kubernetes offerings (Amazon EKS, Google GKE, Azure AKS, Oracle OKE, etc.) and self-managed clusters.

The project focuses on:
- Capturing known-good configurations
- Making validation assumptions explicit
- Enabling reproducible deployment artifacts
- Integrating with existing infrastructure automation

It does not aim to replace Kubernetes distributions, provisioning systems, or managed services.

## Near-Term (Next Release)

### P0. Validate Current Bundlers

**Scope**  
Validate existing bundlers against known-good deployments to ensure they reliably reproduce validated configurations.

**Current Bundlers**
- GPU Operator
- Network Operator
- Skyhook

**Acceptance Criteria**
- [ ] All existing bundlers generate valid deployments from recipe measurements
- [ ] Generated artifacts reproduce known-good configurations
- [ ] Documentation clearly describes deployment steps and configuration options

### P1. Schema Validation

**User Story**  
As a CI/CD pipeline developer, I want to validate snapshots against versioned schemas so malformed data is caught before downstream processing.

**Acceptance Criteria**
- [ ] `eidos validate --schema v1 snapshot.yaml` command
- [ ] JSON Schema embedded via `go:embed`
- [ ] Validation using `github.com/santhosh-tekuri/jsonschema/v5`
- [ ] Clear validation errors with paths and line numbers
- [ ] Exit codes: `0 = valid`, `1 = invalid`
- [ ] Example CI/CD integrations

## Launch Blockers

### Differential Snapshots & Drift Detection

**User Story**  
As a compliance or platform operator, I want to detect configuration drift between clusters to ensure consistency and catch unauthorized changes.

**Acceptance Criteria**
- [ ] `eidos diff baseline.yaml current.yaml` command
- [ ] Output formats:
  - [ ] Human-readable
  - [ ] JSON Patch (RFC 6902)
  - [ ] Tabular output
- [ ] Highlight critical differences:
  - [ ] GPU driver versions
  - [ ] Kubernetes versions
  - [ ] Security-relevant settings
- [ ] Exit codes:
  - [ ] `0 = identical`
  - [ ] `1 = drift detected`
- [ ] CI/CD integration examples
- [ ] Ignore rules for expected differences (timestamps, unique IDs)

### Caching Layer

**User Story**  
As a CI/CD pipeline operator, I want snapshot caching to avoid redundant collection and speed up repeated CLI invocations.

**Acceptance Criteria**
- [ ] In-memory cache with configurable TTL (default: 5 minutes)
- [ ] `--cache-ttl` flag
- [ ] `--no-cache` flag to force fresh collection
- [ ] Cache key derived from collection parameters
- [ ] Automatic cache invalidation on TTL expiry
- [ ] Measurable performance improvement for repeated calls

## Backlog
### Core Platform Capabilities

#### Measurement Filtering

**User Story**  
As a developer, I want to capture only specific measurement types to reduce snapshot size and collection time.

**Acceptance Criteria**
- [ ] `eidos snapshot --filter gpu,os`
- [ ] `eidos snapshot --exclude k8s`
- [ ] Validation errors on unknown measurement types
- [ ] Excluded collectors are skipped entirely
- [ ] Documentation of common filter patterns

#### Configuration Files

**User Story**  
As a power user, I want persistent configuration so I do not need to repeat common flags on every command.

**Acceptance Criteria**
- [ ] Config file at `~/.config/eidos/config.yaml` (XDG-compliant)
- [ ] Defaults for common flags
- [ ] API server URL configuration
- [ ] CLI flags override config file values
- [ ] `eidos config init` command
- [ ] Validation on config load

#### Watch Mode

**User Story**  
As a monitoring system, I want continuous snapshot monitoring with change detection.

**Acceptance Criteria**
- [ ] `eidos snapshot --watch --interval 30s --on-change ./alert.sh`
- [ ] Stream JSON diffs to stdout
- [ ] Configurable polling interval
- [ ] Hook execution on change
- [ ] Graceful shutdown via signal handling

#### Platform-Specific Optimizations in Recipes

**User Story**  
As a GB200 or H100 operator, I want recipes to include platform-specific optimizations by default.

**Problem**  
Current recipes are largely generic and do not encode hardware-specific tuning.

**Approach**
- Introduce overlay-based recipe extensions
- Encode OS-level, kernel, and runtime tuning

**Acceptance Criteria**
- [ ] GB200 overlays (boot parameters, NUMA, IOMMU)
- [ ] H100-specific optimizations
- [ ] OS-level tuning via grub and sysctl
- [ ] Automatic application via generated bundles
- [ ] Documentation describing rationale and trade-offs

**Reference**
- `docs/v1/optimizations/GB200-NVL72.md`

#### Bundlers & Deployment Artifacts

#### Additional Bundlers

Planned bundlers include:

- **NIM Operator Bundler**  
  Support deployment and configuration of NVIDIA Inference Microservices.

- **Nsight Operator Bundler**  
  Enable cluster-wide profiling and observability tooling.

- **KServe Bundler**  
  Provide validated inference serving configurations.

- **Storage Bundler**  
  Encode best-practice storage configurations for GPU workloads.

- **Monitoring Bundler**  
  Integrate metrics, logging, and alerting components.

#### Multi-Bundler Orchestration

Support composing multiple bundlers into a single deployment plan, including:
- Dependency ordering
- Shared configuration inputs
- Coordinated rendering of artifacts

#### Bundle Distribution & Packaging

Explore mechanisms for:
- Versioned bundle packaging
- Distribution via OCI or artifact repositories
- Integrity and provenance metadata

### Automation, CI/CD & Policy

#### PVC-Based Agent Output

Enable Kubernetes agents to write snapshot output to PVCs for:
- Large snapshots
- Air-gapped environments
- Integration with offline pipelines

#### Policy Enforcement

Introduce policy evaluation for:
- Snapshot validation
- Recipe selection
- Bundle generation

Policies may gate deployments in CI/CD environments.

#### Cloud Storage Integration

Support snapshot upload to cloud storage backends for:
- Centralized analysis
- Historical comparison
- Cross-cluster auditing

### APIs & Interfaces

#### gRPC API Mode

Expose core CNS functionality via gRPC for programmatic integration.

#### GraphQL API

Enable flexible querying of configuration data and validation metadata.

#### Multi-Tenancy for API Server

Support isolation and access control for multiple users or organizations.

### Integrations (Cloud, Storage, Distribution)

#### Cloud Provider Integration

Align recipes and bundles with:
- AWS
- GCP
- Azure
- OCI

Including managed Kubernetes specifics and cloud-native primitives.

### Extensibility & Advanced Features

#### Plugin System

Allow external collectors, validators, or bundlers to be added without modifying core code.

#### Distributed Tracing

Add tracing to snapshot collection and bundle generation to improve observability and performance analysis.

### Design Opens & Architectural Questions

This section captures areas where design decisions are still open and community input is valuable.

### Architecture & Design
- Where should validation logic live?
- How tightly coupled should recipes and bundlers be?

### Storage & Distribution
- Best formats for long-term snapshot storage
- Artifact distribution models

### Bundle Generation & Orchestration
- Ordering guarantees
- Partial bundle updates

### Validation & Policy
- Policy languages and enforcement points

### Data Collection & Observability
- Telemetry volume vs signal
- Sampling strategies

### Performance & Scalability
- Large cluster behavior
- Snapshot size growth

### Developer Experience & Extensibility
- Plugin APIs
- Testing patterns

### Migration & Compatibility
- Backward compatibility guarantees
- Transition from earlier CNS versions

## Revision History
- **2026-01-01**: Initial comprehensive roadmap based on project objectives and gap analysis 
- **2026-01-05**: Added [Opens section](#opens) based on architectural decisions, implementation questions
- **2026-01-06**: Updated structure
