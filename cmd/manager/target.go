package main

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	mctrl "sigs.k8s.io/multicluster-runtime"

	headachev1alpha1 "github.com/ntnn/mcr-scheme-headaches/api/v1alpha1"
)

type TargetReconciler struct {
	Log        logr.Logger
	GetCluster func(ctx context.Context, clusterName string) (cluster.Cluster, error)
}

func (tr *TargetReconciler) Reconcile(ctx context.Context, req mctrl.Request) (mctrl.Result, error) {
	log := tr.Log.WithValues("cluster", req.ClusterName, "namespace", req.Namespace, "name", req.Name)
	cl, err := tr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		log.Error(err, "failed to get cluster")
		return mctrl.Result{}, err
	}

	target := headachev1alpha1.Target{}
	if err := cl.GetClient().Get(ctx, req.NamespacedName, &target); err != nil {
		log.Error(err, "failed to get Target")
		return mctrl.Result{}, err
	}

	log.Info("Reconciling Target", "spec", target.Spec)
	return mctrl.Result{}, nil
}
