package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfiguration(t *testing.T) {
	// Default configuration with package name
	opts := NewDefaultConfigurationWithPackage("api")

	// Validate configuration. This will create the targets
	err := opts.Validate()
	assert.NoError(t, err)
	require.NotNil(t, opts.Targets, "No targets found")

	// Make sure we have a EchoServer target with package name equal to 'api'
	target, _ := opts.Targets[EchoServer]
	require.NotNil(t, target, "Expected target not found: %s", EchoServer)
	assert.Equal(t, target.Package, "api")

	// Make sure we have a Client target with package name equal to 'api'
	target, _ = opts.Targets[Client]
	require.NotNil(t, target, "Expected target not found: %s", Client)
	assert.Equal(t, target.Package, "api")

	// Make sure we have a Models target with package name equal to 'api'
	target, _ = opts.Targets[Models]
	require.NotNil(t, target, "Expected target not found: %s", Models)
	assert.Equal(t, target.Package, "api")

	// Make sure we have a EmbeddedSpec target with package name equal to 'api'
	target, _ = opts.Targets[EmbeddedSpec]
	require.NotNil(t, target, "Expected target not found: %s", EmbeddedSpec)
	assert.Equal(t, target.Package, "api")
}

func TestServerAndSpecInSamePackage(t *testing.T) {
	// Default configuration with package name
	opts := NewDefaultConfiguration()
	opts.PackageName = "echo=internal/api/server,client=pkg/api/client,types=pkg/api/models,spec=internal/api/server"

	// Validate configuration. This will create the targets
	err := opts.Validate()
	assert.NoError(t, err)
	require.NotNil(t, opts.Targets, "No targets found")

	// Make sure we have a EchoServer target with package name equal to 'internal/api/server'
	target, _ := opts.Targets[EchoServer]
	require.NotNil(t, target, "Expected target not found: %s", EchoServer)
	assert.Equal(t, target.Package, "internal/api/server")

	// Make sure we have a EmbeddedSpec target with package name equal to 'internal/api/server'
	target, _ = opts.Targets[EmbeddedSpec]
	require.NotNil(t, target, "Expected target not found: %s", EmbeddedSpec)
	assert.Equal(t, target.Package, "internal/api/server")
}

func TestServerAndSpecInSamePackageAndFile(t *testing.T) {
	// Default configuration with package name
	opts := NewDefaultConfiguration()
	opts.PackageName = "echo=internal/api/server,client=pkg/api/client,types=pkg/api/models,spec=internal/api/server"
	opts.OutputFile = "echo=server.go,client=client.go,types=types.go,spec=server.go"

	// Validate configuration. This will create the targets
	err := opts.Validate()
	assert.NoError(t, err)
	require.NotNil(t, opts.Targets, "No targets found")

	// Make sure we have a EchoServer target with package name equal to 'internal/api/server'
	// and that the specified output file is 'server.go'
	target, _ := opts.Targets[EchoServer]
	require.NotNil(t, target, "Expected target not found: %s", EchoServer)
	assert.Equal(t, target.Package, "internal/api/server")
	assert.Equal(t, target.FileName, "server.go")

	// Make sure we have a EmbeddedSpec target with package name equal to 'internal/api/server'
	// and that the specified output file is 'server.go'
	target, _ = opts.Targets[EmbeddedSpec]
	require.NotNil(t, target, "Expected target not found: %s", EmbeddedSpec)
	assert.Equal(t, target.Package, "internal/api/server")
	assert.Equal(t, target.FileName, "server.go")
}

