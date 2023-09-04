package codegen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/tools/imports"
)

const (
	ChiServer     = "chi-server"
	Client        = "client"
	EchoServer    = "echo-server"
	FiberServer   = "fiber-server"
	GinServer     = "gin-server"
	GorillaServer = "gorilla-server"
	Models        = "models"
	StrictServer  = "strict-server"
	EmbeddedSpec  = "embedded-spec"
)

var targetMappings = map[string]string{
	"chi":            ChiServer,
	"chi-server":     ChiServer,
	"client":         Client,
	"server":         EchoServer,
	"echo-server":    EchoServer,
	"echo":           EchoServer,
	"fiber":          FiberServer,
	"fiber-server":   FiberServer,
	"gin":            GinServer,
	"gin-server":     GinServer,
	"gorilla":        GorillaServer,
	"gorilla-server": GorillaServer,
	"strict":         StrictServer,
	"strict-server":  StrictServer,
	"types":          Models,
	"models":         Models,
	"spec":           EmbeddedSpec,
	"embedded-spec":  EmbeddedSpec,
}

var GenerateTargets = map[string]*GenerateTarget{
	EchoServer: {
		Target: EchoServer,
	},
	ChiServer: {
		Target: ChiServer,
	},
	FiberServer: {
		Target: FiberServer,
	},
	GinServer: {
		Target: GinServer,
	},
	GorillaServer: {
		Target: GorillaServer,
	},
	StrictServer: {
		Target: StrictServer,
	},
	Client: {
		Target: Client,
	},
	Models: {
		Target: Models,
	},
	EmbeddedSpec: {
		Target: EmbeddedSpec,
	},
}

type AdditionalImport struct {
	Alias   string `yaml:"alias,omitempty"`
	Package string `yaml:"package"`
}

// Configuration defines code generation customizations
type Configuration struct {
	ModuleName        string               `yaml:"module"`  // Module name for generated code when code is split into different packages
	PackageName       string               `yaml:"package"` // Package names to generate, for backward compatibility
	Generate          GenerateOptions      `yaml:"generate,omitempty"`
	Compatibility     CompatibilityOptions `yaml:"compatibility,omitempty"`
	OutputOptions     OutputOptions        `yaml:"output-options,omitempty"`
	ImportMapping     map[string]string    `yaml:"import-mapping,omitempty"` // ImportMapping specifies the golang package path for each external reference
	AdditionalImports []AdditionalImport   `yaml:"additional-imports,omitempty"`
	OutputFile        string               `yaml:"output,omitempty"`
	Targets           CodegenTargets       `yaml:"-"`
}

type CodegenTargets map[string]*GenerateTarget

type GenerateTarget struct {
	Target   string // Target name
	Package  string // Target package (including path)
	FileName string // Target filename
	Imports  string // Target imports
	Code     string // Target generated code
}

// GenerateOptions specifies which supported output formats to generate.
type GenerateOptions struct {
	ChiServer     bool `yaml:"chi-server,omitempty"`     // ChiServer specifies whether to generate chi server boilerplate
	FiberServer   bool `yaml:"fiber-server,omitempty"`   // FiberServer specifies whether to generate fiber server boilerplate
	EchoServer    bool `yaml:"echo-server,omitempty"`    // EchoServer specifies whether to generate echo server boilerplate
	GinServer     bool `yaml:"gin-server,omitempty"`     // GinServer specifies whether to generate gin server boilerplate
	GorillaServer bool `yaml:"gorilla-server,omitempty"` // GorillaServer specifies whether to generate Gorilla server boilerplate
	Strict        bool `yaml:"strict-server,omitempty"`  // Strict specifies whether to generate strict server wrapper
	Client        bool `yaml:"client,omitempty"`         // Client specifies whether to generate client boilerplate
	Models        bool `yaml:"models,omitempty"`         // Models specifies whether to generate type definitions
	EmbeddedSpec  bool `yaml:"embedded-spec,omitempty"`  // Whether to embed the swagger spec in the generated code
}

