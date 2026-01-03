package helpers

import "time"

// Config holds all configuration for the magefile
type Config struct {
	Image     ImageConfig
	Registry  string
	BuildArgs BuildArgs
}

// ImageConfig holds Docker image configuration
type ImageConfig struct {
	Name string
	Tag  string
}

// FullName returns the full image name with tag
func (i ImageConfig) FullName() string {
	return i.Name + ":" + i.Tag
}

// BuildArgs holds build arguments for Docker
type BuildArgs struct {
	GolangciLintVersion string
	GHVersion           string
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Image: ImageConfig{
			Name: "denkhaus/golang-claude-action",
			Tag:  "latest",
		},
		Registry: "docker.io",
		BuildArgs: BuildArgs{
			GolangciLintVersion: "v2.6.1",
			GHVersion:           "v2.60.0",
		},
	}
}

// FullRegistryPath returns the full registry path including image
func (c *Config) FullRegistryPath() string {
	return c.Registry + "/" + c.Image.FullName()
}

// ReleaseTag returns a date-based tag for releases
func (c *Config) ReleaseTag() string {
	return time.Now().Format("20060102")
}

// ReleaseImage returns the image name with release tag
func (c *Config) ReleaseImage() string {
	return c.Image.Name + ":" + c.ReleaseTag()
}
