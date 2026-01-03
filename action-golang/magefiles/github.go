package main

import (
	"github.com/denkhaus/action-golang/magefiles/github"
	"github.com/denkhaus/action-golang/magefiles/helpers"

	"github.com/magefile/mage/mg"
)

// GitHub namespace
type GitHub mg.Namespace

// ExtractPrompt extracts the prompt from GitHub event and writes to file
func (GitHub) ExtractPrompt() error {
	return github.ExtractPrompt()
}

// CreatePR creates a pull request from current changes
func (GitHub) CreatePR() error {
	return github.CreatePR()
}

// List lists all available github targets
func (GitHub) List() {
	github.List()
	print := helpers.NewPrinter()
	print.Printf(helpers.ColorBlue, "\nUsage:")
	print.Printf(helpers.ColorGreen, "  mage github:extractprompt")
	print.Printf(helpers.ColorGreen, "  mage github:createpr")
	print.Printf(helpers.ColorGreen, "  mage github:list")
}
