package main

import (
	"fmt"

	"github.com/denkhaus/action-golang/magefiles/helpers"
	"github.com/magefile/mage/mg"
)

var (
	print      = helpers.NewPrinter()
	ColorBlue  = helpers.ColorBlue
	ColorGreen = helpers.ColorGreen
	cfg        = helpers.NewConfig()
	client     = helpers.NewDockerClient(cfg)
)

// Docker namespace
type Docker mg.Namespace

// Build builds the Docker image
func (Docker) Build() error {
	return client.Build(cfg.Image.FullName())
}

// Push builds and pushes the image to registry
func (Docker) Push() error {
	return client.Push(cfg.FullRegistryPath())
}

// Test tests the Docker image locally
func (Docker) Test() error {
	return client.TestImage(cfg.Image.FullName())
}

// Shell starts an interactive shell in the container
func (Docker) Shell() error {
	print.Info("Starting interactive shell in %s", cfg.Image.FullName())
	return client.RunInteractive(cfg.Image.FullName(), "/bin/bash")
}

// Clean cleans up dangling Docker images
func (Docker) Clean() error {
	return client.Clean()
}

// Release builds, tests, and pushes with a date-based tag
func (Docker) Release() error {
	releaseTag, err := client.BuildRelease()
	if err != nil {
		return err
	}

	// Test the image before pushing
	if err := client.TestImage(cfg.Image.FullName()); err != nil {
		return fmt.Errorf("test failed before push: %w", err)
	}

	releaseImage := cfg.Image.Name + ":" + releaseTag
	if err := client.Push(releaseImage); err != nil {
		return fmt.Errorf("push release tag failed: %w", err)
	}

	if err := client.Push(cfg.Image.FullName()); err != nil {
		return fmt.Errorf("push latest failed: %w", err)
	}

	print.Success("Release %s pushed successfully", releaseTag)
	return nil
}

// List lists all available docker targets
func (Docker) List() {
	print.Plain("")
	print.Printf(ColorBlue, "Docker Targets - Mage Help")
	print.Plain("")
	print.Printf(ColorGreen, "  mage docker:build    - Build the Docker image (default)")
	print.Printf(ColorGreen, "  mage docker:push     - Build and push to registry")
	print.Printf(ColorGreen, "  mage docker:test     - Test the image locally")
	print.Printf(ColorGreen, "  mage docker:shell    - Start interactive shell in container")
	print.Printf(ColorGreen, "  mage docker:clean    - Clean up dangling images")
	print.Printf(ColorGreen, "  mage docker:release  - Build, test, and push with date-based tag")
	print.Printf(ColorGreen, "  mage docker:list     - Show this help message")
	print.Plain("")
	print.Plain("Examples:")
	print.Plain("  mage docker:build")
	print.Plain("  mage docker:push")
	print.Plain("  mage docker:release  # Full CI/CD pipeline: build → test → push")
}
