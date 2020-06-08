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

package recovererchain

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	diagnosisv1 "netease.com/k8s/kube-diagnoser/api/v1"
	"netease.com/k8s/kube-diagnoser/pkg/util"
)

// RecovererChain manages recoverer in the system.
type RecovererChain interface {
	Run() error
	ListAbnormals(ctx context.Context, log logr.Logger) ([]diagnosisv1.Abnormal, error)
	ListRecoverers(ctx context.Context, log logr.Logger) ([]diagnosisv1.Recoverer, error)
	SyncAbnormal(ctx context.Context, log logr.Logger, abnormal diagnosisv1.Abnormal) (diagnosisv1.Abnormal, error)
}

// recovererChainImpl implements RecovererChain interface.
type recovererChainImpl struct {
	// Client knows how to perform CRUD operations on Kubernetes objects.
	client.Client
	// Log represents the ability to log messages.
	Log logr.Logger
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme *runtime.Scheme
	// Cache knows how to load Kubernetes objects.
	Cache cache.Cache

	// Transport for sending http requests to information collectors.
	transport *http.Transport
	// Channel for queuing Abnormals to be processed by recoverer chain.
	recovererChainCh chan diagnosisv1.Abnormal
	// Channel for notifying stop signal.
	stopCh chan struct{}
}

// NewRecovererChain creates a new RecovererChain.
func NewRecovererChain(
	cli client.Client,
	log logr.Logger,
	scheme *runtime.Scheme,
	cache cache.Cache,
	recovererChainCh chan diagnosisv1.Abnormal,
	stopCh chan struct{},
) RecovererChain {
	transport := utilnet.SetTransportDefaults(
		&http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
			Proxy:             http.ProxyURL(nil),
		})

	return &recovererChainImpl{
		Client:           cli,
		Log:              log,
		Scheme:           scheme,
		Cache:            cache,
		transport:        transport,
		recovererChainCh: recovererChainCh,
		stopCh:           stopCh,
	}
}

// Run runs the recoverer chain.
func (rc *recovererChainImpl) Run() error {
	ctx := context.Background()
	log := rc.Log.WithValues("component", "recovererchain")

	// Wait for all caches to sync before processing.
	if !rc.Cache.WaitForCacheSync(rc.stopCh) {
		return fmt.Errorf("falied to sync cache")
	}

	// List abnormals on start.
	abnormals, err := rc.ListAbnormals(ctx, log)
	if err != nil {
		log.Error(err, "failed to list Abnormals")
		return err
	}

	// Sync all abnormals on start.
	for _, abnormal := range abnormals {
		abnormal, err := rc.SyncAbnormal(ctx, log, abnormal)
		if err != nil {
			log.Error(err, "failed to sync Abnormal", "abnormal", abnormal)
		}

		log.Info("syncing Abnormal successfully", "abnormal", client.ObjectKey{
			Name:      abnormal.Name,
			Namespace: abnormal.Namespace,
		})
	}

	// Process abnormals queuing in recoverer chain channel.
	for abnormal := range rc.recovererChainCh {
		abnormal, err := rc.SyncAbnormal(ctx, log, abnormal)
		if err != nil {
			log.Error(err, "failed to sync Abnormal", "abnormal", abnormal)
		}

		log.Info("syncing Abnormal successfully", "abnormal", client.ObjectKey{
			Name:      abnormal.Name,
			Namespace: abnormal.Namespace,
		})
	}

	return nil
}

// ListAbnormals lists Abnormals from cache.
func (rc *recovererChainImpl) ListAbnormals(ctx context.Context, log logr.Logger) ([]diagnosisv1.Abnormal, error) {
	log.Info("listing Abnormals")

	var abnormalList diagnosisv1.AbnormalList
	if err := rc.Cache.List(ctx, &abnormalList); err != nil {
		return nil, err
	}

	return abnormalList.Items, nil
}

// ListRecoverers lists Recoverers from cache.
func (rc *recovererChainImpl) ListRecoverers(ctx context.Context, log logr.Logger) ([]diagnosisv1.Recoverer, error) {
	log.Info("listing Recoverers")

	var recovererList diagnosisv1.RecovererList
	if err := rc.Cache.List(ctx, &recovererList); err != nil {
		return nil, err
	}

	return recovererList.Items, nil
}

// SyncAbnormal syncs abnormals.
func (rc *recovererChainImpl) SyncAbnormal(ctx context.Context, log logr.Logger, abnormal diagnosisv1.Abnormal) (diagnosisv1.Abnormal, error) {
	log.Info("starting to sync Abnormal", "abnormal", client.ObjectKey{
		Name:      abnormal.Name,
		Namespace: abnormal.Namespace,
	})

	switch abnormal.Status.Phase {
	case diagnosisv1.InformationCollecting:
		log.Info("ignoring Abnormal in phase InformationCollecting", "abnormal", client.ObjectKey{
			Name:      abnormal.Name,
			Namespace: abnormal.Namespace,
		})
	case diagnosisv1.AbnormalDiagnosing:
		log.Info("ignoring Abnormal in phase Diagnosing", "abnormal", client.ObjectKey{
			Name:      abnormal.Name,
			Namespace: abnormal.Namespace,
		})
	case diagnosisv1.AbnormalRecovering:
		recoverers, err := rc.ListRecoverers(ctx, log)
		if err != nil {
			log.Error(err, "failed to list Recoverers")
			rc.recovererChainCh <- abnormal
			return abnormal, err
		}

		abnormal, err := rc.RunRecovery(ctx, log, recoverers, abnormal)
		if err != nil {
			log.Error(err, "failed to run recovery")
			rc.recovererChainCh <- abnormal
			return abnormal, err
		}

		return abnormal, nil
	case diagnosisv1.AbnormalSucceeded:
		log.Info("ignoring Abnormal in phase Succeeded", "abnormal", client.ObjectKey{
			Name:      abnormal.Name,
			Namespace: abnormal.Namespace,
		})
	case diagnosisv1.AbnormalFailed:
		log.Info("ignoring Abnormal in phase Failed", "abnormal", client.ObjectKey{
			Name:      abnormal.Name,
			Namespace: abnormal.Namespace,
		})
	case diagnosisv1.AbnormalUnknown:
		log.Info("ignoring Abnormal in phase Unknown", "abnormal", client.ObjectKey{
			Name:      abnormal.Name,
			Namespace: abnormal.Namespace,
		})
	}

	return abnormal, nil
}

