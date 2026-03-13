package shared

import (
	"fmt"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
)

// MetadataInjectorKey is the key used to store the DI injector in command metadata
const MetadataInjectorKey = "injector"

// SetInjector stores the DI injector in the CLI command metadata
func SetInjector(cmd *cli.Command, injector do.Injector) {
	if cmd.Metadata == nil {
		cmd.Metadata = make(map[string]any)
	}
	cmd.Metadata[MetadataInjectorKey] = injector
}

// GetInjector retrieves the DI injector from the CLI command metadata
func GetInjector(cmd *cli.Command) (do.Injector, error) {
	if cmd.Metadata == nil {
		return nil, fmt.Errorf("command metadata is nil")
	}

	injector, ok := cmd.Metadata[MetadataInjectorKey]
	if !ok {
		return nil, fmt.Errorf("injector not found in metadata (key: %s)", MetadataInjectorKey)
	}

	typedInjector, ok := injector.(do.Injector)
	if !ok {
		return nil, fmt.Errorf("invalid injector type in metadata")
	}

	return typedInjector, nil
}

// MustGetInjector retrieves the DI injector from the CLI command metadata.
// Panics if the injector is not found or has an invalid type.
func MustGetInjector(cmd *cli.Command) do.Injector {
	injector, err := GetInjector(cmd)
	if err != nil {
		panic(err)
	}
	return injector
}
