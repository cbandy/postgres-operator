package pgo_cli_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC126 ✓
var _ = describe("Cluster Commands", func(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("test", func(t *testing.T) {
				t.Run("shows something immediately", func(t *testing.T) {
					output, err := pgo("test", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "DOWN")
				})

				t.Run("detects the cluster eventually", func(t *testing.T) {
					t.Parallel()

					var output string
					var err error

					check := func() bool {
						output, err = pgo("test", cluster(), "-n", namespace()).Exec(t)
						require.NoError(t, err)
						return strings.Contains(output, "UP")
					}

					if !check() && !assert.Eventually(t, check, time.Minute, time.Second) {
						require.Contains(t, output, "UP")
					}
				})
			})
		})
	})
})
