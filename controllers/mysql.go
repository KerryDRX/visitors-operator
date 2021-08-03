package controllers

import (
	"context"
	"time"

	examplecomv1beta1 "github.com/ringdrx/visitors-operator/api/v1beta1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

const databasePort = 3000

func mysqlDeploymentName() string {
	return "mysql"
}

func mysqlServiceName() string {
	return "mysql-service"
}

func mysqlAuthName() string {
	return "mysql-auth"
}

func (r *VisitorsAppReconciler) mysqlAuthSecret(v *examplecomv1beta1.VisitorsApp) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqlAuthName(),
			Namespace: v.Namespace,
		},
		Type: "Opaque",
		StringData: map[string]string{
			"username": "visitors-user",
			"password": "visitors-pass",
		},
	}
	controllerutil.SetControllerReference(v, secret, r.Scheme)
	return secret
}

func (r *VisitorsAppReconciler) mysqlDeployment(v *examplecomv1beta1.VisitorsApp) *appsv1.Deployment {
	labels := labels(v, "mysql")
	size := int32(1)
	databaseImage := v.Spec.DatabaseImage

	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key:                  "username",
		},
	}

	passwordSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key:                  "password",
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqlDeploymentName(),
			Namespace: v.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: databaseImage,
						Name:  "visitors-mysql",
						Ports: []corev1.ContainerPort{{
							ContainerPort: databasePort,
							Name:          "mysql",
						}},
						Env: []corev1.EnvVar{
							{
								Name:  "MYSQL_ROOT_PASSWORD",
								Value: "password",
							},
							{
								Name:  "MYSQL_DATABASE",
								Value: "visitors",
							},
							{
								Name:      "MYSQL_USER",
								ValueFrom: userSecret,
							},
							{
								Name:      "MYSQL_PASSWORD",
								ValueFrom: passwordSecret,
							},
						},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(v, dep, r.Scheme)
	return dep
}

func (r *VisitorsAppReconciler) mysqlService(v *examplecomv1beta1.VisitorsApp) *corev1.Service {
	labels := labels(v, "mysql")

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqlServiceName(),
			Namespace: v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port: databasePort,
			}},
			ClusterIP: "None",
		},
	}

	controllerutil.SetControllerReference(v, s, r.Scheme)
	return s
}

// Returns whether or not the MySQL deployment is running
func (r *VisitorsAppReconciler) isMysqlUp(ctx context.Context, v *examplecomv1beta1.VisitorsApp) bool {
	log := ctrllog.FromContext(ctx)
	deployment := &appsv1.Deployment{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      mysqlDeploymentName(),
		Namespace: v.Namespace,
	}, deployment)

	if err != nil {
		log.Error(err, "Deployment mysql not found")
		return false
	}

	if deployment.Status.ReadyReplicas == 1 {
		return true
	}

	return false
}

func (r *VisitorsAppReconciler) updateDatabaseStatus(ctx context.Context, v *examplecomv1beta1.VisitorsApp) error {
	// v.Status.DatabaseImage = v.Spec.DatabaseImage
	err := r.Update(ctx, v)
	return err
}

func (r *VisitorsAppReconciler) handleDatabaseChanges(ctx context.Context, v *examplecomv1beta1.VisitorsApp) (*ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	foundDeployment := &appsv1.Deployment{}
	foundService := &corev1.Service{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      mysqlDeploymentName(),
		Namespace: v.Namespace,
	}, foundDeployment)
	if err != nil {
		// The deployment may not have been created yet, so requeue
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	err = r.Get(ctx, types.NamespacedName{
		Name:      mysqlServiceName(),
		Namespace: v.Namespace,
	}, foundService)
	if err != nil {
		// The service may not have been created yet, so requeue
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	databaseImage := v.Spec.DatabaseImage

	existingDatabaseImage := (*foundDeployment).Spec.Template.Spec.Containers[0].Image

	if databaseImage != existingDatabaseImage {
		(*foundDeployment).Spec.Template.Spec.Containers[0].Image = databaseImage
		err = r.Update(ctx, foundDeployment)
		if err != nil {
			log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
			return &ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return &ctrl.Result{Requeue: true}, nil
	}

	return nil, nil
}
