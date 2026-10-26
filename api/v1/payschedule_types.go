/*
Copyright 2023 Masudur Rahman.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PayScheduleSpec defines the desired state of PaySchedule
type PayScheduleSpec struct {
	// CronName is the name of the CronJob created for this PaySchedule
	//+kubebuilder:validation:MinLength=1
	CronName string `json:"cronName"`
	// CronSchedule is the schedule in Cron format, e.g. "*/5 * * * *"
	//+kubebuilder:validation:MinLength=1
	CronSchedule string `json:"cronSchedule"`
	// JobImage is the container image the job runs
	//+kubebuilder:validation:MinLength=1
	JobImage string `json:"jobImage"`
	// JobCmd is the shell command the job runs, via `/bin/sh -c`
	//+kubebuilder:validation:MinLength=1
	JobCmd string `json:"jobCmd"`
}

// PayScheduleStatus defines the observed state of PaySchedule
type PayScheduleStatus struct {
	LastScheduled *metav1.Time `json:"lastScheduled,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="CronJob",type=string,JSONPath=".spec.cronName"
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=".spec.cronSchedule"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Last Scheduled",type=date,JSONPath=".status.lastScheduled"

// PaySchedule is the Schema for the payschedules API
type PaySchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PayScheduleSpec   `json:"spec,omitempty"`
	Status PayScheduleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PayScheduleList contains a list of PaySchedule
type PayScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PaySchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PaySchedule{}, &PayScheduleList{})
}
