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

package util

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	diagnosisv1 "netease.com/k8s/kube-diagnoser/api/v1"
)

func TestUpdateAbnormalCondition(t *testing.T) {
	abnormalStatus := diagnosisv1.AbnormalStatus{
		Conditions: []diagnosisv1.AbnormalCondition{
			{
				Type:    diagnosisv1.InformationCollected,
				Status:  corev1.ConditionTrue,
				Reason:  "successfully",
				Message: "sync abnormal successfully",
			},
		},
	}

	tests := []struct {
		status    *diagnosisv1.AbnormalStatus
		condition diagnosisv1.AbnormalCondition
		expected  bool
		desc      string
	}{
		{
			status: &abnormalStatus,
			condition: diagnosisv1.AbnormalCondition{
				Type:    diagnosisv1.InformationCollected,
				Status:  corev1.ConditionTrue,
				Reason:  "successfully",
				Message: "sync abnormal successfully",
			},
			expected: false,
			desc:     "all equal, no update",
		},
		{
			status: &abnormalStatus,
			condition: diagnosisv1.AbnormalCondition{
				Type:    diagnosisv1.AbnormalIdentified,
				Status:  corev1.ConditionTrue,
				Reason:  "successfully",
				Message: "sync abnormal successfully",
			},
			expected: true,
			desc:     "not equal Type, should get updated",
		},
		{
			status: &abnormalStatus,
			condition: diagnosisv1.AbnormalCondition{
				Type:    diagnosisv1.InformationCollected,
				Status:  corev1.ConditionFalse,
				Reason:  "successfully",
				Message: "sync abnormal successfully",
			},
			expected: true,
			desc:     "not equal Status, should get updated",
		},
	}

	for _, test := range tests {
		resultStatus := UpdateAbnormalCondition(test.status, &test.condition)
		assert.Equal(t, test.expected, resultStatus, test.desc)
	}
}

func TestGetAbnormalCondition(t *testing.T) {
	type expectedStruct struct {
		index     int
		condition *diagnosisv1.AbnormalCondition
	}

	tests := []struct {
		status   *diagnosisv1.AbnormalStatus
		condType diagnosisv1.AbnormalConditionType
		expected expectedStruct
		desc     string
	}{
		{
			status:   nil,
			condType: diagnosisv1.InformationCollected,
			expected: expectedStruct{-1, nil},
			desc:     "status nil, not found",
		},
		{
			status: &diagnosisv1.AbnormalStatus{
				Conditions: nil,
			},
			condType: diagnosisv1.InformationCollected,
			expected: expectedStruct{-1, nil},
			desc:     "conditions nil, not found",
		},
		{
			status: &diagnosisv1.AbnormalStatus{
				Conditions: []diagnosisv1.AbnormalCondition{
					{
						Type:    diagnosisv1.InformationCollected,
						Status:  corev1.ConditionTrue,
						Reason:  "successfully",
						Message: "sync abnormal successfully",
					},
				},
			},
			condType: diagnosisv1.InformationCollected,
			expected: expectedStruct{0, &diagnosisv1.AbnormalCondition{
				Type:    diagnosisv1.InformationCollected,
				Status:  corev1.ConditionTrue,
				Reason:  "successfully",
				Message: "sync abnormal successfully"},
			},
			desc: "condition found",
		},
	}

	for _, test := range tests {
		resultIndex, resultCond := GetAbnormalCondition(test.status, test.condType)
		assert.Equal(t, test.expected.index, resultIndex, test.desc)
		assert.Equal(t, test.expected.condition, resultCond, test.desc)
	}
}

