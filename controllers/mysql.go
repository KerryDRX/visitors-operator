package controllers

import (
	"context"

	examplecomv1beta1 "github.com/ringdrx/visitors-operator/api/v1beta1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func mysqlAuthName() string {
	return "my-secret"
}

func mysqlClusterName() string {
	return "my-cluster"
}

func mysqlStatefulSetName() string {
	return mysqlClusterName() + "-mysql"
}

func mysqlServiceRWName() string {
	return mysqlStatefulSetName() + "-master"
}

func mysqlServiceROName() string {
	return mysqlStatefulSetName()
}

// Returns whether or not the MySQL statefulset is running
func (r *VisitorsAppReconciler) isMysqlUp(ctx context.Context, v *examplecomv1beta1.VisitorsApp) bool {
	log := ctrllog.FromContext(ctx)
	statefulset := &appsv1.StatefulSet{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      mysqlStatefulSetName(),
		Namespace: v.Namespace,
	}, statefulset)

	if err != nil {
		log.Info("StatefulSet mysql not found")
		return false
	}

	if statefulset.Status.ReadyReplicas >= 1 {
		return true
	}

	return false
}
