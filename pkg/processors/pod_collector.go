/*
Copyright 2021 The Kube Diagnoser Authors.

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

package processors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/kube-diagnoser/kube-diagnoser/pkg/util"
)

// podCollector manages information of all pods on the node.
type podCollector struct {
	// Context carries values across API boundaries.
	context.Context
	// Logger represents the ability to log messages.
	logr.Logger

	// cache knows how to load Kubernetes objects.
	cache cache.Cache
	// nodeName specifies the node name.
	nodeName string
	// podCollectorEnabled indicates whether podCollector is enabled.
	podCollectorEnabled bool
}

// NewPodCollector creates a new podCollector.
func NewPodCollector(
	ctx context.Context,
	logger logr.Logger,
	cache cache.Cache,
	nodeName string,
	podCollectorEnabled bool,
) Processor {
	return &podCollector{
		Context:             ctx,
		Logger:              logger,
		cache:               cache,
		nodeName:            nodeName,
		podCollectorEnabled: podCollectorEnabled,
	}
}

// Handler handles http requests for pod information.
func (pc *podCollector) Handler(w http.ResponseWriter, r *http.Request) {
	if !pc.podCollectorEnabled {
		http.Error(w, fmt.Sprintf("pod collector is not enabled"), http.StatusUnprocessableEntity)
		return
	}

	switch r.Method {
	case "POST":
		// List all pods on the node.
		pods, err := pc.listPods()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list pods: %v", err), http.StatusInternalServerError)
			return
		}

		data, err := json.Marshal(pods)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to marshal pods: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	default:
		http.Error(w, fmt.Sprintf("method %s is not supported", r.Method), http.StatusMethodNotAllowed)
	}
}

// listPods lists Pods from cache.
func (pc *podCollector) listPods() ([]corev1.Pod, error) {
	pc.Info("listing Pods on node")

	var podList corev1.PodList
	if err := pc.cache.List(pc, &podList); err != nil {
		return nil, err
	}

	podsOnNode := util.RetrievePodsOnNode(podList.Items, pc.nodeName)

	return podsOnNode, nil
}