func TestFormatURL(t *testing.T) {
	tests := []struct {
		scheme   string
		host     string
		port     string
		path     string
		expected *url.URL
		desc     string
	}{
		{
			scheme: "http",
			host:   "127.0.0.1",
			port:   "8080",
			path:   "/test",
			expected: &url.URL{
				Scheme: "http",
				Host:   "127.0.0.1:8080",
				Path:   "/test",
			},
			desc: "regular url",
		},
	}

	for _, test := range tests {
		resultURL := FormatURL(test.scheme, test.host, test.port, test.path)
		assert.Equal(t, test.expected, resultURL, test.desc)
	}
}

func TestValidateAbnormalResult(t *testing.T) {
	time := time.Now()
	abnormal := diagnosisv1.Abnormal{
		Spec: diagnosisv1.AbnormalSpec{
			NodeName: "node1",
		},
		Status: diagnosisv1.AbnormalStatus{
			Identifiable: false,
			Recoverable:  false,
			Phase:        diagnosisv1.AbnormalDiagnosing,
			Conditions: []diagnosisv1.AbnormalCondition{
				{
					Type:    diagnosisv1.InformationCollected,
					Status:  corev1.ConditionTrue,
					Reason:  "successfully",
					Message: "sync abnormal successfully",
				},
			},
			StartTime: metav1.NewTime(time),
			Diagnoser: &diagnosisv1.NamespacedName{
				Namespace: "default",
				Name:      "diagnoser1",
			},
		},
	}

	invalidSpec := abnormal
	invalidSpec.Spec.NodeName = "node2"

	invalidIdentifiable := abnormal
	invalidIdentifiable.Status.Identifiable = true

	invalidRecoverable := abnormal
	invalidRecoverable.Status.Recoverable = true

	invalidPhase := abnormal
	invalidPhase.Status.Phase = diagnosisv1.AbnormalFailed

	invalidConditions := abnormal
	invalidConditions.Status.Conditions = []diagnosisv1.AbnormalCondition{}

	invalidMessage := abnormal
	invalidMessage.Status.Message = "message"

	invalidReason := abnormal
	invalidReason.Status.Reason = "reason"

	invalidStartTime := abnormal
	invalidStartTime.Status.StartTime = metav1.NewTime(time.Add(1000))

	invalidDiagnoser := abnormal
	invalidDiagnoser.Status.Diagnoser = &diagnosisv1.NamespacedName{
		Namespace: "default",
		Name:      "diagnoser2",
	}

	invalidRecoverer := abnormal
	invalidRecoverer.Status.Recoverer = &diagnosisv1.NamespacedName{
		Namespace: "default",
		Name:      "recoverer1",
	}

	valid := abnormal
	valid.Status.Context = &runtime.RawExtension{
		Raw: []byte("test"),
	}

	tests := []struct {
		result   diagnosisv1.Abnormal
		current  diagnosisv1.Abnormal
		expected error
		desc     string
	}{
		{
			current:  diagnosisv1.Abnormal{},
			result:   diagnosisv1.Abnormal{},
			expected: nil,
			desc:     "empty abnormal",
		},
		{
			current:  abnormal,
			result:   abnormal,
			expected: nil,
			desc:     "no change",
		},
		{
			current:  abnormal,
			result:   valid,
			expected: nil,
			desc:     "valid abnormal",
		},
		{
			current:  abnormal,
			result:   invalidSpec,
			expected: fmt.Errorf("spec field of Abnormal must not be modified"),
			desc:     "invalid spec field",
		},
		{
			current:  abnormal,
			result:   invalidIdentifiable,
			expected: fmt.Errorf("identifiable filed of Abnormal must not be modified"),
			desc:     "invalid identifiable field",
		},
		{
			current:  abnormal,
			result:   invalidRecoverable,
			expected: fmt.Errorf("recoverable filed of Abnormal must not be modified"),
			desc:     "invalid recoverable field",
		},
		{
			current:  abnormal,
			result:   invalidPhase,
			expected: fmt.Errorf("phase filed of Abnormal must not be modified"),
			desc:     "invalid phase field",
		},
		{
			current:  abnormal,
			result:   invalidConditions,
			expected: fmt.Errorf("conditions filed of Abnormal must not be modified"),
			desc:     "invalid conditions field",
		},
		{
			current:  abnormal,
			result:   invalidMessage,
			expected: fmt.Errorf("message filed of Abnormal must not be modified"),
			desc:     "invalid message field",
		},
		{
			current:  abnormal,
			result:   invalidReason,
			expected: fmt.Errorf("reason filed of Abnormal must not be modified"),
			desc:     "invalid reason field",
		},
		{
			current:  abnormal,
			result:   invalidStartTime,
			expected: fmt.Errorf("startTime filed of Abnormal must not be modified"),
			desc:     "invalid startTime field",
		},
		{
			current:  abnormal,
			result:   invalidDiagnoser,
			expected: fmt.Errorf("diagnoser filed of Abnormal must not be modified"),
			desc:     "invalid diagnoser field",
		},
		{
			current:  abnormal,
			result:   invalidRecoverer,
			expected: fmt.Errorf("recoverer filed of Abnormal must not be modified"),
			desc:     "invalid recoverer field",
		},
	}

	for _, test := range tests {
		err := ValidateAbnormalResult(test.result, test.current)
		if test.expected == nil {
			assert.NoError(t, err, test.desc)
		} else {
			assert.EqualError(t, err, test.expected.Error(), test.desc)
		}
	}
}

