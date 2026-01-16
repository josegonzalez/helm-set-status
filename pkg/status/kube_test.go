package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRESTClientGetter(t *testing.T) {
	getter := NewRESTClientGetter("test-namespace")
	assert.NotNil(t, getter)
	assert.Equal(t, "test-namespace", getter.namespace)
}

func TestRESTClientGetter_ToRawKubeConfigLoader(t *testing.T) {
	t.Run("with default settings", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("KUBECONFIG")
		os.Unsetenv("HELM_KUBECONTEXT")

		getter := NewRESTClientGetter("default")
		loader := getter.ToRawKubeConfigLoader()
		assert.NotNil(t, loader)
	})

	t.Run("with KUBECONFIG set", func(t *testing.T) {
		os.Setenv("KUBECONFIG", "/tmp/test-kubeconfig")
		defer os.Unsetenv("KUBECONFIG")

		getter := NewRESTClientGetter("default")
		loader := getter.ToRawKubeConfigLoader()
		assert.NotNil(t, loader)
	})

	t.Run("with HELM_KUBECONTEXT set", func(t *testing.T) {
		os.Setenv("HELM_KUBECONTEXT", "test-context")
		defer os.Unsetenv("HELM_KUBECONTEXT")

		getter := NewRESTClientGetter("default")
		loader := getter.ToRawKubeConfigLoader()
		assert.NotNil(t, loader)
	})

	t.Run("with both KUBECONFIG and HELM_KUBECONTEXT set", func(t *testing.T) {
		os.Setenv("KUBECONFIG", "/tmp/test-kubeconfig")
		os.Setenv("HELM_KUBECONTEXT", "test-context")
		defer func() {
			os.Unsetenv("KUBECONFIG")
			os.Unsetenv("HELM_KUBECONTEXT")
		}()

		getter := NewRESTClientGetter("custom-ns")
		loader := getter.ToRawKubeConfigLoader()
		assert.NotNil(t, loader)
	})
}

func TestRESTClientGetter_ToRESTConfig_Error(t *testing.T) {
	// This will fail because there's no valid kubeconfig
	os.Setenv("KUBECONFIG", "/nonexistent/kubeconfig")
	defer os.Unsetenv("KUBECONFIG")

	getter := NewRESTClientGetter("default")
	_, err := getter.ToRESTConfig()
	assert.Error(t, err)
}

func TestRESTClientGetter_ToDiscoveryClient_Error(t *testing.T) {
	// This will fail because there's no valid kubeconfig
	os.Setenv("KUBECONFIG", "/nonexistent/kubeconfig")
	defer os.Unsetenv("KUBECONFIG")

	getter := NewRESTClientGetter("default")
	_, err := getter.ToDiscoveryClient()
	assert.Error(t, err)
}

func TestRESTClientGetter_ToRESTMapper_Error(t *testing.T) {
	// This will fail because there's no valid kubeconfig
	os.Setenv("KUBECONFIG", "/nonexistent/kubeconfig")
	defer os.Unsetenv("KUBECONFIG")

	getter := NewRESTClientGetter("default")
	_, err := getter.ToRESTMapper()
	assert.Error(t, err)
}

// createTestKubeconfig creates a minimal valid kubeconfig for testing
func createTestKubeconfig(t *testing.T) string {
	t.Helper()

	kubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
    insecure-skip-tls-verify: true
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfig), 0600)
	require.NoError(t, err)
	return kubeconfigPath
}

func TestRESTClientGetter_ToRESTConfig_Success(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	os.Setenv("KUBECONFIG", kubeconfigPath)
	defer os.Unsetenv("KUBECONFIG")

	getter := NewRESTClientGetter("default")
	config, err := getter.ToRESTConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "https://127.0.0.1:6443", config.Host)
}

func TestRESTClientGetter_ToDiscoveryClient_Success(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	os.Setenv("KUBECONFIG", kubeconfigPath)
	defer os.Unsetenv("KUBECONFIG")

	getter := NewRESTClientGetter("default")
	client, err := getter.ToDiscoveryClient()
	// This should succeed in creating the client even if there's no server
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestRESTClientGetter_ToRESTMapper_Success(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	os.Setenv("KUBECONFIG", kubeconfigPath)
	defer os.Unsetenv("KUBECONFIG")

	getter := NewRESTClientGetter("default")
	mapper, err := getter.ToRESTMapper()
	// This should succeed in creating the mapper even if there's no server
	require.NoError(t, err)
	assert.NotNil(t, mapper)
}
