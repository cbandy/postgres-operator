package pgo_cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TC42 ✓
var _ = describe("Cluster Commands", func(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("label", func(t *testing.T) {
				t.Run("modifies the cluster", func(t *testing.T) {
					output, err := pgo("label", cluster(), "--label=purpose=power", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "applied")

					output, err = pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "purpose=power")

					output, err = pgo("show", "cluster", "--selector=purpose=power", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, cluster())
				})
			})
		})
	})
})
