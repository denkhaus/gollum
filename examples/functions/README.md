# Func Step Examples

This directory contains example func steps that can be used in flows.

## Installation

1. Create directory: `~/.config/gollum/functions/`
2. Copy `.go` files to that directory
3. Restart Gollum or reload flows

## Usage in Flows

```xml
<step type="func" function="ToUpper">
    <params>
        <param name="input" value="${input.text}"/>
    </params>
    <output assign="${output.uppercase}"/>
</step>
```

## Available Functions

### String Functions

- **ToUpper(input string) string** - Convert to uppercase
- **ToLower(input string) string** - Convert to lowercase
- **Replace(input, old, new string) string** - Replace occurrences
- **Format(args map[string]any) string** - Format string with template

## Creating Custom Functions

1. Create a new `.go` file
2. Add exported functions with the signature: `func Name(params...) returnType`
3. File name becomes function name (e.g., `utils.go` -> `utils` is not a function, the functions inside are)
4. Functions are automatically loaded and available in flows

### Example

```go
package main

func Add(a, b int) int {
    return a + b
}
```

Usage in flow:
```xml
<step type="func" function="Add">
    <params>
        <param name="a" value="${input.x}"/>
        <param name="b" value="${input.y}"/>
    </params>
    <output assign="${output.sum}"/>
</step>
```
