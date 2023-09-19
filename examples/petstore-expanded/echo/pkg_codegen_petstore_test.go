package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io"
	"net"
	"net/http"
	"testing"

	examplePetstoreClient "github.com/deepmap/oapi-codegen/examples/petstore-expanded"
	examplePetstore "github.com/deepmap/oapi-codegen/examples/petstore-expanded/echo/api"
	"github.com/deepmap/oapi-codegen/pkg/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/lint"
)

func checkLint(t *testing.T, filename string, code []byte) {
	linter := new(lint.Linter)
	problems, err := linter.Lint(filename, code)
	assert.NoError(t, err)
	assert.Len(t, problems, 0)
}

func TestExamplePetStoreCodeGeneration(t *testing.T) {
	// Input vars for code generation:
	opts := codegen.NewDefaultConfigurationWithPackage("petstore")

	// Get a spec from the example PetStore definition:
	swagger, err := examplePetstore.GetSwagger()
	assert.NoError(t, err)

	// Run our code generation:
	output, err := codegen.Generate(swagger, opts)
	require.NotNil(t, output, "No targets found")
	assert.NoError(t, err)

	// Make sure we have a EchoServer target with a summary comment containing newlines
	target, _ := output[codegen.EchoServer]
	require.NotNil(t, target, "Expected target not found: %s", codegen.EchoServer)
	assert.Contains(t, target.Code, "// Deletes a pet by ID")

	// Make sure we have a Client target with method signatures returing response structs:
	target, _ = output[codegen.Client]
	require.NotNil(t, target, "Expected target not found: %s", codegen.Client)
	assert.Contains(t, target.Code, "func (c *Client) FindPetByID(ctx context.Context, id int64, reqEditors ...RequestEditorFn) (*http.Response, error) {")

	// Make sure we have a Models target with property comments:
	target, _ = output[codegen.Models]
	require.NotNil(t, target, "Expected target not found: %s", codegen.Models)
	assert.Contains(t, target.Code, "// Id Unique id of the pet")

	// Loop through the code
	for _, c := range output.ToArray() {
		// Check that we have valid (formattable) code:
		_, err = format.Source([]byte(c.Code))
		assert.NoError(t, err)

		// Check that we have a package:
		assert.Contains(t, c.Code, "package petstore")

		// Make sure the generated code is valid:
		checkLint(t, "test.gen.go", []byte(c.Code))
	}
}

func TestExamplePetStoreCodeGenerationWithUserTemplates(t *testing.T) {
	userTemplates := map[string]string{"typedef.tmpl": "//blah\n//blah"}

	// Input vars for code generation:
	opts := codegen.Configuration{
		Generate: codegen.GenerateOptions{
			Models: true,
		},
		OutputOptions: codegen.OutputOptions{
			UserTemplates: userTemplates,
		},
		Targets: map[string]*codegen.GenerateTarget{
			codegen.Models: codegen.GenerateTargets[codegen.Models],
		},
	}
	opts.Targets[codegen.Models].Package = "models"

	// Get a spec from the example PetStore definition:
	swagger, err := examplePetstore.GetSwagger()
	assert.NoError(t, err)

	// Run our code generation:
	output, err := codegen.Generate(swagger, opts)
	require.NotNil(t, output, "No targets found")
	assert.NoError(t, err)

	target, _ := output[codegen.Models]
	require.NotNil(t, target, "Expected target not found: %s", codegen.Models)
	assert.NotEmpty(t, target.Code)

	// Check that we have valid (formattable) code:
	_, err = format.Source([]byte(target.Code))
	assert.NoError(t, err)

	// Check that we have a package:
	assert.Contains(t, target.Code, "package models")

	// Check that the built-in template has been overridden
	assert.Contains(t, target.Code, "//blah")
}

func TestExamplePetStoreCodeGenerationWithFileUserTemplates(t *testing.T) {
	userTemplates := map[string]string{"typedef.tmpl": "../../../pkg/codegen/templates/typedef.tmpl"}

	// Input vars for code generation:
	opts := codegen.Configuration{
		Generate: codegen.GenerateOptions{
			Models: true,
		},
		OutputOptions: codegen.OutputOptions{
			UserTemplates: userTemplates,
		},
		Targets: map[string]*codegen.GenerateTarget{
			codegen.Models: codegen.GenerateTargets[codegen.Models],
		},
	}
	opts.Targets[codegen.Models].Package = "models"

	// Get a spec from the example PetStore definition:
	swagger, err := examplePetstore.GetSwagger()
	assert.NoError(t, err)

	// Run our code generation:
	output, err := codegen.Generate(swagger, opts)
	require.NotNil(t, output, "No targets found")
	assert.NoError(t, err)

	target, _ := output[codegen.Models]
	require.NotNil(t, target, "Expected target not found: %s", codegen.Models)
	assert.NotEmpty(t, target.Code)

	// Check that we have valid (formattable) code:
	_, err = format.Source([]byte(target.Code))
	assert.NoError(t, err)

	// Check that we have a package:
	assert.Contains(t, target.Code, "package models")
	// Check that the built-in template has been overridden
	assert.Contains(t, target.Code, "// Package models provides primitives to interact with the openapi")
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
	//packageName := "api"
	opts := codegen.Configuration{
		Generate: codegen.GenerateOptions{
			Models: true,
		},
		OutputOptions: codegen.OutputOptions{
			UserTemplates: userTemplates,
		},
		Targets: map[string]*codegen.GenerateTarget{
			codegen.Models: codegen.GenerateTargets[codegen.Models],
		},
	}
	opts.Targets[codegen.Models].Package = "models"

	// Get a spec from the example PetStore definition:
	swagger, err := examplePetstore.GetSwagger()
	assert.NoError(t, err)

	// Run our code generation:
	output, err := codegen.Generate(swagger, opts)
	require.NotNil(t, output, "No targets found")
	assert.NoError(t, err)

	target, _ := output[codegen.Models]
	require.NotNil(t, target, "Expected target not found: %s", codegen.Models)
	assert.NotEmpty(t, target.Code)

	// Check that we have valid (formattable) code:
	_, err = format.Source([]byte(target.Code))
	assert.NoError(t, err)

	// Check that we have a package:
	assert.Contains(t, target.Code, "package models")

	// Check that the built-in template has been overriden
	assert.Contains(t, target.Code, "//blah")
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