// CompatibilityOptions specifies backward compatibility settings for the
// code generator.
type CompatibilityOptions struct {
	// In the past, we merged schemas for `allOf` by inlining each schema
	// within the schema list. This approach, though, is incorrect because
	// `allOf` merges at the schema definition level, not at the resulting model
	// level. So, new behavior merges OpenAPI specs but generates different code
	// than we have in the past. Set OldMergeSchemas to true for the old behavior.
	// Please see https://github.com/deepmap/oapi-codegen/issues/531
	OldMergeSchemas bool `yaml:"old-merge-schemas,omitempty"`
	// Enum values can generate conflicting typenames, so we've updated the
	// code for enum generation to avoid these conflicts, but it will result
	// in some enum types being renamed in existing code. Set OldEnumConflicts to true
	// to revert to old behavior. Please see:
	// Please see https://github.com/deepmap/oapi-codegen/issues/549
	OldEnumConflicts bool `yaml:"old-enum-conflicts,omitempty"`
	// It was a mistake to generate a go type definition for every $ref in
	// the OpenAPI schema. New behavior uses type aliases where possible, but
	// this can generate code which breaks existing builds. Set OldAliasing to true
	// for old behavior.
	// Please see https://github.com/deepmap/oapi-codegen/issues/549
	OldAliasing bool `yaml:"old-aliasing,omitempty"`
	// When an object contains no members, and only an additionalProperties specification,
	// it is flattened to a map. Set
	DisableFlattenAdditionalProperties bool `yaml:"disable-flatten-additional-properties,omitempty"`
	// When an object property is both required and readOnly the go model is generated
	// as a pointer. Set DisableRequiredReadOnlyAsPointer to true to mark them as non pointer.
	// Please see https://github.com/deepmap/oapi-codegen/issues/604
	DisableRequiredReadOnlyAsPointer bool `yaml:"disable-required-readonly-as-pointer,omitempty"`
	// When set to true, always prefix enum values with their type name instead of only
	// when typenames would be conflicting.
	AlwaysPrefixEnumValues bool `yaml:"always-prefix-enum-values,omitempty"`
	// Our generated code for Chi has historically inverted the order in which Chi middleware is
	// applied such that the last invoked middleware ends up executing first in the Chi chain
	// This resolves the behavior such that middlewares are chained in the order they are invoked.
	// Please see https://github.com/deepmap/oapi-codegen/issues/786
	ApplyChiMiddlewareFirstToLast bool `yaml:"apply-chi-middleware-first-to-last,omitempty"`
	// Our generated code for gorilla/mux has historically inverted the order in which gorilla/mux middleware is
	// applied such that the last invoked middleware ends up executing first in the middlewares chain
	// This resolves the behavior such that middlewares are chained in the order they are invoked.
	// Please see https://github.com/deepmap/oapi-codegen/issues/841
	ApplyGorillaMiddlewareFirstToLast bool `yaml:"apply-gorilla-middleware-first-to-last,omitempty"`
}

// OutputOptions are used to modify the output code in some way.
type OutputOptions struct {
	SkipFmt       bool              `yaml:"skip-fmt,omitempty"`       // Whether to skip go imports on the generated code
	SkipPrune     bool              `yaml:"skip-prune,omitempty"`     // Whether to skip pruning unused components on the generated code
	IncludeTags   []string          `yaml:"include-tags,omitempty"`   // Only include operations that have one of these tags. Ignored when empty.
	ExcludeTags   []string          `yaml:"exclude-tags,omitempty"`   // Exclude operations that have one of these tags. Ignored when empty.
	UserTemplates map[string]string `yaml:"user-templates,omitempty"` // Override built-in templates from user-provided files

	ExcludeSchemas      []string `yaml:"exclude-schemas,omitempty"`      // Exclude from generation schemas with given names. Ignored when empty.
	ResponseTypeSuffix  string   `yaml:"response-type-suffix,omitempty"` // The suffix used for responses types
	ClientTypeName      string   `yaml:"client-type-name,omitempty"`     // Override the default generated client type with the value
	InitialismOverrides bool     `yaml:"initialism-overrides,omitempty"` // Whether to use the initialism overrides
}

// Creates a new default configuration.
func NewDefaultConfiguration() Configuration {
	return Configuration{
		Generate: GenerateOptions{
			EchoServer:   true,
			Client:       true,
			Models:       true,
			EmbeddedSpec: true,
		},
		OutputOptions: OutputOptions{
			SkipFmt:   false,
			SkipPrune: false,
		},
		Targets: map[string]*GenerateTarget{
			EchoServer:   GenerateTargets[EchoServer],
			Client:       GenerateTargets[Client],
			Models:       GenerateTargets[Models],
			EmbeddedSpec: GenerateTargets[EmbeddedSpec],
		},
	}
}

// Creates a default configuration with a packge name.
func NewDefaultConfigurationWithPackage(pkg string) Configuration {
	configuration := NewDefaultConfiguration()
	configuration.PackageName = pkg

	for _, target := range configuration.Targets {
		target.Package = pkg
	}

	return configuration
}

// Checks if the specified target is is enabled, i.e. exists in the configuration.
func (c Configuration) IsTargetEnabled(target string) bool {
	if _, enabled := c.Targets[target]; enabled {
		return true
	}
	return false
}

// Creates a target from an alias. If the alias is unknown, returns an error.
func (c Configuration) TargetFromAlias(alias string) error {
	target := targetMappings[strings.ToLower(alias)]
	if target == "nil" {
		return fmt.Errorf("Invalid alias:%s" + alias)
	}
	c.Targets[target] = GenerateTargets[target]
	return nil
}

// Validate checks whether Configuration represent a valid configuration.
func (c Configuration) Validate() error {
	// Make sure we have at least one package name
	if c.PackageName == "" {
		return errors.New("package name must be specified")
	}
	// Validate the package name(s)
	if err := updateTargetPackageNames(&c); err != nil {
		return err
	}
	// If output file name was specified, validate
	if c.OutputFile != "" {
		if err := updateTargetOutputNames(&c); err != nil {
			return err
		}
	}
	return nil
}

