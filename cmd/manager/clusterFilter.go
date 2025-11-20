package main

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type clusterFilter predicate.Predicate

type ClusterFilter struct {
	ClusterName string
}

func (cf *ClusterFilter) Create(event.TypedCreateEvent[client.Object]) bool {
	return false
}

func (cf *ClusterFilter) Delete(event.TypedDeleteEvent[client.Object]) bool {
	return false
}

func (cf *ClusterFilter) Update(event.TypedUpdateEvent[client.Object]) bool {
	return false
}

func (cf *ClusterFilter) Generic(event.TypedGenericEvent[client.Object]) bool {
	return false
}
