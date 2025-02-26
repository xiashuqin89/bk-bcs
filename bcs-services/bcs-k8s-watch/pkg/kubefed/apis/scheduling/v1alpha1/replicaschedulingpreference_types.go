/*
 * Tencent is pleased to support the open source community by making Blueking Container Service available.
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReplicaSchedulingPreferenceSpec defines the desired state of ReplicaSchedulingPreference
type ReplicaSchedulingPreferenceSpec struct {
	// TODO (@irfanurrehman); upgrade this to label selector only if need be.
	// The idea of this API is to have a a set of preferences which can
	// be used for a target FederatedDeployment or FederatedReplicaset.
	// Although the set of preferences in question can be applied to multiple
	// target objects using label selectors, but there are no clear advantages
	// of doing that as of now.
	// To keep the implementation and usage simple, matching ns/name of RSP
	// resource to the target resource is sufficient and only additional information
	// needed in RSP resource is a target kind (FederatedDeployment or FederatedReplicaset).
	TargetKind string `json:"targetKind"`

	// Total number of pods desired across federated clusters.
	// Replicas specified in the spec for target deployment template or replicaset
	// template will be discarded/overridden when scheduling preferences are
	// specified.
	TotalReplicas int32 `json:"totalReplicas"`

	// If set to true then already scheduled and running replicas may be moved to other clusters
	// in order to match current state to the specified preferences. Otherwise, if set to false,
	// up and running replicas will not be moved.
	// +optional
	Rebalance bool `json:"rebalance,omitempty"`

	// A mapping between cluster names and preferences regarding a local workload object (dep, rs, .. ) in
	// these clusters.
	// "*" (if provided) applies to all clusters if an explicit mapping is not provided.
	// If omitted, clusters without explicit preferences should not have any replicas scheduled.
	// +optional
	Clusters map[string]ClusterPreferences `json:"clusters,omitempty"`
}

// ClusterPreferences xxx
// Preferences regarding number of replicas assigned to a cluster workload object (dep, rs, ..) within
// a federated workload object.
type ClusterPreferences struct {
	// Minimum number of replicas that should be assigned to this cluster workload object. 0 by default.
	// +optional
	MinReplicas int64 `json:"minReplicas,omitempty"`

	// Maximum number of replicas that should be assigned to this cluster workload object.
	// Unbounded if no value provided (default).
	// +optional
	MaxReplicas *int64 `json:"maxReplicas,omitempty"`

	// A number expressing the preference to put an additional replica to this cluster workload object.
	// 0 by default.
	Weight int64 `json:"weight,omitempty"`
}

// ReplicaSchedulingPreferenceStatus defines the observed state of ReplicaSchedulingPreference
type ReplicaSchedulingPreferenceStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReplicaSchedulingPreference xxx
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=replicaschedulingpreferences
type ReplicaSchedulingPreference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReplicaSchedulingPreferenceSpec   `json:"spec,omitempty"`
	Status ReplicaSchedulingPreferenceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReplicaSchedulingPreferenceList contains a list of ReplicaSchedulingPreference
type ReplicaSchedulingPreferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReplicaSchedulingPreference `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReplicaSchedulingPreference{}, &ReplicaSchedulingPreferenceList{})
}
