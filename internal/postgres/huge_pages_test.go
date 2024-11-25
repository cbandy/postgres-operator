// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestHugePagesWorkaround(t *testing.T) {
	t.Parallel()

	alwaysExpect := func(t *testing.T, script string) {
		t.Helper()

		t.Run("ShellCheckBash", func(t *testing.T) {
			shellcheck := require.ShellCheck(t)

			dir := t.TempDir()
			file := filepath.Join(dir, "script.bash")
			assert.NilError(t, os.WriteFile(file, []byte(script), 0o600))

			// Expect ShellCheck for Bash to be happy.
			// - https://www.shellcheck.net/wiki/SC2148
			cmd := exec.Command(shellcheck, "--enable=all", "--shell=bash", file)
			output, err := cmd.CombinedOutput()
			assert.NilError(t, err, "%q\n%s", cmd.Args, output)
		})

		t.Run("PrettyYAML", func(t *testing.T) {
			b, err := yaml.Marshal(script)
			assert.NilError(t, err)
			assert.Assert(t, strings.HasPrefix(string(b), `|`),
				"expected literal block scalar, got:\n%s", b)
		})
	}

	t.Run("NoHugePages", func(t *testing.T) {
		for _, tt := range []corev1.ResourceRequirements{
			// zero value, no requirements
			{},

			// explicit zero huge pages
			{Limits: corev1.ResourceList{"hugepages-2Mi": resource.MustParse("0")}},
		} {
			script := HugePagesWorkaround(tt)

			assert.Assert(t, cmp.Contains(script, `echo 0`))
			assert.Assert(t, cmp.Contains(script, `/sys/fs/cgroup/hugetlb`))
			assert.Assert(t, cmp.Contains(script, `.limit_in_bytes`))
			assert.Assert(t, cmp.Contains(script, `.max`))

			alwaysExpect(t, script)
		}
	})

	t.Run("HugePages", func(t *testing.T) {
		script := HugePagesWorkaround(corev1.ResourceRequirements{
			Limits: corev1.ResourceList{"hugepages-2Mi": resource.MustParse("1Gi")},
		})

		assert.Assert(t, cmp.Contains(script, `/sys/fs/cgroup/hugetlb`))
		assert.Assert(t, cmp.Contains(script, `.rsvd.limit_in_bytes`))
		assert.Assert(t, cmp.Contains(script, `.rsvd.max`))

		alwaysExpect(t, script)
	})
}

func TestSetHugePages(t *testing.T) {
	t.Run("hugepages not set at all", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)

		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:     "test-instance1",
			Replicas: initialize.Int32(1),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{},
			},
		}}

		pgParameters := NewParameters()
		SetHugePages(cluster, &pgParameters)

		assert.Equal(t, pgParameters.Default.Has("huge_pages"), true)
		assert.Equal(t, pgParameters.Default.Value("huge_pages"), "off")
	})

	t.Run("hugepages quantity not set", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)

		emptyQuantity, _ := resource.ParseQuantity("")
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:     "test-instance1",
			Replicas: initialize.Int32(1),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceHugePagesPrefix + "2Mi": emptyQuantity,
				},
			},
		}}

		pgParameters := NewParameters()
		SetHugePages(cluster, &pgParameters)

		assert.Equal(t, pgParameters.Default.Has("huge_pages"), true)
		assert.Equal(t, pgParameters.Default.Value("huge_pages"), "off")
	})

	t.Run("hugepages set to zero", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)

		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:     "test-instance1",
			Replicas: initialize.Int32(1),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceHugePagesPrefix + "2Mi": resource.MustParse("0Mi"),
				},
			},
		}}

		pgParameters := NewParameters()
		SetHugePages(cluster, &pgParameters)

		assert.Equal(t, pgParameters.Default.Has("huge_pages"), true)
		assert.Equal(t, pgParameters.Default.Value("huge_pages"), "off")
	})

	t.Run("hugepages set correctly", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)

		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:     "test-instance1",
			Replicas: initialize.Int32(1),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceHugePagesPrefix + "2Mi": resource.MustParse("16Mi"),
				},
			},
		}}

		pgParameters := NewParameters()
		SetHugePages(cluster, &pgParameters)

		assert.Equal(t, pgParameters.Default.Has("huge_pages"), true)
		assert.Equal(t, pgParameters.Default.Value("huge_pages"), "try")
	})

}
