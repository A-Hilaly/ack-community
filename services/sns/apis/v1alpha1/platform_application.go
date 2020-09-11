// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package v1alpha1

import (
	ackv1alpha1 "github.com/aws/aws-controllers-k8s/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PlatformApplicationSpec defines the desired state of PlatformApplication
type PlatformApplicationSpec struct {
	EventDeliveryFailure      *string `json:"eventDeliveryFailure,omitempty"`
	EventEndpointCreated      *string `json:"eventEndpointCreated,omitempty"`
	EventEndpointDeleted      *string `json:"eventEndpointDeleted,omitempty"`
	EventEndpointUpdated      *string `json:"eventEndpointUpdated,omitempty"`
	FailureFeedbackRoleARN    *string `json:"failureFeedbackRoleARN,omitempty"`
	Name                      *string `json:"name,omitempty"`
	Platform                  *string `json:"platform,omitempty"`
	PlatformCredential        *string `json:"platformCredential,omitempty"`
	PlatformPrincipal         *string `json:"platformPrincipal,omitempty"`
	SuccessFeedbackRoleARN    *string `json:"successFeedbackRoleARN,omitempty"`
	SuccessFeedbackSampleRate *string `json:"successFeedbackSampleRate,omitempty"`
}

// PlatformApplicationStatus defines the observed state of PlatformApplication
type PlatformApplicationStatus struct {
	// All CRs managed by ACK have a common `Status.ACKResourceMetadata` member
	// that is used to contain resource sync state, account ownership,
	// constructed ARN for the resource
	ACKResourceMetadata *ackv1alpha1.ResourceMetadata `json:"ackResourceMetadata"`
	// All CRS managed by ACK have a common `Status.Conditions` member that
	// contains a collection of `ackv1alpha1.Condition` objects that describe
	// the various terminal states of the CR and its backend AWS service API
	// resource
	Conditions []*ackv1alpha1.Condition `json:"conditions"`
}

// PlatformApplication is the Schema for the PlatformApplications API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type PlatformApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PlatformApplicationSpec   `json:"spec,omitempty"`
	Status            PlatformApplicationStatus `json:"status,omitempty"`
}

// PlatformApplicationList contains a list of PlatformApplication
// +kubebuilder:object:root=true
type PlatformApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlatformApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlatformApplication{}, &PlatformApplicationList{})
}
