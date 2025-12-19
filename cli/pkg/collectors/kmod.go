package collectors

import (
	"fmt"
	"os"
	"strings"
)

type KModCollector struct {
}

const KModType string = "KMod"

type KModConfig struct {
	Name string
}

func (s *KModCollector) Collect(config any) ([]Configuraion, error) {
	root := "/proc/modules"
	res := make([]Configuraion, 0)

	cmdline, err := os.ReadFile(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read KMod config: %w", err)
	}

	params := strings.Split(string(cmdline), "\n")

	for _, param := range params {
		p := strings.TrimSpace(param)
		if p == "" {
			continue
		}

		mod := strings.Split(p, " ")

		res = append(res, Configuraion{
			Type: KModType,
			Data: KModConfig{
				Name: mod[0],
			},
		})
	}

	return res, nil
}
