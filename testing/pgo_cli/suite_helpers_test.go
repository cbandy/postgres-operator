package pgo_cli_test

import (
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/crunchydata/postgres-operator/testing/kubeapi"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/storage/names"
)

type Pool struct {
	*kubeapi.Proxy
	*pgxpool.Pool
}

func (p *Pool) Close() error { p.Pool.Close(); return p.Proxy.Close() }

func clusterConnection(t *testing.T, namespace, name, dsn string) *Pool {
	t.Helper()

	pods, err := TestContext.Kubernetes.ListPods(
		namespace, map[string]string{"name": name, "pg-cluster": name})
	require.NoError(t, err)
	require.NotEmpty(t, pods)

	proxy, err := TestContext.Kubernetes.PodPortForward(pods[0].Namespace, pods[0].Name, "5432")
	require.NoError(t, err)

	host, port, err := net.SplitHostPort(proxy.LocalAddr())
	if err != nil {
		proxy.Close()
		require.NoError(t, err)
	}

	pool, err := pgxpool.Connect(context.Background(), dsn+" host="+host+" port="+port)
	if err != nil {
		proxy.Close()
		require.NoError(t, err)
	}

	return &Pool{proxy, pool}
}

func clusterDeployment(t *testing.T, namespace, name string) *apps_v1.Deployment {
	t.Helper()

	deployments, err := TestContext.Kubernetes.ListDeployments(
		namespace, map[string]string{"name": name, "pg-cluster": name})
	require.NoError(t, err)

	if len(deployments) > 0 {
		return &deployments[0]
	}
	return nil
}

func clusterLogs(t *testing.T, namespace, name string) string {
	t.Helper()

	pods, err := TestContext.Kubernetes.ListPods(
		namespace, map[string]string{"name": name, "pg-cluster": name})
	require.NoError(t, err)
	require.NotEmpty(t, pods)

	logs, err := TestContext.Kubernetes.PodLogs(pods[0].Namespace, pods[0].Name, false)
	require.NoError(t, err)

	return logs
}

func clusterPSQL(t *testing.T, namespace, name, psql string) (string, string) {
	t.Helper()

	pods, err := TestContext.Kubernetes.ListPods(
		namespace, map[string]string{"name": name, "pg-cluster": name})
	require.NoError(t, err)
	require.NotEmpty(t, pods)

	stdout, stderr, err := TestContext.Kubernetes.PodExec(
		pods[0].Namespace, pods[0].Name,
		strings.NewReader(psql), "psql", "-U", "postgres", "-f-")
	require.NoError(t, err)

	return stdout, stderr
}

//func podExec(t *testing.T,
//	namespace, name, container string,
//	stdin io.Reader, command ...string,
//) (stdout, stderr string) {
//	t.Helper()
//
//	var err error
//	if container != "" {
//		stdout, stderr, err = TestContext.Kubernetes.Exec(namespace, name, container, stdin, command...)
//	} else {
//		stdout, stderr, err = TestContext.Kubernetes.PodExec(namespace, name, stdin, command...)
//	}
//	require.NoError(t, err)
//	return
//}

func requireClusterReady(t *testing.T, namespace, name string, timeout time.Duration) {
	t.Helper()

	ready := func() bool {
		deployment := clusterDeployment(t, namespace, name)
		return deployment != nil &&
			deployment.Status.Replicas == deployment.Status.ReadyReplicas
	}

	if !ready() {
		require.Eventuallyf(t, ready, timeout, time.Second,
			"timeout waiting for %q in %q", name, namespace)
	}
}

func withCluster(t *testing.T, namespace func() string, during func(func() string)) {
	t.Helper()

	var name string
	var once sync.Once

	during(func() string {
		once.Do(func() {
			name = names.SimpleNameGenerator.GenerateName("pgo-test-")
			_, err := pgo("create", "cluster", name, "-n", namespace()).Exec(t)
			assert.NoError(t, err)
		})
		return name
	})
}

func withNamespace(t *testing.T, during func(func() string)) {
	t.Helper()

	// Use the namespace specified in the environment.
	if name := os.Getenv("PGO_NAMESPACE"); name != "" {
		during(func() string { return name })
		return
	}

	// Prepare to cleanup a namespace that might be created.
	var namespace *core_v1.Namespace
	var once sync.Once

	defer func() {
		if namespace != nil {
			err := TestContext.Kubernetes.DeleteNamespace(namespace.Name)
			assert.NoErrorf(t, err, "unable to tear down namespace %q", namespace.Name)
		}
	}()

	during(func() string {
		once.Do(func() {
			ns, err := TestContext.Kubernetes.GenerateNamespace("pgo-test-")
			require.NoError(t, err)
			namespace = ns

			_, err = pgo("update", "namespace", namespace.Name).Exec(t)
			assert.NoErrorf(t, err, "unable to take ownership of namespace %q", namespace.Name)
		})

		return namespace.Name
	})
}
