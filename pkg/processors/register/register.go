package register

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kube-diagnoser/kube-diagnoser/pkg/features"
	k8scollector "github.com/kube-diagnoser/kube-diagnoser/pkg/processors/collector/k8s"
	runtimecollector "github.com/kube-diagnoser/kube-diagnoser/pkg/processors/collector/runtime"
	systemcollector "github.com/kube-diagnoser/kube-diagnoser/pkg/processors/collector/system"
	k8sdiagnoser "github.com/kube-diagnoser/kube-diagnoser/pkg/processors/diagnoser/k8s"
	runtimediagnoser "github.com/kube-diagnoser/kube-diagnoser/pkg/processors/diagnoser/runtime"
	executorprocessor "github.com/kube-diagnoser/kube-diagnoser/pkg/processors/executor"
	k8srecover "github.com/kube-diagnoser/kube-diagnoser/pkg/processors/recover/k8s"
)

// RegistryOption contains options of all kinds of Processors, it might be append in the future.
type RegistryOption struct {
	// NodeName specifies the node name.
	NodeName string
	// DockerEndpoint specifies the docker endpoint.
	DockerEndpoint string
	// DataRoot is root directory of persistent kube diagnoser data.
	DataRoot string
	// BindAddress is the address on which to advertise.
	BindAddress string
}

// RegisterProcessors will initialize all processors and add into router to provide HTTP service.
func RegisterProcessors(mgr manager.Manager,
	opts *RegistryOption,
	featureGate features.KubeDiagnoserFeatureGate,
	router *mux.Router,
	setupLog logr.Logger) error {
	// Setup operation processors.
	podListCollector := k8scollector.NewPodListCollector(
		context.Background(),
		ctrl.Log.WithName("processor/podListCollector"),
		mgr.GetCache(),
		opts.NodeName,
		featureGate.Enabled(features.PodCollector),
	)
	podDetailCollector := k8scollector.NewPodDetailCollector(
		context.Background(),
		ctrl.Log.WithName("processor/podDetailCollector"),
		mgr.GetCache(),
		opts.NodeName,
		featureGate.Enabled(features.PodCollector),
	)
	containerCollector, err := k8scollector.NewContainerCollector(
		context.Background(),
		ctrl.Log.WithName("processor/containerCollector"),
		opts.DockerEndpoint,
		featureGate.Enabled(features.ContainerCollector),
	)
	if err != nil {
		setupLog.Error(err, "unable to create processor", "processors", "containerCollector")
		return fmt.Errorf("unable to create processor: %v", err)
	}
	processCollector := systemcollector.NewProcessCollector(
		context.Background(),
		ctrl.Log.WithName("processor/processCollector"),
		featureGate.Enabled(features.ProcessCollector),
	)
	dockerInfoCollector, err := k8scollector.NewDockerInfoCollector(
		context.Background(),
		ctrl.Log.WithName("processor/dockerInfoCollector"),
		opts.DockerEndpoint,
		featureGate.Enabled(features.DockerInfoCollector),
	)
	if err != nil {
		setupLog.Error(err, "unable to create processor", "processors", "dockerInfoCollector")
		return fmt.Errorf("unable to create processor: %v", err)
	}
	dockerdGoroutineCollector := runtimecollector.NewDockerdGoroutineCollector(
		context.Background(),
		ctrl.Log.WithName("processor/dockerdGoroutineCollector"),
		opts.DataRoot,
		featureGate.Enabled(features.DockerdGoroutineCollector),
	)
	containerdGoroutineCollector := runtimecollector.NewContainerdGoroutineCollector(
		context.Background(),
		ctrl.Log.WithName("processor/containerdGoroutineCollector"),
		featureGate.Enabled(features.ContainerdGoroutineCollector),
	)
	mountInfoCollector := systemcollector.NewMountInfoCollector(
		context.Background(),
		ctrl.Log.WithName("processor/mountInfoCollector"),
		featureGate.Enabled(features.MountInfoCollector),
	)

	commandExecutor := executorprocessor.NewCommandExecutor(
		context.Background(),
		ctrl.Log.WithName("processor/commandExecutor"),
		featureGate.Enabled(features.CommandExecutor),
	)
	nodeCordon := k8srecover.NewNodeCordon(
		context.Background(),
		ctrl.Log.WithName("processor/nodeCordon"),
		mgr.GetClient(),
		opts.NodeName,
		featureGate.Enabled(features.NodeCordon),
	)

	goProfiler := runtimediagnoser.NewGoProfiler(
		context.Background(),
		ctrl.Log.WithName("processor/goProfiler"),
		mgr.GetCache(),
		opts.DataRoot,
		opts.BindAddress,
		featureGate.Enabled(features.GoProfiler),
	)
	coreFileProfiler, err := runtimediagnoser.NewCoreFileProfiler(
		context.Background(),
		ctrl.Log.WithName("processor/coreFileProfiler"),
		opts.DockerEndpoint,
		featureGate.Enabled(features.CoreFileProfiler),
		opts.DataRoot)
	if err != nil {
		setupLog.Error(err, "unable to create processor", "processors", "coreFileProfiler")
		return fmt.Errorf("unable to create processor: %v", err)
	}

	subpathRemountDiagnoser := k8sdiagnoser.NewSubPathRemountDiagnoser(
		context.Background(),
		ctrl.Log.WithName("processor/subpathRemountDiagnoser"),
		mgr.GetCache(),
		featureGate.Enabled(features.SubpathRemountDiagnoser),
	)

	subpathRemountRecover := k8srecover.NewSubPathRemountRecover(
		context.Background(),
		ctrl.Log.WithName("processor/subpathRemountRecover"),
		featureGate.Enabled(features.SubpathRemountDiagnoser),
	)

	// Handlers for collecting information.
	router.HandleFunc("/processor/podListCollector", podListCollector.Handler)
	router.HandleFunc("/processor/podDetailCollector", podDetailCollector.Handler)
	router.HandleFunc("/processor/containerCollector", containerCollector.Handler)
	router.HandleFunc("/processor/processCollector", processCollector.Handler)
	router.HandleFunc("/processor/dockerInfoCollector", dockerInfoCollector.Handler)
	router.HandleFunc("/processor/dockerdGoroutineCollector", dockerdGoroutineCollector.Handler)
	router.HandleFunc("/processor/containerdGoroutineCollector", containerdGoroutineCollector.Handler)
	router.HandleFunc("/processor/mountInfoCollector", mountInfoCollector.Handler)
	// Handlers for executing specified command.
	router.HandleFunc("/processor/commandExecutor", commandExecutor.Handler)
	router.HandleFunc("/processor/nodeCordon", nodeCordon.Handler)
	// Handlers for profiling programs.
	router.HandleFunc("/processor/coreFileProfiler", coreFileProfiler.Handler)
	router.HandleFunc("/processor/goProfiler", goProfiler.Handler)

	// Handlers for diagnosing programs
	router.HandleFunc("/processor/subpathRemountDiagnoser", subpathRemountDiagnoser.Handler)

	router.HandleFunc("/processor/subpathRemountRecover", subpathRemountRecover.Handler)
	return nil
}