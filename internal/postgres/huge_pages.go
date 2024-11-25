// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// HugePagesWorkaround returns a Bash command that addresses a misconfigured
// "hugetlb" cgroup in old container runtimes. [v1.1.0] of the OCI Runtime
// specifies the behavior we want since 2023, but implementations vary.
//
// NOTE: Regardless of container runtime, this is safe but ineffective on old
// Linux kernels. It works in RHEL since May 2022.
//
// [v1.1.0] https://github.com/opencontainers/runtime-spec/releases/tag/v1.1.0
func HugePagesWorkaround(resources corev1.ResourceRequirements) string {
	// When the node has huge pages but the container requests none, some runtimes
	// assign the "hugetlb" allocation limit and not the reservation limit.
	//
	// By default (huge_pages = try) Postgres checks for huge pages by reserving
	// them during startup. Without a reservation limit, this call succeeds and
	// Postgres quickly crashes when it tries to allocate them.
	//
	// The following sets all "hugetlb" allocation limits and reservation limits
	// to zero when they are present. It considers cgroup v1 and v2.
	//
	// https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v1/hugetlb.html
	// https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#hugetlb
	if !hasHugePages(resources) {
		return "for limit in $(\n" +
			"  compgen -G '/sys/fs/cgroup/hugetlb.*.max'\n" +
			"  compgen -G '/sys/fs/cgroup/hugetlb/hugetlb.*.limit_in_bytes'\n" +
			`); do echo 0 > "${limit}"; done`
	}

	// When the container requires huge pages, some runtimes assign its "hugetlb"
	// allocation limit and not its reservation limit.
	//
	// The following sets each "hugetlb" reservation limit to its corresponding
	// allocation limit in cgroup v1 and v2.
	return "for limit in $(\n" +
		"  compgen -G '/sys/fs/cgroup/hugetlb.*.rsvd.max'\n" +
		"  compgen -G '/sys/fs/cgroup/hugetlb/hugetlb.*.rsvd.limit_in_bytes'\n" +
		`); do cat "${limit/.rsvd./.}" > "${limit}"; done`
}

// This function looks for a valid huge_pages resource request. If it finds one,
// it sets the PostgreSQL parameter "huge_pages" to "try". If it doesn't find
// one, it sets "huge_pages" to "off".
func SetHugePages(cluster *v1beta1.PostgresCluster, pgParameters *Parameters) {
	if HugePagesRequested(cluster) {
		pgParameters.Default.Add("huge_pages", "try")
	} else {
		pgParameters.Default.Add("huge_pages", "off")
	}
}

// This helper function checks to see if a huge_pages value greater than zero has
// been set in any of the PostgresCluster's instances' resource specs
func HugePagesRequested(cluster *v1beta1.PostgresCluster) bool {
	for _, instance := range cluster.Spec.InstanceSets {
		if hasHugePages(instance.Resources) {
			return true
		}
	}
	return false
}

// hasHugePages returns true when resources contains any positive huge pages.
func hasHugePages(resources corev1.ResourceRequirements) bool {
	// Kubernetes requires that huge page requests == limits, so it's sufficient
	// to look only in limits. Instead of prescribing huge page sizes, Kubernetes
	// encodes the size in the resouce name.
	//
	// https://docs.k8s.io/tasks/manage-hugepages/scheduling-hugepages
	for name, quantity := range resources.Limits {
		if strings.HasPrefix(name.String(), corev1.ResourceHugePagesPrefix) &&
			quantity.Value() > 0 {
			return true
		}
	}
	return false
}
