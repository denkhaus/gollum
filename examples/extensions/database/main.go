// Package main is a template extension for Gollum
// This extension demonstrates how to register a custom service with the DI container
package main

import (
	"fmt"

	"github.com/samber/do/v2"
)

// injector is automatically injected by the extension system
var injector do.Injector

// DatabaseService is an example service interface
type DatabaseService interface {
	Query(query string, args ...any) ([]map[string]any, error)
	Exec(query string, args ...any) error
	Close() error
}

// myDatabaseService implements DatabaseService
type myDatabaseService struct {
	connectionString string
}

// NewDatabaseService creates a new database service
func NewDatabaseService(_ do.Injector) (DatabaseService, error) {
	return &myDatabaseService{
		connectionString: "postgres://localhost:5432/mydb",
	}, nil
}

func (s *myDatabaseService) Query(query string, args ...any) ([]map[string]any, error) {
	// Implement your query logic here
	//nolint:forbidigo // fmt.Printf is acceptable in example code
	fmt.Printf("Query: %s with args: %v\n", query, args)
	return nil, nil
}

func (s *myDatabaseService) Exec(query string, args ...any) error {
	// Implement your exec logic here
	//nolint:forbidigo // fmt.Printf is acceptable in example code
	fmt.Printf("Exec: %s with args: %v\n", query, args)
	return nil
}

func (s *myDatabaseService) Close() error {
	//nolint:forbidigo // fmt.Println is acceptable in example code
	fmt.Println("Closing database connection")
	return nil
}

// Init is required for all extensions
// It is called automatically when the extension is loaded
//
//nolint:unparam // error return is required by extension interface
func Init() error {
	//nolint:forbidigo // fmt.Println is acceptable in example code
	fmt.Println("Database extension: initializing...")

	// Register the service with the DI container
	do.Provide(injector, NewDatabaseService)

	//nolint:forbidigo // fmt.Println is acceptable in example code
	fmt.Println("Database extension: registered DatabaseService")
	return nil
}
