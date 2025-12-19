package serializers

import (
	"encoding/json"
	"fmt"
)

// StdoutSerializer is a serializer that outputs configuration data to stdout in JSON format.
type StdoutSerializer struct {
}

// Serialize outputs the given configuration data to stdout in JSON format.
// It implements the Serializer interface.
func (s *StdoutSerializer) Serialize(config any) error {
	j, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize to json: %w", err)
	}

	fmt.Println(string(j))
	return nil
}
