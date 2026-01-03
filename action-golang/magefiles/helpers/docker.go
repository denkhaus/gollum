package helpers

import (
	"fmt"
	"strings"

	"github.com/magefile/mage/sh"
)

// DockerClient handles Docker operations
type DockerClient struct {
	config *Config
	print  *Printer
}

// NewDockerClient creates a new Docker client
func NewDockerClient(config *Config) *DockerClient {
	return &DockerClient{
		config: config,
		print:  print,
	}
}

// Build builds a Docker image with the given tag
func (d *DockerClient) Build(tag string) error {
	d.print.Info("Building Docker image: %s", tag)

	args := []string{
		"build",
		"--build-arg", "GOLANGCI_LINT_VERSION=" + d.config.BuildArgs.GolangciLintVersion,
		"--build-arg", "GH_VERSION=" + d.config.BuildArgs.GHVersion,
		"-t", tag,
		"-f", "Dockerfile",
		".",
	}

	if err := sh.RunV("docker", args...); err != nil {
		d.print.Error("Build failed: %v", err)
		return fmt.Errorf("docker build failed: %w", err)
	}

	d.print.Success("Build complete: %s", tag)
	return nil
}

// Push pushes a Docker image to registry
func (d *DockerClient) Push(imageRef string) error {
	d.print.Info("Pushing image to registry: %s", imageRef)

	if err := sh.RunV("docker", "push", imageRef); err != nil {
		d.print.Error("Push failed: %v", err)
		return fmt.Errorf("docker push failed: %w", err)
	}

	d.print.Success("Push complete: %s", imageRef)
	return nil
}

// Run executes a command in a Docker container
func (d *DockerClient) Run(image string, args ...string) error {
	fullArgs := append([]string{"run", "--rm", image}, args...)
	return sh.Run("docker", fullArgs...)
}

// RunInteractive runs an interactive container
func (d *DockerClient) RunInteractive(image string, args ...string) error {
	fullArgs := append([]string{"run", "--rm", "-it", image}, args...)
	return sh.Run("docker", fullArgs...)
}

// Clean removes dangling Docker images
func (d *DockerClient) Clean() error {
	d.print.Info("Cleaning up dangling Docker images...")

	if err := sh.Run("docker", "image", "prune", "-f"); err != nil {
		d.print.Error("Cleanup failed: %v", err)
		return fmt.Errorf("docker prune failed: %w", err)
	}

	d.print.Success("Cleanup complete")
	return nil
}

// Login logs in to a Docker registry
func (d *DockerClient) Login(registry string) error {
	d.print.Warning("Logging in to %s...", registry)
	d.print.Plain("Hint: Use 'docker login' manually or set up CI credentials")
	return sh.Run("docker", "login", registry)
}

// TestImage tests a Docker image by running verification commands
func (d *DockerClient) TestImage(image string) error {
	d.print.Info("Testing Docker image: %s", image)

	tests := []struct {
		directory string
		name      string
		args      []string
	}{
		{".", "go version", []string{"go", "version"}},
		{".", "gh version", []string{"gh", "--version"}},
		{".", "golangci-lint version", []string{"golangci-lint", "--version"}},
		{".", "mockgen version", []string{"mockgen", "-version"}},
		{".", "bun version", []string{"bun", "--version"}},
		{"/opt", "mage version", []string{"mage", "--version"}},
		{"/opt", "mage docker:list", []string{"mage", "docker:list"}},
	}

	for _, test := range tests {
		d.print.Warning("Running %s...", test.name)
		cmd := fmt.Sprintf("cd %s && %s", test.directory, strings.Join(test.args, " "))

		if err := d.Run(image, "sh", "-c", cmd); err != nil {
			d.print.Error("Test '%s' failed: %v", test.name, err)
			return fmt.Errorf("test %s failed: %w", test.name, err)
		}
	}

	d.print.Success("All tests passed!")
	return nil
}

// BuildRelease builds a release image with both date tag and 'latest'
func (d *DockerClient) BuildRelease() (string, error) {
	releaseTag := d.config.ReleaseTag()
	releaseImage := d.config.ReleaseImage()
	latestImage := d.config.Image.FullName()

	d.print.Info("Building release: %s (also tagging as %s)", releaseImage, latestImage)

	args := []string{
		"build",
		"--build-arg", "GOLANGCI_LINT_VERSION=" + d.config.BuildArgs.GolangciLintVersion,
		"--build-arg", "GH_VERSION=" + d.config.BuildArgs.GHVersion,
		"-t", releaseImage,
		"-t", latestImage,
		"-f", "Dockerfile",
		".",
	}

	if err := sh.RunV("docker", args...); err != nil {
		return "", fmt.Errorf("docker build failed: %w", err)
	}

	d.print.Success("Release image built: %s", releaseImage)
	return releaseTag, nil
}

// ImageExists checks if an image exists locally
func (d *DockerClient) ImageExists(image string) bool {
	output, err := sh.Output("docker", "images", "-q", image)
	return err == nil && len(strings.TrimSpace(output)) > 0
}
