package codegen

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/golangci/lint-1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	examplePetstoreClient "github.com/deepmap/oapi-codegen/examples/petstore-expanded"
	examplePetstore "github.com/deepmap/oapi-codegen/examples/petstore-expanded/echo/api"
	"github.com/deepmap/oapi-codegen/pkg/util"
)

const (
	remoteRefFile = `https://raw.githubusercontent.com/deepmap/oapi-codegen/master/examples/petstore-expanded` +
		`/petstore-expanded.yaml`
	remoteRefImport = `github.com/deepmap/oapi-codegen/examples/petstore-expanded`
)

func checkLint(t *testing.T, filename string, code []byte) {
	linter := new(lint.Linter)
	problems, err := linter.Lint(filename, code)
	assert.NoError(t, err)
	assert.Len(t, problems, 0)
}

func TestExamplePetStoreCodeGeneration(t *testing.T) {

	// Input vars for code generation:
	opts := NewDefaultConfigurationWithPackage("petstore")

	// Get a spec from the example PetStore definition:
	swagger, err := examplePetstore.GetSwagger()
	assert.NoError(t, err)

	// Run our code generation:
	code, err := Generate(swagger, opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, code)

	// Loop through the code
	for _, c := range code.Output {
		out := c.GetOutput()
		// Check that we have valid (formattable) code:
		_, err = format.Source([]byte(out))
		assert.NoError(t, err)

		// Check that we have a package:
		assert.Contains(t, out, "package petstore")

		if c.Target.Target == Client {
			// Check that the client method signatures return response structs:
			assert.Contains(t, out, "func (c *Client) FindPetByID(ctx context.Context, id int64, reqEditors ...RequestEditorFn) (*http.Response, error) {")
		}

		if c.Target.Target == Models {
			// Check that the property comments were generated
			assert.Contains(t, out, "// Id Unique id of the pet")
		}

		if strings.Contains(c.Target.Target, "server") {
			// Check that the summary comment contains newlines
			assert.Contains(t, out, "// Deletes a pet by ID")
		}

		// Make sure the generated code is valid:
		checkLint(t, "test.gen.go", []byte(out))
	}
}

func TestExamplePetStoreCodeGenerationWithUserTemplates(t *testing.T) {

	userTemplates := map[string]string{"typedef.tmpl": "//blah\n//blah"}

	// Input vars for code generation:
	packageName := "api"
	opts := Configuration{
		Generate: []*GenerateOptions{
			{
				Target:  Models,
				Enabled: true,
				Package: packageName,
				Output:  "types.go",
			},
		},
		OutputOptions: OutputOptions{
			UserTemplates: userTemplates,
		},
	}

	// Get a spec from the example PetStore definition:
	swagger, err := examplePetstore.GetSwagger()
	assert.NoError(t, err)

	// Run our code generation:
	code, err := Generate(swagger, opts)
	out := code.GetOutput(Models)
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// Check that we have valid (formattable) code:
	_, err = format.Source([]byte(out))
	assert.NoError(t, err)

	// Check that we have a package:
	assert.Contains(t, out, "package api")

	// Check that the built-in template has been overridden
	assert.Contains(t, out, "//blah")
}

func TestExamplePetStoreCodeGenerationWithFileUserTemplates(t *testing.T) {

	userTemplates := map[string]string{"typedef.tmpl": "./templates/typedef.tmpl"}

	// Input vars for code generation:
	packageName := "api"
	opts := Configuration{
		Generate: []*GenerateOptions{
			{
				Target:  Models,
				Enabled: true,
				Package: packageName,
				Output:  "types.go",
			},
		},
		OutputOptions: OutputOptions{
			UserTemplates: userTemplates,
		},
	}

	// Get a spec from the example PetStore definition:
	swagger, err := examplePetstore.GetSwagger()
	assert.NoError(t, err)

	// Run our code generation:
	code, err := Generate(swagger, opts)
	out := code.GetOutput(Models)
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// Check that we have valid (formattable) code:
	_, err = format.Source([]byte(out))
	assert.NoError(t, err)

	// Check that we have a package:
	assert.Contains(t, out, "package api")
	// Check that the built-in template has been overridden
	assert.Contains(t, out, "// Package api provides primitives to interact with the openapi")
}

func TestExamplePetStoreCodeGenerationWithHTTPUserTemplates(t *testing.T) {

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer ln.Close()

	//nolint:errcheck
	// Does not matter if the server returns an error on close etc.
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, writeErr := w.Write([]byte("//blah"))
		assert.NoError(t, writeErr)
	}))

	t.Logf("Listening on %s", ln.Addr().String())

	userTemplates := map[string]string{"typedef.tmpl": fmt.Sprintf("http://%s", ln.Addr().String())}

	// Input vars for code generation:
	packageName := "api"
	opts := Configuration{
		PackageName: packageName,
		Generate: []*GenerateOptions{
			{
				Target:  Models,
				Enabled: true,
				Package: packageName,
				Output:  "types.go",
			},
		},
		OutputOptions: OutputOptions{
			UserTemplates: userTemplates,
		},
	}

	// Get a spec from the example PetStore definition:
	swagger, err := examplePetstore.GetSwagger()
	assert.NoError(t, err)

	// Run our code generation:
	code, err := Generate(swagger, opts)
	out := code.GetOutput(Models)
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// Check that we have valid (formattable) code:
	_, err = format.Source([]byte(out))
	assert.NoError(t, err)

	// Check that we have a package:
	assert.Contains(t, out, "package api")

	// Check that the built-in template has been overriden
	assert.Contains(t, out, "//blah")
}

