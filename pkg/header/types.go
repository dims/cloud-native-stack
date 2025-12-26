package header

import (
	"fmt"
	"strings"
	"time"
)

var (
	ApiVersionDomain = "dgxc.io"
	ApiVersionV1     = "v1"
)

// Option is a functional option for configuring Header instances.
type Option func(*Header)

// WithMetadata returns an Option that adds a metadata key-value pair to the Header.
// If the Metadata map is nil, it will be initialized.
func WithMetadata(key, value string) Option {
	return func(h *Header) {
		if h.Metadata == nil {
			h.Metadata = make(map[string]string)
		}
		h.Metadata[key] = value
	}
}

// WithKind returns an Option that sets the Kind field of the Header.
// Kind represents the type of the resource (e.g., "Snapshot", "Recipe").
func WithKind(kind string) Option {
	return func(h *Header) {
		h.Kind = kind
	}
}

// WithAPIVersion returns an Option that sets the APIVersion field of the Header.
// The APIVersion identifies the schema version for the resource.
func WithAPIVersion(version string) Option {
	return func(h *Header) {
		h.APIVersion = version
	}
}

// SetKind updates the Kind field of the Header.
func (h *Header) SetKind(kind string) {
	h.Kind = kind
}

// New creates a new Header instance with the provided functional options.
// The Metadata map is initialized automatically.
func New(opts ...Option) *Header {
	s := &Header{
		Metadata: make(map[string]string),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Header contains metadata and versioning information for CNS resources.
// It follows Kubernetes-style resource conventions with Kind, APIVersion, and Metadata fields.
type Header struct {
	// Kind is the type of the snapshot object.
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	// APIVersion is the API version of the snapshot object.
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`

	// Metadata contains key-value pairs with metadata about the snapshot.
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Set initializes the Header fields with the provided kind.
// It automatically constructs the APIVersion using the format "<kind>.dgxc.io/v1"
// and adds a recommendation-timestamp to the Metadata.
func (h *Header) Set(kind string) {
	h.Kind = kind
	h.APIVersion = fmt.Sprintf("%s.%s/%s", strings.ToLower(kind), ApiVersionDomain, ApiVersionV1)
	h.Metadata = make(map[string]string)
	h.Metadata["recommendation-timestamp"] = time.Now().UTC().Format(time.RFC3339)
}
