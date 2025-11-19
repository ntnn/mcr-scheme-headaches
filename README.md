# mcr-scheme-headaches

Create two kind clusters:

    kind create cluster --name mcr-scheme-headaches-source

    kind create cluster --name mcr-scheme-headaches-target

Export the kubeconfigs:

    kind export kubeconfig --name mcr-scheme-headaches-source --kubeconfig source.kubeconfig

    kind export kubeconfig --name mcr-scheme-headaches-target --kubeconfig target.kubeconfig

Install the Source CRD in the source cluster:

    kubectl --kubeconfig source.kubeconfig apply -f ./config/crd/bases/example.mcr-scheme-headaches.ntnn.github.com_sources.yaml

And the Target CRD in the target cluster:

    kubectl --kubeconfig target.kubeconfig apply -f ./config/crd/bases/example.mcr-scheme-headaches.ntnn.github.com_targets.yaml

And start the manager:

    go run ./cmd/manager --source-kubeconfig source.kubeconfig --target-kubeconfig target.kubeconfig

We'd expect the manager to engage both clusters and create a Target CR
in the target cluster based on the Source CR in the source cluster.

However currently mcr does not support reconciling against multiple
clusters that have differing schemes and fails to engage the
`cluster.Cluster`:

```
2025-11-19T14:21:32+01:00       ERROR   clusters-cluster-provider       error adding cluster    {"name": "target", "error": "failed to engage cluster \"target\": failed to watch for cluster \"target\": no kind is registered for the type v1alpha1.Source in scheme \"pkg/runtime/scheme.go:110\""}
sigs.k8s.io/multicluster-runtime/providers/clusters.(*Provider).Start
        /Users/I567861/SAPDevelop/golang/pkg/mod/sigs.k8s.io/multicluster-runtime@v0.22.0-beta.0.0.20251119125600-21deeffb172b/providers/clusters/provider.go:89
sigs.k8s.io/multicluster-runtime/pkg/manager.(*mcManager).Start.func1
        /Users/I567861/SAPDevelop/golang/pkg/mod/sigs.k8s.io/multicluster-runtime@v0.22.0-beta.0.0.20251119125600-21deeffb172b/pkg/manager/manager.go:256
sigs.k8s.io/controller-runtime/pkg/manager.RunnableFunc.Start
        /Users/I567861/SAPDevelop/golang/pkg/mod/sigs.k8s.io/controller-runtime@v0.22.4/pkg/manager/manager.go:312
sigs.k8s.io/controller-runtime/pkg/manager.(*runnableGroup).reconcile.func1
        /Users/I567861/SAPDevelop/golang/pkg/mod/sigs.k8s.io/controller-runtime@v0.22.4/pkg/manager/runnable_group.go:260
```

So when creating the Source CR in the source cluster, the manager
doesn't reconcile it:

    kubectl --kubeconfig source.kubeconfig apply -f ./config/samples/example_v1alpha1_source.yaml

No Target CR is created in the target cluster:

    kubectl --kubeconfig target.kubeconfig get targets.example.mcr-scheme-headaches.ntnn.github.com

And the conditions in the source cluster are not updated:

    kubectl --kubeconfig source.kubeconfig get sources.example.mcr-scheme-headaches.ntnn.github.com source-sample -o yaml

## Solution

### ForCluster

One option would be to allow filtering which clusters a reconciler
enganges, e.g. by adding a `ForCluster` method to the builder:

```go
	if err := mcbuilder.ControllerManagedBy(mgr).
			Named("source-controller").
			ForCluster(sourceClusterFilter).
			For(&headachev1alpha1.Source{}).
			Complete(sr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Source")
		os.Exit(1)
	}
```

The `ForCluster` could take a function like this:

```go
type ForClusterFilter func(context.Context, cluster.Cluster) bool

func (mcbuilder *Builder) ForCluster(filter ForClusterFilter) *Builder {
    // ...
}
```

### ForProvider

`ForProvider` could work like `ForCluster`, but take a provider
reference. If the provider has the cluster the reconciler should engage,
it would engage it.

```go
func (mcbuilder *Builder) ForProvider(provider multicluster.Provider) *Builder {
    // ...
}
```
