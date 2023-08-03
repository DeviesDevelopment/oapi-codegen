package codegen

import (
	"errors"
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

var DefaultOptions = map[string]GenerateOptions{
	EchoServer: {
		Target:  EchoServer,
		Enabled: true,
		Package: "internal/api/server",
		Output:  "server.go",
	},
	ChiServer: {
		Target:  ChiServer,
		Enabled: true,
		Package: "internal/api/server",
		Output:  "server.go",
	},
	GinServer: {
		Target:  GinServer,
		Enabled: true,
		Package: "internal/api/server",
		Output:  "server.go",
	},
	GorillaServer: {
		Target:  GorillaServer,
		Enabled: true,
		Package: "internal/api/server",
		Output:  "server.go",
	},
	StrictServer: {
		Target:  StrictServer,
		Enabled: true,
		Package: "internal/api/server",
		Output:  "server.go",
	},
	Client: {
		Target:  Client,
		Enabled: true,
		Package: "pkg/api/client",
		Output:  "client.go",
	},
	Models: {
		Target:  Models,
		Enabled: true,
		Package: "pkg/api/models",
		Output:  "types.go",
	},
	EmbeddedSpec: {
		Target:  EmbeddedSpec,
		Enabled: true,
		Package: "internal/api/server",
		Output:  "openapi_spec.go",
	},
}

type AdditionalImport struct {
	Alias   string `yaml:"alias,omitempty"`
	Package string `yaml:"package"`
}

// Configuration defines code generation customizations
type Configuration struct {
	ModuleName        string               `yaml:"module"`
	PackageName       string               `yaml:"package"` // Package name to generate, for backward compatibility
	Generate          []*GenerateOptions   `yaml:"generate,omitempty"`
	Compatibility     CompatibilityOptions `yaml:"compatibility,omitempty"`
	OutputOptions     OutputOptions        `yaml:"output-options,omitempty"`
	ImportMapping     map[string]string    `yaml:"import-mapping,omitempty"` // ImportMapping specifies the golang package path for each external reference
	AdditionalImports []AdditionalImport   `yaml:"additional-imports,omitempty"`
}

// GenerateOptions specifies which supported output formats to generate.
type GenerateOptions struct {
	Target  string `yaml:"target,omitempty"`  // Output target
	Enabled bool   `yaml:"enabled,omitempty"` // Flag indicating if the target is enabled and should be generated
	Package string `yaml:"package,omitempty"` // Package name for the target
	Output  string `yaml:"output,omitempty"`  // File name for the output
}

type GenerateOutput struct {
	Target  *GenerateOptions
	Imports string
	Code    string
}

type CodeOutput struct {
	Output map[string]*GenerateOutput
}

type FileOutput struct {
	Path string
	Name string
	Code string
}

func (f *FileOutput) OutputPath(mkdir bool) string {
	s := strings.Split(f.Path, "/")
	p := filepath.Join(s...)

	if mkdir {
		os.MkdirAll(p, os.ModePerm)
	}
	return filepath.Join(p, f.Name)
}

type Iterator interface {
	HasNext() bool
	Next() *FileOutput
}

type Collection interface {
	Iterator() Iterator
}

type OutputCollection struct {
	Output []*FileOutput
}

type OutputIterator struct {
	Index  int
	Output []*FileOutput
}

func (c CodeOutput) Collection() *OutputCollection {
	merged := map[string]string{}
	output := make([]*FileOutput, 0)

	// Loop through all targets
	for _, o := range c.Output {
		// Already merged?
		if merged[o.Target.OutputPath(false)] != "" {
			continue
		}
		// Join imports and code
		s := strings.Join([]string{o.Imports, o.Code}, "\n")
		fo := &FileOutput{
			Path: o.Target.Package,
			Name: o.Target.Output,
			Code: s,
		}
		output = append(output, fo)

		// Now loop through targets
		for _, t := range c.Output {
			// Don't compare with self
			if o.Target == t.Target {
				continue
			}

			// Do they share package and output file name?
			if o.Target.OutputPath(false) == t.Target.OutputPath(false) {
				// Merge code into existing file
				s := strings.Join([]string{fo.Code, t.Code}, "\n")
				fo.Code = s
				merged[o.Target.OutputPath(false)] = "true"
			}
		}
	}
	return &OutputCollection{
		Output: output,
	}
}

func (o *OutputCollection) Iterator() Iterator {
	return &OutputIterator{
		Output: o.Output,
	}
}

func (o *OutputIterator) HasNext() bool {
	if o.Index < len(o.Output) {
		return true
	}
	return false
}

func (o *OutputIterator) Next() *FileOutput {
	if o.HasNext() {
		output := o.Output[o.Index]
		o.Index++
		return output
	}
	return nil
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

func TargetFromAlias(alias string) (string, error) {
	target := targetMappings[strings.ToLower(alias)]
	if target == "nil" {
		return "", errors.New("Invalid alias:" + alias)
	}
	return target, nil
}

func NewDefaultConfiguration() Configuration {
	return Configuration{
		Generate: []*GenerateOptions{
			{
				Target:  EchoServer,
				Enabled: true,
				Package: "internal/api/server",
				Output:  "server.go",
			},
			{
				Target:  Client,
				Enabled: true,
				Package: "pkg/api/client",
				Output:  "client.go",
			},
			{
				Target:  Models,
				Enabled: true,
				Package: "pkg/api/models",
				Output:  "types.go",
			},
			{
				Target:  EmbeddedSpec,
				Enabled: true,
				Package: "internal/api/server",
				Output:  "openapi_spec.go",
			},
		},
		OutputOptions: OutputOptions{
			SkipFmt:   false,
			SkipPrune: false,
		},
	}
}

func NewDefaultConfigurationWithPackage(pkg string) Configuration {
	return Configuration{
		Generate: []*GenerateOptions{
			{
				Target:  EchoServer,
				Enabled: true,
				Package: pkg,
				Output:  "server.go",
			},
			{
				Target:  Client,
				Enabled: true,
				Package: pkg,
				Output:  "client.go",
			},
			{
				Target:  Models,
				Enabled: true,
				Package: pkg,
				Output:  "types.go",
			},
			{
				Target:  EmbeddedSpec,
				Enabled: true,
				Package: pkg,
				Output:  "openapi_spec.go",
			},
		},
		OutputOptions: OutputOptions{
			SkipFmt:   false,
			SkipPrune: false,
		},
	}
}

func (c Configuration) IsTargetEnabled(target string) bool {
	for _, g := range c.Generate {
		if strings.EqualFold(target, g.Target) {
			return true
		}
	}
	return false
}

func (c Configuration) GetTarget(target string) *GenerateOptions {
	for _, g := range c.Generate {
		if strings.EqualFold(target, g.Target) {
			return g
		}
	}
	return nil
}

// UpdateDefaults sets reasonable default values for unset fields in Configuration
func (o Configuration) UpdateDefaults() Configuration {
	if reflect.ValueOf(o.Generate).IsZero() {
		o.Generate = NewDefaultConfiguration().Generate
	}
	return o
}

// Validate checks whether Configuration represent a valid configuration
func (o Configuration) Validate() error {
	if o.Generate == nil {
		return errors.New("package name must be specified")
	}

	// Only one server type should be specified at a time.
	nServers := 0
	if o.IsTargetEnabled(ChiServer) {
		nServers++
	}
	if o.IsTargetEnabled(EchoServer) {
		nServers++
	}
	if o.IsTargetEnabled(GinServer) {
		nServers++
	}
	if nServers > 1 {
		return errors.New("only one server type is supported at a time")
	}
	return nil
}

func (g GenerateOptions) GolangPackage() string {
	if strings.Contains(g.Package, "/") {
		s := strings.SplitAfter(g.Package, "/")
		return s[len(s)-1]
	}
	return g.Package
}

// Should be able to remove this...
func (g GenerateOptions) OutputPath(mkdir bool) string {
	s := strings.Split(g.Package, "/")
	p := filepath.Join(s...)

	if mkdir {
		os.MkdirAll(p, os.ModePerm)
	}
	return filepath.Join(p, g.Output)
}

func (c CodeOutput) AddOutput(output *GenerateOutput) {
	if c.Output == nil {
		c.Output = make(map[string]*GenerateOutput)
	}

	c.Output[output.Target.Target] = output
}

func (c CodeOutput) GetOutput(target string) string {
	code, ok := c.Output[target]
	if !ok {
		return ""
	}
	return code.GetOutput()
}

func (g GenerateOutput) GetOutput() string {
	outBytes, err := imports.Process(g.Target.Output+".go", []byte(g.Imports+g.Code), nil)
	if err != nil {
		return ""
	}
	return string(outBytes)
}
