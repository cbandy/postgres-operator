// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:validation:XValidation:message=`syncPeriodSeconds must be less than leaderLeaseDurationSeconds`,rule=`self.syncPeriodSeconds < self.leaderLeaseDurationSeconds`
type PatroniSpec struct {
	// Patroni dynamic configuration settings. Changes to this value will be
	// automatically reloaded without validation. Changes to certain PostgreSQL
	// parameters cause PostgreSQL to restart.
	// More info: https://patroni.readthedocs.io/en/latest/dynamic_configuration.html
	// ---
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	DynamicConfiguration v1beta1.SchemalessObject `json:"dynamicConfiguration,omitempty"`

	// TTL of the cluster leader lock. "Think of it as the
	// length of time before initiation of the automatic failover process."
	// Changing this value causes PostgreSQL to restart.
	// ---
	// +optional
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=3
	LeaderLeaseDurationSeconds *int32 `json:"leaderLeaseDurationSeconds,omitempty"`

	// Patroni log configuration settings.
	// +optional
	Logging *PatroniLogConfig `json:"logging,omitempty"`

	// The port on which Patroni should listen.
	// Changing this value causes PostgreSQL to restart.
	// ---
	// +optional
	// +kubebuilder:default=8008
	// +kubebuilder:validation:Minimum=1024
	Port *int32 `json:"port,omitempty"`

	// The interval for refreshing the leader lock and applying
	// dynamicConfiguration. Must be less than leaderLeaseDurationSeconds.
	// Changing this value causes PostgreSQL to restart.
	// ---
	// +optional
	// +kubebuilder:default=10
	// +kubebuilder:validation:Minimum=1
	SyncPeriodSeconds *int32 `json:"syncPeriodSeconds,omitempty"`

	// Switchover gives options to perform ad hoc switchovers in a PostgresCluster.
	// +optional
	Switchover *v1beta1.PatroniSwitchover `json:"switchover,omitempty"`
}

type PatroniLogConfig struct {

	// Limits the total amount of space taken by Patroni log files.
	// Minimum value is 25MB.
	// More info: https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity
	// ---
	// +kubebuilder:validation:XValidation:message=`cannot be smaller than 25M`,rule=`!self.isLessThan(quantity("25M"))`
	// +required
	StorageLimit *resource.Quantity `json:"storageLimit"`

	// The Patroni log level.
	// More info: https://docs.python.org/3/library/logging.html#levels
	// ---
	// +default="INFO"
	// +kubebuilder:validation:Enum={CRITICAL,ERROR,WARNING,INFO,DEBUG,NOTSET}
	// +optional
	Level *string `json:"level,omitempty"`
}

// Default sets the default values for certain Patroni configuration attributes, including:
// - Lock Lease Duration
// - Patroni's API port
// - Frequency of syncing with Kube API
func (s *PatroniSpec) Default() {
	if s.LeaderLeaseDurationSeconds == nil {
		s.LeaderLeaseDurationSeconds = new(int32)
		*s.LeaderLeaseDurationSeconds = 30
	}
	if s.Port == nil {
		s.Port = new(int32)
		*s.Port = 8008
	}
	if s.SyncPeriodSeconds == nil {
		s.SyncPeriodSeconds = new(int32)
		*s.SyncPeriodSeconds = 10
	}
}