func TestExamplePetStoreParseFunction(t *testing.T) {

	bodyBytes := []byte(`{"id": 5, "name": "testpet", "tag": "cat"}`)

	cannedResponse := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
		Header:     http.Header{},
	}
	cannedResponse.Header.Add("Content-type", "application/json")

	findPetByIDResponse, err := examplePetstoreClient.ParseFindPetByIDResponse(cannedResponse)
	assert.NoError(t, err)
	assert.NotNil(t, findPetByIDResponse.JSON200)
	assert.Equal(t, int64(5), findPetByIDResponse.JSON200.Id)
	assert.Equal(t, "testpet", findPetByIDResponse.JSON200.Name)
	assert.NotNil(t, findPetByIDResponse.JSON200.Tag)
	assert.Equal(t, "cat", *findPetByIDResponse.JSON200.Tag)
}

func TestExampleOpenAPICodeGeneration(t *testing.T) {

	// Input vars for code generation:
	opts := NewDefaultConfigurationWithPackage("testswagger")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	// Get a spec from the test definition in this file:
	swagger, err := loader.LoadFromData([]byte(testOpenAPIDefinition))
	assert.NoError(t, err)

	// Run our code generation:
	code, err := Generate(swagger, opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, code.Output)

	for _, c := range code.Output {
		out := c.GetOutput()
		// Check that we have valid (formattable) code:
		_, err := format.Source([]byte(out))
		assert.NoError(t, err)

		// Check that we have a package:
		assert.Contains(t, out, "package testswagger")

		if c.Target.Target == Client {
			// Check that response structs are generated correctly:
			assert.Contains(t, out, "type GetTestByNameResponse struct {")

			// Check that response structs contains fallbacks to interface for invalid types:
			// Here an invalid array with no items.
			assert.Contains(t, out, `
type GetTestByNameResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *[]Test
	XML200       *[]Test
	JSON422      *[]interface{}
	XML422       *[]interface{}
	JSONDefault  *Error
}`)

			// Check that the helper methods are generated correctly:
			assert.Contains(t, out, "func (r GetTestByNameResponse) Status() string {")
			assert.Contains(t, out, "func ParseGetTestByNameResponse(rsp *http.Response) (*GetTestByNameResponse, error) {")
			assert.Contains(t, out, "func (c *Client) GetTestByName(ctx context.Context, name string, params *GetTestByNameParams, reqEditors ...RequestEditorFn) (*http.Response, error) {")
			assert.Contains(t, out, "func (c *ClientWithResponses) GetTestByNameWithResponse(ctx context.Context, name string, params *GetTestByNameParams, reqEditors ...RequestEditorFn) (*GetTestByNameResponse, error) {")
		}

		if c.Target.Target == Models {
			// Check the client method signatures:
			assert.Contains(t, out, "type GetTestByNameParams struct {")
			assert.Contains(t, out, "Top *int `form:\"$top,omitempty\" json:\"$top,omitempty\"`")
			assert.Contains(t, out, "DeadSince *time.Time    `json:\"dead_since,omitempty\" tag1:\"value1\" tag2:\"value2\"`")
			assert.Contains(t, out, "type EnumTestNumerics int")
			assert.Contains(t, out, "N2 EnumTestNumerics = 2")
			assert.Contains(t, out, "type EnumTestEnumNames int")
			assert.Contains(t, out, "Two  EnumTestEnumNames = 2")
			assert.Contains(t, out, "Double EnumTestEnumVarnames = 2")
		}

		// Make sure the generated code is valid:
		checkLint(t, "test.gen.go", []byte(out))
	}
}

func TestExtPropGoTypeSkipOptionalPointer(t *testing.T) {
	opts := NewDefaultConfigurationWithPackage("api")
	spec := "test_specs/x-go-type-skip-optional-pointer.yaml"
	swagger, err := util.LoadSwagger(spec)
	require.NoError(t, err)

	// Run our code generation:
	code, err := Generate(swagger, opts)
	out := code.GetOutput(Client)
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// Check that we have valid (formattable) code:
	_, err = format.Source([]byte(out))
	assert.NoError(t, err)

	// Check that optional pointer fields are skipped if requested
	assert.Contains(t, out, "NullableFieldSkipFalse *string `json:\"nullableFieldSkipFalse\"`")
	assert.Contains(t, out, "NullableFieldSkipTrue  string  `json:\"nullableFieldSkipTrue\"`")
	assert.Contains(t, out, "OptionalField          *string `json:\"optionalField,omitempty\"`")
	assert.Contains(t, out, "OptionalFieldSkipFalse *string `json:\"optionalFieldSkipFalse,omitempty\"`")
	assert.Contains(t, out, "OptionalFieldSkipTrue  string  `json:\"optionalFieldSkipTrue,omitempty\"`")

	// Check that the extension applies on custom types as well
	assert.Contains(t, out, "CustomTypeWithSkipTrue string  `json:\"customTypeWithSkipTrue,omitempty\"`")

	// Check that the extension has no effect on required fields
	assert.Contains(t, out, "RequiredField          string  `json:\"requiredField\"`")
}

