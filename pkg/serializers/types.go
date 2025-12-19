package serializers

// Serializer is an interface for serializing configuration data.
// Implementations of this interface can serialize data to various formats
// such as JSON, YAML, or plain text.
type Serializer interface {
	Serialize(config any) error
}
