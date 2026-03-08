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
func NewDatabaseService(inj do.Injector) (DatabaseService, error) {
	return &myDatabaseService{
		connectionString: "postgres://localhost:5432/mydb",
	}, nil
}

func (s *myDatabaseService) Query(query string, args ...any) ([]map[string]any, error) {
	// Implement your query logic here
	fmt.Printf("Query: %s with args: %v\n", query, args)
	return nil, nil
}

func (s *myDatabaseService) Exec(query string, args ...any) error {
	// Implement your exec logic here
	fmt.Printf("Exec: %s with args: %v\n", query, args)
	return nil
}

func (s *myDatabaseService) Close() error {
	fmt.Println("Closing database connection")
	return nil
}

// Init is required for all extensions
// It is called automatically when the extension is loaded
func Init() error {
	fmt.Println("Database extension: initializing...")

	// Register the service with the DI container
	do.Provide(injector, NewDatabaseService)

	fmt.Println("Database extension: registered DatabaseService")
	return nil
}