// RunRecovery recovers an abnormal with recoverers.
func (rc *recovererChainImpl) RunRecovery(ctx context.Context, log logr.Logger, recoverers []diagnosisv1.Recoverer, abnormal diagnosisv1.Abnormal) (diagnosisv1.Abnormal, error) {
	// Skip recovery if SkipRecovery is true.
	if abnormal.Spec.SkipRecovery {
		log.Info("skipping recovery", "abnormal", client.ObjectKey{
			Name:      abnormal.Name,
			Namespace: abnormal.Namespace,
		})
		abnormal, err := rc.SetAbnormalSucceeded(ctx, log, abnormal)
		if err != nil {
			return abnormal, err
		}

		return abnormal, nil
	}

	for _, recoverer := range recoverers {
		// Execute only matched recoverers if AssignedRecoverers is not empty.
		matched := false
		if len(abnormal.Spec.AssignedRecoverers) == 0 {
			matched = true
		} else {
			for _, assignedRecoverer := range abnormal.Spec.AssignedRecoverers {
				if recoverer.Name == assignedRecoverer.Name && recoverer.Namespace == assignedRecoverer.Namespace {
					log.Info("assigned recoverer matched", "recoverer", client.ObjectKey{
						Name:      recoverer.Name,
						Namespace: recoverer.Namespace,
					})
					matched = true
					break
				}
			}
		}

		if !matched {
			continue
		}

		scheme := strings.ToLower(string(recoverer.Spec.Scheme))
		host := recoverer.Spec.IP
		port := recoverer.Spec.Port
		path := recoverer.Spec.Path
		url := util.FormatURL(scheme, host, port, path)
		timeout := time.Duration(recoverer.Spec.TimeoutSeconds) * time.Second

		cli := &http.Client{
			Timeout:   timeout,
			Transport: rc.transport,
		}

		// Send http request to the recoverers with payload of abnormal.
		result, err := util.DoHTTPRequestWithAbnormal(abnormal, url, *cli, log)
		if err != nil {
			log.Error(err, "failed to do http request to recoverer", recoverer, client.ObjectKey{
				Name:      recoverer.Name,
				Namespace: recoverer.Namespace,
			})
			continue
		}

		// Validate an abnormal after processed by a recoverer.
		err = util.ValidateAbnormalResult(result, abnormal)
		if err != nil {
			log.Error(err, "invalid result from recoverer", recoverer, client.ObjectKey{
				Name:      recoverer.Name,
				Namespace: recoverer.Namespace,
			})
			continue
		}

		abnormal.Status.Recoverer = diagnosisv1.NamespacedName{
			Name:      recoverer.Name,
			Namespace: recoverer.Namespace,
		}
		abnormal, err := rc.SetAbnormalSucceeded(ctx, log, abnormal)
		if err != nil {
			return abnormal, err
		}

		return abnormal, nil
	}

	abnormal, err := rc.SetAbnormalFailed(ctx, log, abnormal)
	if err != nil {
		return abnormal, err
	}

	return abnormal, nil
}

// SetAbnormalFailed sets abnormal phase to Succeeded.
func (rc *recovererChainImpl) SetAbnormalSucceeded(ctx context.Context, log logr.Logger, abnormal diagnosisv1.Abnormal) (diagnosisv1.Abnormal, error) {
	log.Info("setting Abnormal phase to succeeded", "abnormal", client.ObjectKey{
		Name:      abnormal.Name,
		Namespace: abnormal.Namespace,
	})

	abnormal.Status.Phase = diagnosisv1.AbnormalSucceeded
	abnormal.Status.Recoverable = true
	util.UpdateAbnormalCondition(&abnormal.Status, &diagnosisv1.AbnormalCondition{
		Type:   diagnosisv1.AbnormalRecovered,
		Status: corev1.ConditionTrue,
	})
	if err := rc.Status().Update(ctx, &abnormal); err != nil {
		log.Error(err, "unable to update Abnormal")
		return abnormal, err
	}

	return abnormal, nil
}

// SetAbnormalFailed sets abnormal phase to Failed.
func (rc *recovererChainImpl) SetAbnormalFailed(ctx context.Context, log logr.Logger, abnormal diagnosisv1.Abnormal) (diagnosisv1.Abnormal, error) {
	log.Info("setting Abnormal phase to failed", "abnormal", client.ObjectKey{
		Name:      abnormal.Name,
		Namespace: abnormal.Namespace,
	})

	abnormal.Status.Phase = diagnosisv1.AbnormalFailed
	abnormal.Status.Recoverable = false
	util.UpdateAbnormalCondition(&abnormal.Status, &diagnosisv1.AbnormalCondition{
		Type:   diagnosisv1.AbnormalRecovered,
		Status: corev1.ConditionFalse,
	})
	if err := rc.Status().Update(ctx, &abnormal); err != nil {
		log.Error(err, "unable to update Abnormal")
		return abnormal, err
	}

	return abnormal, nil
}
