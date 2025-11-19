/*
Copyright 2025.
SPDX-License-Identifier: Apache-2.0

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

// The main package running the resource-broker.
package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
	mctrl "sigs.k8s.io/multicluster-runtime"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	"sigs.k8s.io/multicluster-runtime/providers/clusters"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	headachev1alpha1 "github.com/ntnn/mcr-scheme-headaches/api/v1alpha1"
)

var (
	setupLog = ctrl.Log.WithName("setup")

	fSourceKubeconfig = flag.String("source-kubeconfig", "source.kubeconfig", "source kubeconfig")
	fTargetKubeconfig = flag.String("target-kubeconfig", "target.kubeconfig", "target kubeconfig")
)

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ctx := mctrl.SetupSignalHandler()

	_, sourceCluster, err := setupCluster(
		*fSourceKubeconfig,
		&headachev1alpha1.Source{},
		&headachev1alpha1.SourceList{},
	)

	targetConfig, targetCluster, err := setupCluster(
		*fTargetKubeconfig,
		&headachev1alpha1.Target{},
		&headachev1alpha1.TargetList{},
	)

	provider := clusters.New()
	if err := provider.Add(ctx, "source", sourceCluster); err != nil {
		setupLog.Error(err, "unable to add source cluster to provider")
		os.Exit(1)
	}
	if err := provider.Add(ctx, "target", targetCluster); err != nil {
		setupLog.Error(err, "unable to add target cluster to provider")
		os.Exit(1)
	}

	// Add both source and target CRDs to the managers scheme
	scheme := runtime.NewScheme()
	if err := headachev1alpha1.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "unable to add Headachev1alpha1 to scheme")
		os.Exit(1)
	}

	mgr, err := mctrl.NewManager(
		targetConfig, // doesn't matter but manager needs a config
		provider,
		mctrl.Options{
			Scheme: scheme,
		},
	)
	if err != nil {
		setupLog.Error(err, "unable to start multicluster manager")
		os.Exit(1)
	}

	log := ctrl.Log.WithName("headache")

	sr := &SourceReconciler{
		Log:        log.WithName("controllers").WithName("Source"),
		GetCluster: mgr.GetCluster,
	}

	// sourceObj := &unstructured.Unstructured{}
	// sourceObj.SetGroupVersionKind(schema.GroupVersionKind{
	// 	Group:   "example.mcr-scheme-headaches.ntnn.github.com",
	// 	Kind:    "Source",
	// 	Version: "v1alpha1",
	// })
	// if err := mcbuilder.ControllerManagedBy(mgr).
	// 	Named("source-controller").
	// 	For(sourceObj).
	// 	Complete(sr); err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "Source")
	// 	os.Exit(1)
	// }

	if err := mcbuilder.ControllerManagedBy(mgr).
		Named("source-controller").
		For(&headachev1alpha1.Source{}).
		Complete(sr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Source")
		os.Exit(1)
	}

	tr := &TargetReconciler{
		Log:        log.WithName("controllers").WithName("Target"),
		GetCluster: mgr.GetCluster,
	}

	if err := mcbuilder.ControllerManagedBy(mgr).
		Named("target-controller").
		For(&headachev1alpha1.Target{}).
		Complete(tr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Target")
		os.Exit(1)
	}

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "error in manager")
		os.Exit(1)
	}
}

func getConfig(path string) (*rest.Config, error) {
	rawConfig, err := clientcmd.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to load kubeconfig from %q: %w", path, err)
	}

	config, err := clientcmd.NewNonInteractiveClientConfig(
		*rawConfig,
		rawConfig.CurrentContext,
		&clientcmd.ConfigOverrides{},
		nil,
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to create rest config from kubeconfig %q: %w", path, err)
	}
	return config, nil
}

func setupCluster(configPath string, objects ...runtime.Object) (*rest.Config, cluster.Cluster, error) {
	schemeBuilder := &scheme.Builder{GroupVersion: headachev1alpha1.GroupVersion}
	schemeBuilder.Register(objects...)

	scheme := runtime.NewScheme()
	if err := schemeBuilder.AddToScheme(scheme); err != nil {
		return nil, nil, fmt.Errorf("error setting up scheme: %w", err)
	}

	config, err := getConfig(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting kubeconfig %q: %w", configPath, err)
	}

	cl, err := cluster.New(
		config,
		func(o *cluster.Options) {
			o.Scheme = scheme
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create cluster: %w", err)
	}

	return config, cl, nil
}