// Returns the package name to use in the source code output of a target.
func (g GenerateTarget) GolangPackage() string {
	if strings.Contains(g.Package, "/") {
		s := strings.SplitAfter(g.Package, "/")
		return s[len(s)-1]
	}
	return g.Package
}

// Returns the complete filename path of a target, including it's directories.
// Depending on the 'mdir' input argument, also creates the directories if
// needed.
func (g GenerateTarget) OutputPath(mkdir bool) string {
	s := strings.Split(g.Package, "/")
	p := filepath.Join(s...)

	if mkdir {
		os.MkdirAll(p, os.ModePerm)
	}
	return filepath.Join(p, g.FileName)
}

// Concatenates the imports and the generated code and formats the code. Then returns the
// result. Primarily for use with unit tests where you don't want to convert the generated
// code into an output collection and iterate over it.
func (g GenerateTarget) GetOutput(format bool) string {
	s := strings.Join([]string{g.Imports, g.Code}, "\n")

	if format {
		outputBytes, err := imports.Process(g.FileName, []byte(s), nil)
		if err != nil {
			return ""
		}
		return string(outputBytes)
	}
	return s
}

// A valid target to package mapping should contain a single value, or 2 values separated 
// by the '=' sign.
func validateTargetMapping(cfg *Configuration, s string) ([]string, error) {
	c := strings.Split(s, "=")

	switch len(c) {
	case 1:
		return c, nil
	case 2:
		if _, exists := targetMappings[c[0]]; !exists {
			return nil, fmt.Errorf("Invalid target mapping:%s", c)
		}
		return c, nil
	default:
		return nil, fmt.Errorf("Invalid target mapping:%s", s)
	}
}

// Retrieves the value of a string field from a configuration by name.
func getStringConfigFieldByName(cfg *Configuration, name string) (reflect.Value, error) {
	// Get pointer to configuration struct
	ps := reflect.ValueOf(cfg)
	// Get value of pointer (struct)
	s := ps.Elem()
	// Get the field value
	s = s.FieldByName(name)
	if s.Kind() != reflect.String {
		return reflect.Zero(reflect.TypeOf("")), fmt.Errorf("Config field '%s' is not a string", name)
	}
	return s, nil
}

// Validate the target to mappings for the given element name of the configuration. Element name
// can be either "Package" or "OutputFile" and needs to be a string.
func validateTargetMappings(cfg *Configuration, fieldName string) (map[string]string, error) {
	// Create a map of target to package/output mappings
	mappings := map[string]string{}

	// Get the reflect field value from the configuration
	configField, err := getStringConfigFieldByName(cfg, fieldName)
	if err != nil {
		return nil, err
	}
	// Split the value into an array and lop thorugh the values
	vals := strings.Split(configField.String(), ",")
	for _, p := range vals {
		// Validate the mapping
		mapping, err := validateTargetMapping(cfg, p)
		if err != nil {
			return nil, err
		}
		// Check if it's a package name that should be used for multiple targets
		if len(mapping) == 1 {
			// Check if we already have a multiple target package name
			if _, exists := mappings[""]; exists {
				return nil, fmt.Errorf("A mapping without target was specified more than once: %s", mapping[0])
			}
			// No multiple target mapping exists
			mappings[""] = mapping[0]
			continue
		}
		// Make sure the target mapping is only specified once
		//target, _ := TargetFromAlias(mapping[0])
		target, _ := targetMappings[mapping[0]]
		if _, exists := mappings[target]; exists {
			return nil, fmt.Errorf("A target mapping already exists: %s", target)
		}
		// A valid target to package mapping
		mappings[target] = mapping[1]
	}
	return mappings, nil
}

// Updates target package names from configuration.
func updateTargetPackageNames(cfg *Configuration) error {
	// Validate the mappings
	mappings, err := validateTargetMappings(cfg, "PackageName")
	if err != nil {
		return err
	}
	// Mappings are valid. Update the target package names
	for _, t := range cfg.Targets {
		// Check if we need to update the specific target
		if m, exists := mappings[t.Target]; exists {
			t.Package = m
			continue
		}
		// Check if we have a common package to use for updating the target
		if m, exists := mappings[""]; exists {
			t.Package = m
		}
	}

	return nil
}

// Updates target package names from configuration.
func updateTargetOutputNames(cfg *Configuration) error {
	// Validate the mappings
	mappings, err := validateTargetMappings(cfg, "OutputFile")
	if err != nil {
		return err
	}
	// Mappings are valid. Update the target output names
	for _, t := range cfg.Targets {
		// Check if we need to update the specific target
		if m, exists := mappings[t.Target]; exists {
			t.FileName = m
			continue
		}
		// Check if we have a common output to use for updating the target
		if m, exists := mappings[""]; exists {
			t.FileName = m
		}
	}

	return nil
}
