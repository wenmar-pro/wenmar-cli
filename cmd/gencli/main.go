// Package main implements gencli — a spec-driven CLI command generator.
//
// It reads the enriched OpenAPI spec + an overrides YAML file and emits
// Go source files with cobra command declarations, flag wiring, and
// thin handler closures that call the shared runners in cmd/runners.go.
//
// Usage:
//
//	go run ./cmd/gencli -spec <path-to-enriched.yaml> -overrides cmd/gen_overrides.yaml -out cmd/gen_
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	specPath := flag.String("spec", "", "path to enriched OpenAPI spec YAML")
	overridesPath := flag.String("overrides", "cmd/gen_overrides.yaml", "path to gen_overrides.yaml")
	outDir := flag.String("out", "cmd", "output directory for generated files")
	flag.Parse()

	if *specPath == "" {
		fmt.Fprintln(os.Stderr, "error: -spec is required")
		os.Exit(1)
	}

	spec, err := loadSpec(*specPath)
	if err != nil {
		fatal("loading spec: %v", err)
	}

	overrides, err := loadOverrides(*overridesPath)
	if err != nil {
		fatal("loading overrides: %v", err)
	}

	// Group operations by CLI resource.
	groups := groupOperations(spec, overrides)

	// Emit one file per resource group.
	for _, group := range groups {
		fileName := filepath.Join(*outDir, "gen_"+group.Resource+".go")
		code, err := emitGroup(group, spec, overrides)
		if err != nil {
			fatal("emitting %s: %v", group.Resource, err)
		}
		if err := os.WriteFile(fileName, []byte(code), 0644); err != nil {
			fatal("writing %s: %v", fileName, err)
		}
		fmt.Printf("  generated %s\n", fileName)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gencli: "+format+"\n", args...)
	os.Exit(1)
}

// Spec is a minimal OpenAPI spec representation for parsing.
type Spec struct {
	Paths  map[string]PathItem `yaml:"paths"`
}

type PathItem map[string]Operation // keyed by HTTP method (get, post, patch, delete, put)

type Operation struct {
	OperationID  string              `yaml:"operationId"`
	Summary      string              `yaml:"summary"`
	Tags         []string            `yaml:"tags"`
	Parameters   []Parameter         `yaml:"parameters"`
	RequestBody  *RequestBody        `yaml:"requestBody"`
	Responses    map[string]Response `yaml:"responses"`
	XPaginated   bool                `yaml:"x-paginated"`
}

type Parameter struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"` // "path", "query"
	Required bool   `yaml:"required"`
	Schema   Schema `yaml:"schema"`
}

type Schema struct {
	Type  string   `yaml:"type"`
	Ref   string   `yaml:"$ref"`
	Items *Schema  `yaml:"items"`
}

type RequestBody struct {
	Required bool            `yaml:"required"`
	Content  map[string]Media `yaml:"content"`
}

type Media struct {
	Schema Schema `yaml:"schema"`
}

type Response struct {
	Description string            `yaml:"description"`
	Content     map[string]Media `yaml:"content"`
}

func loadSpec(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// Overrides is the gen_overrides.yaml structure.
type Overrides struct {
	Commands      map[string]CommandOverride `yaml:"commands"`
	FlagOverrides map[string]map[string]FlagOverride `yaml:"flag_overrides"`
	Exclude       []string                   `yaml:"exclude"`
}

type CommandOverride struct {
	Resource string `yaml:"resource"`
	Command  string `yaml:"command"`
	Summary  string `yaml:"summary"`
	Method   string `yaml:"method"` // override SDK method name if it differs from operationId mapping
}

type FlagOverride struct {
	Flag     string `yaml:"flag"`
	Help     string `yaml:"help"`
	Required bool   `yaml:"required"`
}

func loadOverrides(path string) (*Overrides, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var o Overrides
	if err := yaml.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}