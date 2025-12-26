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

// Option is a functional option for configuring the Header
type Option func(*Header)

// WithMetadata adds a metadata key-value pair to the Header.
func WithMetadata(key, value string) Option {
	return func(h *Header) {
		if h.Metadata == nil {
			h.Metadata = make(map[string]string)
		}
		h.Metadata[key] = value
	}
}

// WithKind sets the Kind field of the Header.
func WithKind(kind string) Option {
	return func(h *Header) {
		h.Kind = kind
	}
}

// WithAPIVersion sets the APIVersion field of the Header.
func WithAPIVersion(version string) Option {
	return func(h *Header) {
		h.APIVersion = version
	}
}

// SetKind sets the Kind field of the Header.
func (h *Header) SetKind(kind string) {
	h.Kind = kind
}

// New creates a new Header with the provided options.
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

// Header contains metadata about a snapshot or recommendation.
type Header struct {
	// Kind is the type of the snapshot object.
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	// APIVersion is the API version of the snapshot object.
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`

	// Metadata contains key-value pairs with metadata about the snapshot.
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Set initializes the header fields with the provided kind and version.
// The APIVersion is constructed using the kind.
func (h *Header) Set(kind string) {
	h.Kind = kind
	h.APIVersion = fmt.Sprintf("%s.%s/%s", strings.ToLower(kind), ApiVersionDomain, ApiVersionV1)
	h.Metadata = make(map[string]string)
	h.Metadata["recommendation-timestamp"] = time.Now().UTC().Format(time.RFC3339)
}