func TestIsAbnormalNodeNameMatched(t *testing.T) {
	tests := []struct {
		abnormal diagnosisv1.Abnormal
		node     string
		expected bool
		desc     string
	}{
		{
			abnormal: diagnosisv1.Abnormal{
				Spec: diagnosisv1.AbnormalSpec{
					NodeName: "",
				},
			},
			node:     "node1",
			expected: true,
			desc:     "empty node name",
		},
		{
			abnormal: diagnosisv1.Abnormal{
				Spec: diagnosisv1.AbnormalSpec{
					NodeName: "node1",
				},
			},
			node:     "node1",
			expected: true,
			desc:     "node name matched",
		},
	}

	for _, test := range tests {
		matched := IsAbnormalNodeNameMatched(test.abnormal, test.node)
		assert.Equal(t, test.expected, matched, test.desc)
	}
}

func TestSetAbnormalContext(t *testing.T) {
	type expectedStruct struct {
		abnormal diagnosisv1.Abnormal
		err      error
	}

	tests := []struct {
		abnormal diagnosisv1.Abnormal
		key      string
		value    interface{}
		expected expectedStruct
		desc     string
	}{
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: nil,
				},
			},
			key:   "key1",
			value: "value1",
			expected: expectedStruct{
				abnormal: diagnosisv1.Abnormal{
					Status: diagnosisv1.AbnormalStatus{
						Context: &runtime.RawExtension{
							Raw: func(keysAndValues ...string) []byte {
								testingMap, err := newTestingMap(keysAndValues...)
								if err != nil {
									t.Errorf("%v", err)
								}
								return testingMap
							}("key1", "value1"),
						},
					},
				},
				err: nil,
			},
			desc: "nil context",
		},
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: &runtime.RawExtension{},
				},
			},
			key:   "key1",
			value: "value1",
			expected: expectedStruct{
				abnormal: diagnosisv1.Abnormal{
					Status: diagnosisv1.AbnormalStatus{
						Context: &runtime.RawExtension{
							Raw: func(keysAndValues ...string) []byte {
								testingMap, err := newTestingMap(keysAndValues...)
								if err != nil {
									t.Errorf("%v", err)
								}
								return testingMap
							}("key1", "value1"),
						},
					},
				},
				err: nil,
			},
			desc: "empty context",
		},
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: &runtime.RawExtension{
						Raw: func(keysAndValues ...string) []byte {
							testingMap, err := newTestingMap(keysAndValues...)
							if err != nil {
								t.Errorf("%v", err)
							}
							return testingMap
						}("key1", "value1", "key2", "value2"),
					},
				},
			},
			key:   "key3",
			value: "value3",
			expected: expectedStruct{
				abnormal: diagnosisv1.Abnormal{
					Status: diagnosisv1.AbnormalStatus{
						Context: &runtime.RawExtension{
							Raw: func(keysAndValues ...string) []byte {
								testingMap, err := newTestingMap(keysAndValues...)
								if err != nil {
									t.Errorf("%v", err)
								}
								return testingMap
							}("key1", "value1", "key2", "value2", "key3", "value3"),
						},
					},
				},
				err: nil,
			},
			desc: "context updated",
		},
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: &runtime.RawExtension{
						Raw: func(keysAndValues ...string) []byte {
							testingMap, err := newTestingMap(keysAndValues...)
							if err != nil {
								t.Errorf("%v", err)
							}
							return testingMap
						}("key1", "value1", "key2", "value2"),
					},
				},
			},
			key:   "key2",
			value: "value3",
			expected: expectedStruct{
				abnormal: diagnosisv1.Abnormal{
					Status: diagnosisv1.AbnormalStatus{
						Context: &runtime.RawExtension{
							Raw: func(keysAndValues ...string) []byte {
								testingMap, err := newTestingMap(keysAndValues...)
								if err != nil {
									t.Errorf("%v", err)
								}
								return testingMap
							}("key1", "value1", "key2", "value3"),
						},
					},
				},
				err: nil,
			},
			desc: "context overwritten",
		},
	}

	for _, test := range tests {
		abnormal, err := SetAbnormalContext(test.abnormal, test.key, test.value)
		assert.Equal(t, test.expected.abnormal, abnormal, test.desc)
		if test.expected.err == nil {
			assert.NoError(t, err, test.desc)
		} else {
			assert.EqualError(t, err, test.expected.err.Error(), test.desc)
		}
	}
}

