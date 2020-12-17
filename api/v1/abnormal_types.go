/*
Copyright 2020 The Kube Diagnoser Authors.

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
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// PrometheusAlertSource means that the abnormal is detected via prometheus alert.
	PrometheusAlertSource AbnormalSourceType = "PrometheusAlert"
	// KubernetesEventSource means that the abnormal is detected via kubernetes event.
	KubernetesEventSource AbnormalSourceType = "KubernetesEvent"
	// CustomSource means that the abnormal is a customized abnormal created by user.
	CustomSource AbnormalSourceType = "Custom"

	// InformationCollecting means that the information manager is sending abnormal to assigned
	// information collectors.
	InformationCollecting AbnormalPhase = "InformationCollecting"
	// AbnormalDiagnosing means that the abnormal has been passed to diagnoser chain and some of
	// the diagnosers have been started. At least one diagnoser is still running.
	AbnormalDiagnosing AbnormalPhase = "Diagnosing"
	// AbnormalRecovering means that the abnormal has been passed to recoverer chain and some of
	// the recoverers have been started. At least one recoverer is still running.
	AbnormalRecovering AbnormalPhase = "Recovering"
	// AbnormalSucceeded means that the abnormal has been successfully recovered by some of
	// the recoverers.
	AbnormalSucceeded AbnormalPhase = "Succeeded"
	// AbnormalFailed means that all diagnosers and recoverers have been executed, and none of
	// diagnosers and recoverers is able to diagnose and recover the abnormal.
	AbnormalFailed AbnormalPhase = "Failed"
	// AbnormalUnknown means that for some reason the state of the abnormal could not be obtained.
	AbnormalUnknown AbnormalPhase = "Unknown"

	// InformationCollectorType means that the command executor is an information collector.
	InformationCollectorType AbnormalProcessorType = "InformationCollector"
	// DiagnoserType means that the command executor is an diagnoser.
	DiagnoserType AbnormalProcessorType = "Diagnoser"
	// RecovererType means that the command executor is an recoverer.
	RecovererType AbnormalProcessorType = "Recoverer"

	// InformationCollected means that the abnormal has been passed to information manager.
	InformationCollected AbnormalConditionType = "InformationCollected"
	// AbnormalIdentified means that the abnormal has been identified by the diagnoser chain.
	AbnormalIdentified AbnormalConditionType = "Identified"
	// AbnormalRecovered means that the abnormal has been recovered by the recoverer chain.
	AbnormalRecovered AbnormalConditionType = "Recovered"

	// ArthasJavaProfilerType means that the java profiler is run by arthas.
	ArthasJavaProfilerType JavaProfilerType = "Arthas"
	// MemoryAnalyzerJavaProfilerType means that the java profiler is run by eclipse memory analyzer.
	MemoryAnalyzerJavaProfilerType JavaProfilerType = "MemoryAnalyzer"
)

// AbnormalSpec defines the desired state of Abnormal.
type AbnormalSpec struct {
	// Source is the abnormal source. Valid sources are PrometheusAlert, KubernetesEvent and Custom.
	Source AbnormalSourceType `json:"source"`
	// PrometheusAlert contains the prometheus alert about the abnormal from prometheus
	// alert source. This must be specified if abnormal source is PrometheusAlert.
	// +optional
	PrometheusAlert *PrometheusAlert `json:"prometheusAlert,omitempty"`
	// KubernetesEvent contains the kubernetes event about the abnormal from kubernetes
	// event source. This must be specified if abnormal source is KubernetesEvent.
	// +optional
	KubernetesEvent *corev1.Event `json:"kubernetesEvent,omitempty"`
	// One of NodeName and PodReference must be specified.
	// NodeName is a specific node which the abnormal is on.
	// +optional
	NodeName string `json:"nodeName,omitempty"`
	// PodReference contains details of the target pod.
	// +optional
	PodReference *PodReference `json:"podReference,omitempty"`
	// AssignedInformationCollectors is the list of information collectors to execute
	// information collecting logics. Information collectors would be executed in the
	// specified sequence. Only assigned information collectors will be executed.
	// +optional
	AssignedInformationCollectors []NamespacedName `json:"assignedInformationCollectors,omitempty"`
	// AssignedDiagnosers is the list of diagnosers to execute diagnosing logics.
	// Diagnosers would be executed in the specified sequence. Only assigned diagnosers
	// will be executed.
	// +optional
	AssignedDiagnosers []NamespacedName `json:"assignedDiagnosers,omitempty"`
	// AssignedRecoverers is the list of recoverers to execute recovering logics.
	// Recoverers would be executed in the specified sequence. Only assigned recoverers
	// will be executed.
	// +optional
	AssignedRecoverers []NamespacedName `json:"assignedRecoverers,omitempty"`
	// CommandExecutors is the list of commands to execute during information collecting, diagnosing
	// and recovering.
	// +optional
	CommandExecutors []CommandExecutorSpec `json:"commandExecutors,omitempty"`
	// Profilers is the list of profiler desired behaviors to be performed during information collecting,
	// diagnosing and recovering.
	// +optional
	Profilers []ProfilerSpec `json:"profilers,omitempty"`
	// Context is a blob of information about the abnormal, meant to be user-facing
	// content and display instructions. This field may contain customized values for
	// custom source.
	// +optional
	Context *runtime.RawExtension `json:"context,omitempty"`
}

// AbnormalSourceType is the source of abnormals.
type AbnormalSourceType string

// PrometheusAlert is a generic representation of an prometheus alert.
// It is the "Alert" type in model.go: https://github.com/prometheus/common/blob/v0.12.0/model/alert.go#L29.
type PrometheusAlert struct {
	// Labels contains label value pairs for purpose of aggregation, matching, and disposition
	// dispatching. This must minimally include an "alertname" label.
	Labels model.LabelSet `json:"labels"`
	// Annotations contains extra key value information which does not define alert identity.
	Annotations model.LabelSet `json:"annotations"`
	// StartsAt specifies the known start time for this alert.
	// +optional
	StartsAt metav1.Time `json:"startsAt,omitempty"`
	// EndsAt specifies the known end time for this alert.
	// +optional
	EndsAt metav1.Time `json:"endsAt,omitempty"`
	// GeneratorURL specifies the url of alert generator.
	GeneratorURL string `json:"generatorURL"`
}

// PodReference contains details of the target pod.
type PodReference struct {
	// Namespace specifies the namespace of a pod.
	Namespace string `json:"namespace"`
	// Name specifies the name of a pod.
	Name string `json:"name"`
	// ContainerName specifies name of the target container.
	// +optional
	ContainerName string `json:"containerName,omitempty"`
}

// NamespacedName represents a kubernetes api resource.
type NamespacedName struct {
	// Namespace specifies the namespace of a kubernetes api resource.
	Namespace string `json:"namespace"`
	// Name specifies the name of a kubernetes api resource.
	Name string `json:"name"`
}

// CommandExecutorSpec describes how to execute a command with the given arguments. A CommandExecutor
// could be an information collector, a diagnoser or a recoverer.
type CommandExecutorSpec struct {
	// Command represents a command being prepared and run.
	Command []string `json:"command"`
	// Type is the type of the command executor. There are three possible type values:
	//
	// InformationCollector: The command executor will be run by information manager.
	// Diagnoser: The command executor will be run by diagnoser chain.
	// Recoverer: The command executor will be run by recoverer chain.
	Type AbnormalProcessorType `json:"type"`
	// Number of seconds after which the command times out.
	// Defaults to 30 seconds. Minimum value is 1.
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`
}

// ProfilerSpec describes desired behavior of a profiler to be performed against a program to determine
// its performance status.
type ProfilerSpec struct {
	// Name specifies the name of a profiler.
	Name string `json:"name"`
	// Type is the type of the profiler. There are three possible type values:
	//
	// InformationCollector: The profiler will be run by information manager.
	// Diagnoser: The profiler will be run by diagnoser chain.
	// Recoverer: The profiler will be run by recoverer chain.
	Type AbnormalProcessorType `json:"type"`
	// One and only one of the following programming languages should be specified.
	// Go specifies the action to perform for profiling a go program.
	// +optional
	Go *GoProfilerSpec `json:"go,omitempty"`
	// Java specifies the action to perform for profiling a java program.
	// +optional
	Java *JavaProfilerSpec `json:"java,omitempty"`
	// Number of seconds after which the profiler times out.
	// Defaults to 30 seconds. Minimum value is 1.
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`
	// Number of seconds after which the profiler endpoint expires.
	// Defaults to 7200 seconds. Minimum value is 1.
	// +optional
	ExpirationSeconds int32 `json:"expirationSeconds,omitempty"`
}

// GoProfilerSpec specifies the action to perform for profiling a go program.
type GoProfilerSpec struct {
	// Source specifies the profile source. It must be a local file path or a URL.
	Source string `json:"source"`
}

// JavaProfilerSpec specifies the action to perform for profiling a java program.
type JavaProfilerSpec struct {
	// Type is the type of the java profiler. There are two possible type values:
	//
	// Arthas: The profiler will be run by arthas.
	// MemoryAnalyzer: The profiler will be run by eclipse memory analyzer.
	Type JavaProfilerType `json:"type"`
	// HPROFFilePath is the path of hprof file. It must be an absolute path on node.
	// +optional
	HPROFFilePath string `json:"hprofFilePath,omitempty"`
}

// AbnormalStatus defines the observed state of Abnormal.
type AbnormalStatus struct {
	// Identifiable indicates if the abnormal could be identified by the diagnoser chain.
	Identifiable bool `json:"identifiable"`
	// Recoverable indicates if the abnormal could be recovered by the recoverer chain.
	Recoverable bool `json:"recoverable"`
	// Phase is a simple, high-level summary of where the abnormal is in its lifecycle.
	// The conditions array, the reason and message fields contain more detail about the
	// pod's status.
	// There are six possible phase values:
	//
	// InformationCollecting: The abnormal has been passed to information manager and some of the
	// information collectors have been started. At least one information collector is still running.
	// Diagnosing: The abnormal has been passed to diagnoser chain and some of the diagnosers
	// have been started. At least one diagnoser is still running.
	// Recovering: The abnormal has been passed to recoverer chain and some of the recoverers
	// have been started. At least one recoverer is still running.
	// Succeeded: The abnormal has been successfully recovered by some of the recoverers.
	// Failed: All diagnosers and recoverers have been executed, and none of diagnosers and
	// recoverers is able to diagnose and recover the abnormal.
	// Unknown: For some reason the state of the abnormal could not be obtained.
	// +optional
	Phase AbnormalPhase `json:"phase,omitempty"`
	// Conditions contains current service state of abnormal.
	// +optional
	Conditions []AbnormalCondition `json:"conditions,omitempty"`
	// Message is a human readable message indicating details about why the abnormal is in
	// this condition.
	// +optional
	Message string `json:"message,omitempty"`
	// Reason is a brief CamelCase message indicating details about why the abnormal is in
	// this state.
	// +optional
	Reason string `json:"reason,omitempty"`
	// StartTime is RFC 3339 date and time at which the object was acknowledged by the system.
	// +optional
	StartTime metav1.Time `json:"startTime,omitempty"`
	// Diagnoser indicates the diagnoser which has identified the abnormal successfully.
	// +optional
	Diagnoser *NamespacedName `json:"diagnoser,omitempty"`
	// Recoverer indicates the recoverer which has recovered the abnormal successfully.
	// +optional
	Recoverer *NamespacedName `json:"recoverer,omitempty"`
	// CommandExecutors is the list of command execution results.
	// +optional
	CommandExecutors []CommandExecutorStatus `json:"commandExecutors,omitempty"`
	// Profilers is the list of profiler status.
	// +optional
	Profilers []ProfilerStatus `json:"profilers,omitempty"`
	// Context is a blob of information about the abnormal, meant to be user-facing
	// content and display instructions. This field may contain customized values for
	// custom source.
	// +optional
	Context *runtime.RawExtension `json:"context,omitempty"`
}

// CommandExecutorStatus is the command execution result.
type CommandExecutorStatus struct {
	// Command represents a command being prepared and run.
	Command []string `json:"command"`
	// Type is the type of the command executor. There are three possible type values:
	//
	// InformationCollector: The command executor will be run by information manager.
	// Diagnoser: The command executor will be run by diagnoser chain.
	// Recoverer: The command executor will be run by recoverer chain.
	Type AbnormalProcessorType `json:"type"`
	// Stdout is standard output of the command.
	// +optional
	Stdout string `json:"stdout,omitempty"`
	// Stderr is standard error of the command.
	// +optional
	Stderr string `json:"stderr,omitempty"`
	// Error is the command execution error.
	// +optional
	Error string `json:"error,omitempty"`
}

// ProfilerStatus is the profiler status.
type ProfilerStatus struct {
	// Name specifies the name of a profiler.
	Name string `json:"name"`
	// Type is the type of the profiler. There are three possible type values:
	//
	// InformationCollector: The profiler will be run by information manager.
	// Diagnoser: The profiler will be run by diagnoser chain.
	// Recoverer: The profiler will be run by recoverer chain.
	Type AbnormalProcessorType `json:"type"`
	// One and only one of the following programming languages should be specified.
	// Go is the result of go profiler.
	// +optional
	Go *GoProfilerStatus `json:"go,omitempty"`
	// Java is the status of java profiler.
	// +optional
	Java *JavaProfilerStatus `json:"java,omitempty"`
	// Expired indicates if the profiler endpoint has expired.
	// +optional
	Expired bool `json:"expired,omitempty"`
	// Error is the profiler error.
	// +optional
	Error string `json:"error,omitempty"`
}

// GoProfilerStatus is the result of go profiler.
type GoProfilerStatus struct {
	// Endpoint specifies how to navigate through a performance profile.
	Endpoint string `json:"endpoint"`
}

// JavaProfilerStatus is the status of java profiler.
type JavaProfilerStatus struct {
	// Type is the type of the java profiler. There are two possible type values:
	//
	// Arthas: The profiler will be run by arthas.
	// MemoryAnalyzer: The profiler will be run by eclipse memory analyzer.
	Type JavaProfilerType `json:"type"`
	// One and only one of the following java profiler should be specified.
	// Arthas is the result of arthas java profiler.
	// +optional
	Arthas *ArthasProfilerStatus `json:"arthas,omitempty"`
	// MemoryAnalyzer is the result of eclipse memory analyzer java profiler.
	// +optional
	MemoryAnalyzer *MemoryAnalyzerProfilerStatus `json:"memoryAnalyzer,omitempty"`
}

// ArthasProfilerStatus is the result of arthas java profiler.
type ArthasProfilerStatus struct {
	// Endpoint specifies how to navigate through web console of arthas.
	Endpoint string `json:"endpoint"`
}

// MemoryAnalyzerProfilerStatus is the result of eclipse memory analyzer java profiler.
type MemoryAnalyzerProfilerStatus struct {
	// Endpoint specifies how to navigate through web of eclipse memory analyzer results.
	Endpoint string `json:"endpoint"`
}

// AbnormalCondition contains details for the current condition of this abnormal.
type AbnormalCondition struct {
	// Type is the type of the condition.
	Type AbnormalConditionType `json:"type"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// LastTransitionTime specifies last time the condition transitioned from one status
	// to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason is a unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message is a human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// AbnormalPhase is a label for the condition of a abnormal at the current time.
type AbnormalPhase string

// AbnormalProcessorType is a valid for CommandExecutor.Type.
type AbnormalProcessorType string

// AbnormalConditionType is a valid value for AbnormalCondition.Type.
type AbnormalConditionType string

// JavaProfilerType is a valid value for JavaProfiler.Type.
type JavaProfilerType string

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Abnormal is the Schema for the abnormals API.
type Abnormal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AbnormalSpec   `json:"spec,omitempty"`
	Status AbnormalStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AbnormalList contains a list of Abnormal.
type AbnormalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Abnormal `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Abnormal{}, &AbnormalList{})
}
