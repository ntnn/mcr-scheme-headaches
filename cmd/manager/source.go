package main

import (
	"context"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	mctrl "sigs.k8s.io/multicluster-runtime"

	headachev1alpha1 "github.com/ntnn/mcr-scheme-headaches/api/v1alpha1"
)

type SourceReconciler struct {
	Log        logr.Logger
	GetCluster func(ctx context.Context, clusterName string) (cluster.Cluster, error)
}

func (sr *SourceReconciler) Reconcile(ctx context.Context, req mctrl.Request) (mctrl.Result, error) {
	log := sr.Log.WithValues("cluster", req.ClusterName, "namespace", req.Namespace, "name", req.Name)
	cl, err := sr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		log.Error(err, "failed to get cluster")
		return mctrl.Result{}, err
	}

	source := headachev1alpha1.Source{}
	if err := cl.GetClient().Get(ctx, req.NamespacedName, &source); err != nil {
		log.Error(err, "failed to get Source")
		return mctrl.Result{}, err
	}

	source.Status.Conditions = append(source.Status.Conditions, metav1.Condition{
		Type:    "Reconciling",
		Status:  metav1.ConditionTrue,
		Reason:  "ReconcilingSource",
		Message: "Reconciling Source resource",
	})
	if err := cl.GetClient().Status().Update(ctx, &source); err != nil {
		log.Error(err, "failed to update Source status")
		return mctrl.Result{}, err
	}

	log.Info("Reconciling Source", "spec", source.Spec)

	targetCl, err := sr.GetCluster(ctx, "target")
	if err != nil {
		log.Error(err, "failed to get target cluster")
		return mctrl.Result{}, err
	}

	target := headachev1alpha1.Target{}
	if err := targetCl.GetClient().Get(ctx, req.NamespacedName, &target); err != nil {
		if apierrors.IsNotFound(err) {
			target = headachev1alpha1.Target{
				ObjectMeta: metav1.ObjectMeta{
					Name:      source.Name,
					Namespace: source.Namespace,
				},
				Status: headachev1alpha1.TargetStatus{
					Message:        source.Spec.Message,
					AdditionalInfo: source.Spec.AdditionalInfo,
				},
			}
			if err := targetCl.GetClient().Create(ctx, &target); err != nil {
				log.Error(err, "failed to create Target")
				return mctrl.Result{}, err
			}
			log.Info("Created Target", "name", target.Name, "namespace", target.Namespace)
			source.Status.Conditions = append(source.Status.Conditions, metav1.Condition{
				Type:    "CreatedTarget",
				Status:  metav1.ConditionTrue,
				Reason:  "TargetCreated",
				Message: "Target resource has been created in target cluster",
			})
			if err := cl.GetClient().Status().Update(ctx, &source); err != nil {
				log.Error(err, "failed to update Source status")
				return mctrl.Result{}, err
			}
			return mctrl.Result{}, nil
		}
		log.Error(err, "failed to get Target")
		return mctrl.Result{}, err
	}

	target.Status.Message = source.Spec.Message
	target.Status.AdditionalInfo = source.Spec.AdditionalInfo
	if err := targetCl.GetClient().Status().Update(ctx, &target); err != nil {
		log.Error(err, "failed to update Target status")
		return mctrl.Result{}, err
	}

	source.Status.Conditions = append(source.Status.Conditions, metav1.Condition{
		Type:    "UpdatedTarget",
		Status:  metav1.ConditionTrue,
		Reason:  "TargetUpdated",
		Message: "Target resource has been updated in target cluster",
	})
	if err := cl.GetClient().Status().Update(ctx, &source); err != nil {
		log.Error(err, "failed to update Source status")
		return mctrl.Result{}, err
	}

	return mctrl.Result{}, nil
}