func TestGetAbnormalContext(t *testing.T) {
	type expectedStruct struct {
		value []byte
		err   error
	}

	tests := []struct {
		abnormal diagnosisv1.Abnormal
		key      string
		expected expectedStruct
		desc     string
	}{
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: nil,
				},
			},
			key: "key1",
			expected: expectedStruct{
				value: nil,
				err:   fmt.Errorf("abnormal context nil"),
			},
			desc: "nil context",
		},
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: &runtime.RawExtension{},
				},
			},
			key: "key1",
			expected: expectedStruct{
				value: nil,
				err:   fmt.Errorf("abnormal context empty"),
			},
			desc: "empty context",
		},
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: &runtime.RawExtension{
						Raw: func(keysAndValues ...string) []byte {
							testingMap, err := newTestingMap(keysAndValues...)
							if err != nil {
								t.Errorf("%v", err)
							}
							return testingMap
						}("key1", "value1", "key2", "value2"),
					},
				},
			},
			key: "key2",
			expected: expectedStruct{
				value: func(str string) []byte {
					result, err := json.Marshal(str)
					if err != nil {
						t.Errorf("%v", err)
					}
					return result
				}("value2"),
				err: nil,
			},
			desc: "context found",
		},
	}

	for _, test := range tests {
		abnormal, err := GetAbnormalContext(test.abnormal, test.key)
		assert.Equal(t, test.expected.value, abnormal, test.desc)
		if test.expected.err == nil {
			assert.NoError(t, err, test.desc)
		} else {
			assert.EqualError(t, err, test.expected.err.Error(), test.desc)
		}
	}
}