func TestEverythingInSamePackageAndFile(t *testing.T) {
	// Default configuration with package name
	opts := NewDefaultConfiguration()
	opts.PackageName = "api"
	opts.OutputFile = "api.go"

	// Validate configuration. This will create the targets
	err := opts.Validate()
	assert.NoError(t, err)
	require.NotNil(t, opts.Targets, "No targets found")

	// Make sure we have a EchoServer target with package name equal to 'api'
	target, _ := opts.Targets[EchoServer]
	require.NotNil(t, target, "Expected target not found: %s", EchoServer)
	assert.Equal(t, target.Package, "api")
	assert.Equal(t, target.FileName, "api.go")

	// Make sure we have a Client target with package name equal to 'api'
	target, _ = opts.Targets[Client]
	require.NotNil(t, target, "Expected target not found: %s", Client)
	assert.Equal(t, target.Package, "api")
	assert.Equal(t, target.FileName, "api.go")

	// Make sure we have a Models target with package name equal to 'api'
	target, _ = opts.Targets[Models]
	require.NotNil(t, target, "Expected target not found: %s", Models)
	assert.Equal(t, target.Package, "api")
	assert.Equal(t, target.FileName, "api.go")

	// Make sure we have a EmbeddedSpec target with package name equal to 'api'
	target, _ = opts.Targets[EmbeddedSpec]
	require.NotNil(t, target, "Expected target not found: %s", EmbeddedSpec)
	assert.Equal(t, target.Package, "api")
	assert.Equal(t, target.FileName, "api.go")
}

func TestDefaultConfigurationWithSecondaryServer(t *testing.T) {
	// Default configuration with package name
	opts := NewDefaultConfiguration()
	// Enable GinServer target. This would normally be done using the -generate flag and
	// the generationTargets method in oapi-codegen.go, but here we need to explicitly add
	// the target
	opts.Targets[GinServer] = GenerateTargets[GinServer]
	// Specify package names
	opts.PackageName = "echo=internal/api/server/echo,gin=internal/api/server/gin,api"

	// Validate configuration. This will (re)create the targets
	// Also, having multiple servers should be allowed
	err := opts.Validate()
	assert.NoError(t, err)
	require.NotNil(t, opts.Targets, "No targets found")

	// Make sure we have a EchoServer target with package name equal to 'internal/api/server/gin'
	target, _ := opts.Targets[EchoServer]
	require.NotNil(t, target, "Expected target not found: %s", EchoServer)
	assert.Equal(t, target.Package, "internal/api/server/echo")

	// Make sure we have a GinServer target with package name equal to 'internal/api/server/gin'
	target, _ = opts.Targets[GinServer]
	require.NotNil(t, target, "Expected target not found: %s", GinServer)
	assert.Equal(t, target.Package, "internal/api/server/gin")

	// The rest of the targets hould have package name equal to 'api'
	target, _ = opts.Targets[Client]
	require.NotNil(t, target, "Expected target not found: %s", Client)
	assert.Equal(t, target.Package, "api")

	// The rest of the targets hould have package name equal to 'api'
	target, _ = opts.Targets[Models]
	require.NotNil(t, target, "Expected target not found: %s", Models)
	assert.Equal(t, target.Package, "api")

	// The rest of the targets hould have package name equal to 'api'
	target, _ = opts.Targets[EmbeddedSpec]
	require.NotNil(t, target, "Expected target not found: %s", EmbeddedSpec)
	assert.Equal(t, target.Package, "api")
}

func TestGenerateTargetSpecifiedMoreThanOnce(t *testing.T) {
	// Default configuration with package name
	opts := NewDefaultConfiguration()
	opts.PackageName = "client=pkg/api/client,client=internal/api/client,server"

	// Validate configuration. This should fail as the client target is specified more than once
	err := opts.Validate()
	assert.Error(t, err, "A target mapping already exists: client")
}

func TestTargetSpecifiedMoreThanOnce(t *testing.T) {
	// Default configuration with package name
	opts := NewDefaultConfiguration()
	opts.PackageName = "echo=internal/api/server,models,client=internal/api/client,models"

	// Validate configuration. This should fail as the client target is specified more than once
	err := opts.Validate()
	assert.Error(t, err, "A mapping without target was specified more than once: models")
}
