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
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	specPath := flag.String("spec", "", "path to enriched OpenAPI spec YAML")
	overridesPath := flag.String("overrides", "cmd/gen_overrides.yaml", "path to gen_overrides.yaml")
	outDir := flag.String("out", "cmd", "output directory for generated files")
	buildTag := flag.String("build-tag", "", "optional //go:build constraint to emit (e.g. ignore for golden fixtures)")
	flag.Parse()

	if *specPath == "" {
		fmt.Fprintln(os.Stderr, "error: -spec is required")
		os.Exit(1)
	}

	if err := runGenerate(*specPath, *overridesPath, *outDir, *buildTag); err != nil {
		fatal("%v", err)
	}
}

// runGenerate loads the spec + overrides and emits one file per resource
// group into outDir. It is the shared entry point for the CLI and the
// golden regen-drift test. When buildTag is non-empty it is emitted as a
// //go:build constraint (used for golden fixtures so they don't compile).
func runGenerate(specPath, overridesPath, outDir, buildTag string) error {
	spec, err := loadSpec(specPath)
	if err != nil {
		return fmt.Errorf("loading spec: %w", err)
	}

	overrides, err := loadOverrides(overridesPath)
	if err != nil {
		return fmt.Errorf("loading overrides: %w", err)
	}

	// Group operations by CLI resource.
	groups := groupOperations(spec, overrides)

	// Emit one file per resource group.
	for _, group := range groups {
		fileName := filepath.Join(outDir, "gen_"+group.Resource+".go")
		code, err := emitGroup(group, spec, overrides, buildTag)
		if err != nil {
			return fmt.Errorf("emitting %s: %w", group.Resource, err)
		}
		if err := os.WriteFile(fileName, []byte(code), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", fileName, err)
		}
		fmt.Printf("  generated %s\n", fileName)
	}
	return nil
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gencli: "+format+"\n", args...)
	os.Exit(1)
}

// Spec is a minimal OpenAPI spec representation for parsing.
type Spec struct {
	Paths      map[string]PathItem `yaml:"paths"`
	Components Components          `yaml:"components"`
}

// Components holds the spec's reusable schema definitions.
type Components struct {
	Schemas map[string]Schema `yaml:"schemas"`
}

// Resolve follows a "#/components/schemas/<name>" $ref one level and
// returns the referenced schema. Unresolvable refs return the schema
// untouched so callers treat it as an opaque (non-object) schema.
func (s *Spec) Resolve(sc Schema) Schema {
	if sc.Ref == "" {
		return sc
	}
	const prefix = "#/components/schemas/"
	name := strings.TrimPrefix(sc.Ref, prefix)
	if target, ok := s.Components.Schemas[name]; ok {
		return target
	}
	return sc
}

type PathItem map[string]Operation // keyed by HTTP method (get, post, patch, delete, put)

type Operation struct {
	OperationID          string              `yaml:"operationId"`
	Summary              string              `yaml:"summary"`
	Tags                 []string            `yaml:"tags"`
	Parameters           []Parameter         `yaml:"parameters"`
	RequestBody          *RequestBody        `yaml:"requestBody"`
	Responses            map[string]Response `yaml:"responses"`
	XPaginated           bool                `yaml:"x-paginated"`
	XWenmarRequestSchema string              `yaml:"x-wenmar-request-schema"`
}

type Parameter struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"` // "path", "query"
	Required bool   `yaml:"required"`
	Schema   Schema `yaml:"schema"`
}

type Schema struct {
	Type       string            `yaml:"type"`
	Ref        string            `yaml:"$ref"`
	Items      *Schema           `yaml:"items"`
	Properties map[string]Schema `yaml:"properties"`
	Required   *[]string         `yaml:"required"`
}

type RequestBody struct {
	Required bool             `yaml:"required"`
	Content  map[string]Media `yaml:"content"`
}

type Media struct {
	Schema Schema `yaml:"schema"`
}

type Response struct {
	Description string           `yaml:"description"`
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
	Commands      map[string]CommandOverride         `yaml:"commands"`
	FlagOverrides map[string]map[string]FlagOverride `yaml:"flag_overrides"`
	Exclude       []string                           `yaml:"exclude"`
	Groups        map[string]GroupOverride           `yaml:"groups"`
}

// GroupOverride shapes a resource parent command.
type GroupOverride struct {
	Aliases []string `yaml:"aliases"`
	Short   string   `yaml:"short"`
}

type CommandOverride struct {
	Resource         string   `yaml:"resource"`
	Command          string   `yaml:"command"`
	Summary          string   `yaml:"summary"`
	ActionSummary    string   `yaml:"action_summary"`
	Method           string   `yaml:"method"`
	RequestStruct    string   `yaml:"request_struct"`
	PositionalArg    string   `yaml:"positional_arg"`
	QueryParamStruct string   `yaml:"query_param_struct"`
	Paginated        *bool    `yaml:"paginated"`
	Tab              string   `yaml:"tab"`
	Aliases          []string `yaml:"aliases"`
	Short            string   `yaml:"short"`
	IdParam          string   `yaml:"id_param"`
	Example          string   `yaml:"example"`
}

type FlagOverride struct {
	Flag     string `yaml:"flag"`
	Help     string `yaml:"help"`
	Required *bool  `yaml:"required"`
	Suppress bool   `yaml:"suppress"`
	Default  string `yaml:"default"`
	FlagType string `yaml:"flag_type"` // "intslice" for array params bound via IntSliceVar
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