func TestRemoveAbnormalContext(t *testing.T) {
	type expectedStruct struct {
		abnormal diagnosisv1.Abnormal
		removed  bool
		err      error
	}

	tests := []struct {
		abnormal diagnosisv1.Abnormal
		key      string
		expected expectedStruct
		desc     string
	}{
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: nil,
				},
			},
			key: "key1",
			expected: expectedStruct{
				abnormal: diagnosisv1.Abnormal{
					Status: diagnosisv1.AbnormalStatus{
						Context: nil,
					},
				},
				removed: true,
				err:     nil,
			},
			desc: "nil context",
		},
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: &runtime.RawExtension{},
				},
			},
			key: "key1",
			expected: expectedStruct{
				abnormal: diagnosisv1.Abnormal{
					Status: diagnosisv1.AbnormalStatus{
						Context: &runtime.RawExtension{},
					},
				},
				removed: true,
				err:     nil,
			},
			desc: "empty context",
		},
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: &runtime.RawExtension{
						Raw: func(keysAndValues ...string) []byte {
							testingMap, err := newTestingMap(keysAndValues...)
							if err != nil {
								t.Errorf("%v", err)
							}
							return testingMap
						}("key1", "value1", "key2", "value2"),
					},
				},
			},
			key: "key2",
			expected: expectedStruct{
				abnormal: diagnosisv1.Abnormal{
					Status: diagnosisv1.AbnormalStatus{
						Context: &runtime.RawExtension{
							Raw: func(keysAndValues ...string) []byte {
								testingMap, err := newTestingMap(keysAndValues...)
								if err != nil {
									t.Errorf("%v", err)
								}
								return testingMap
							}("key1", "value1"),
						},
					},
				},
				removed: true,
				err:     nil,
			},
			desc: "context removed",
		},
		{
			abnormal: diagnosisv1.Abnormal{
				Status: diagnosisv1.AbnormalStatus{
					Context: &runtime.RawExtension{
						Raw: []byte{0, 1, 2},
					},
				},
			},
			key: "key1",
			expected: expectedStruct{
				abnormal: diagnosisv1.Abnormal{
					Status: diagnosisv1.AbnormalStatus{
						Context: &runtime.RawExtension{
							Raw: []byte{0, 1, 2},
						},
					},
				},
				removed: false,
				err:     fmt.Errorf("invalid character '\\x00' looking for beginning of value"),
			},
			desc: "invalid context",
		},
	}

	for _, test := range tests {
		abnormal, removed, err := RemoveAbnormalContext(test.abnormal, test.key)
		assert.Equal(t, test.expected.abnormal, abnormal, test.desc)
		assert.Equal(t, test.expected.removed, removed, test.desc)
		if test.expected.err == nil {
			assert.NoError(t, err, test.desc)
		} else {
			assert.EqualError(t, err, test.expected.err.Error(), test.desc)
		}
	}
}

func TestRetrievePodsOnNode(t *testing.T) {
	tests := []struct {
		pods     []corev1.Pod
		nodeName string
		expected []corev1.Pod
		desc     string
	}{
		{
			pods:     []corev1.Pod{},
			nodeName: "node1",
			expected: []corev1.Pod{},
			desc:     "empty slice",
		},
		{
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod1",
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod2",
					},
					Spec: corev1.PodSpec{
						NodeName: "node2",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod3",
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
				},
			},
			nodeName: "node1",
			expected: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod1",
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod3",
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
				},
			},
			desc: "pods not on provided node removed",
		},
	}

	for _, test := range tests {
		resultPods := RetrievePodsOnNode(test.pods, test.nodeName)
		assert.Equal(t, test.expected, resultPods, test.desc)
	}
}

func newTestingMap(keysAndValues ...string) ([]byte, error) {
	if len(keysAndValues) < 2 || len(keysAndValues)%2 == 1 {
		return nil, fmt.Errorf("invalid input for keys and values: %v", keysAndValues)
	}

	testingMap := make(map[string]interface{})
	for i := 0; i < len(keysAndValues)-1; i = i + 2 {
		testingMap[keysAndValues[i]] = keysAndValues[i+1]
	}

	raw, err := json.Marshal(testingMap)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal testing raw data: %v", err)
	}

	return raw, nil
}