func TestGoTypeImport(t *testing.T) {
	opts := Configuration{
		PackageName: "api",
		Generate: []*GenerateOptions{
			{
				Target:  EchoServer,
				Enabled: true,
				Package: "api",
				Output:  "server.go",
			},
			{
				Target:  Models,
				Enabled: true,
				Package: "api",
				Output:  "types.go",
			},
			{
				Target:  EmbeddedSpec,
				Enabled: true,
				Package: "api",
				Output:  "openapi_spec.go",
			},
		},
		OutputOptions: OutputOptions{
			SkipFmt: true,
		},
	}

	for _, t := range opts.Generate {
		if t.Target == Client {
			t.Enabled = false
		}
	}

	spec := "test_specs/x-go-type-import-pet.yaml"
	swagger, err := util.LoadSwagger(spec)
	require.NoError(t, err)

	// Run our code generation:
	code, err := Generate(swagger, opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, code.Output)

	for _, c := range code.Output {
		out := c.Imports + c.Code
		// Check that we have valid (formattable) code:
		_, err = format.Source([]byte(out))
		assert.NoError(t, err)

		imports := []string{
			`github.com/CavernaTechnologies/pgext`, // schemas - direct object
			`myuuid "github.com/google/uuid"`,      // schemas - object
			`github.com/lib/pq`,                    // schemas - array
			`github.com/spf13/viper`,               // responses - direct object
			`golang.org/x/text`,                    // responses - complex object
			`golang.org/x/email`,                   // requestBodies - in components
			`github.com/fatih/color`,               // parameters - query
			`github.com/go-openapi/swag`,           // parameters - path
			`github.com/jackc/pgtype`,              // direct parameters - path
			`github.com/mailru/easyjson`,           // direct parameters - query
			`github.com/subosito/gotenv`,           // direct request body
		}

		// Check import
		for _, imp := range imports {
			assert.Contains(t, out, imp)
		}

		// Make sure the generated code is valid:
		checkLint(t, "test.gen.go", []byte(out))
	}
}

func TestRemoteExternalReference(t *testing.T) {
	packageName := "api"
	opts := Configuration{
		PackageName: packageName,
		Generate: []*GenerateOptions{
			{
				Target:  Models,
				Enabled: true,
				Package: packageName,
				Output:  "types.go",
			},
		},
		ImportMapping: map[string]string{
			remoteRefFile: remoteRefImport,
		},
	}
	spec := "test_specs/remote-external-reference.yaml"
	swagger, err := util.LoadSwagger(spec)
	require.NoError(t, err)

	// Run our code generation:
	code, err := Generate(swagger, opts)
	out := code.GetOutput(Models)
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// Check that we have valid (formattable) code:
	_, err = format.Source([]byte(out))
	assert.NoError(t, err)

	// Check that we have a package:
	assert.Contains(t, out, "package api")

	// Check import
	assert.Contains(t, out, `externalRef0 "github.com/deepmap/oapi-codegen/examples/petstore-expanded"`)

	// Check generated oneOf structure:
	assert.Contains(t, out, `
// ExampleSchema_Item defines model for ExampleSchema.Item.
type ExampleSchema_Item struct {
	union json.RawMessage
}
`)

	// Check generated oneOf structure As method:
	assert.Contains(t, out, `
// AsExternalRef0NewPet returns the union data inside the ExampleSchema_Item as a externalRef0.NewPet
func (t ExampleSchema_Item) AsExternalRef0NewPet() (externalRef0.NewPet, error) {
`)

	// Check generated oneOf structure From method:
	assert.Contains(t, out, `
// FromExternalRef0NewPet overwrites any union data inside the ExampleSchema_Item as the provided externalRef0.NewPet
func (t *ExampleSchema_Item) FromExternalRef0NewPet(v externalRef0.NewPet) error {
`)

	// Check generated oneOf structure Merge method:
	assert.Contains(t, out, `
// FromExternalRef0NewPet overwrites any union data inside the ExampleSchema_Item as the provided externalRef0.NewPet
func (t *ExampleSchema_Item) FromExternalRef0NewPet(v externalRef0.NewPet) error {
`)

	// Make sure the generated code is valid:
	checkLint(t, "test.gen.go", []byte(out))

}

//go:embed test_spec.yaml
var testOpenAPIDefinition string
