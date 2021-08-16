package controllers

import (
	"context"
	"time"

	examplecomv1beta1 "github.com/ringdrx/visitors-operator/api/v1beta1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

const backendPort = 8000
const backendImage = "kerryduan/visitors-service:1.0.0"

func backendDeploymentName(v *examplecomv1beta1.VisitorsApp) string {
	return v.Name + "-backend"
}

func backendServiceName(v *examplecomv1beta1.VisitorsApp) string {
	return v.Name + "-backend-service"
}

func (r *VisitorsAppReconciler) backendDeployment(v *examplecomv1beta1.VisitorsApp) *appsv1.Deployment {
	labels := labels(v, "backend")
	backendSize := v.Spec.BackendSize

	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key:                  "USER",
		},
	}

	passwordSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key:                  "PASSWORD",
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backendDeploymentName(v),
			Namespace: v.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &backendSize,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           backendImage,
						ImagePullPolicy: corev1.PullAlways,
						Name:            "visitors-service",
						Ports: []corev1.ContainerPort{{
							ContainerPort: backendPort,
							Name:          "visitors",
						}},
						Env: []corev1.EnvVar{
							{
								Name:  "MYSQL_DATABASE",
								Value: "visitors_db",
							},
							{
								Name:  "MYSQL_SERVICE_HOST_RW",
								Value: mysqlServiceRWName(),
							},
							{
								Name:  "MYSQL_SERVICE_HOST_RO",
								Value: mysqlServiceROName(),
							},
							{
								Name:      "MYSQL_USERNAME",
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

func (r *VisitorsAppReconciler) backendService(v *examplecomv1beta1.VisitorsApp) *corev1.Service {
	labels := labels(v, "backend")
	backendServiceNodePort := v.Spec.BackendServiceNodePort

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backendServiceName(v),
			Namespace: v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       backendPort,
				TargetPort: intstr.FromInt(int(backendPort)),
				NodePort:   backendServiceNodePort,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	controllerutil.SetControllerReference(v, s, r.Scheme)
	return s
}

func (r *VisitorsAppReconciler) updateBackendStatus(ctx context.Context, v *examplecomv1beta1.VisitorsApp) error {
	v.Status.BackendImage = backendImage
	err := r.Update(ctx, v)
	return err
}

func (r *VisitorsAppReconciler) handleBackendChanges(ctx context.Context, v *examplecomv1beta1.VisitorsApp) (*ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	foundDeployment := &appsv1.Deployment{}
	foundService := &corev1.Service{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      backendDeploymentName(v),
		Namespace: v.Namespace,
	}, foundDeployment)
	if err != nil {
		// The deployment may not have been created yet, so requeue
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	err = r.Get(ctx, types.NamespacedName{
		Name:      backendServiceName(v),
		Namespace: v.Namespace,
	}, foundService)
	if err != nil {
		// The service may not have been created yet, so requeue
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	backendSize := v.Spec.BackendSize
	backendServiceNodePort := v.Spec.BackendServiceNodePort

	existingBackendSize := *foundDeployment.Spec.Replicas
	existingBackendServiceNodePort := (*foundService).Spec.Ports[0].NodePort

	if backendSize != existingBackendSize {
		foundDeployment.Spec.Replicas = &backendSize
		err = r.Update(ctx, foundDeployment)
		if err != nil {
			log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
			return &ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return &ctrl.Result{Requeue: true}, nil
	}

	if backendServiceNodePort != existingBackendServiceNodePort {
		(*foundService).Spec.Ports[0].NodePort = backendServiceNodePort
		err = r.Update(ctx, foundService)
		if err != nil {
			log.Error(err, "Failed to update Service.", "Service.Namespace", foundService.Namespace, "Service.Name", foundService.Name)
			return &ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return &ctrl.Result{Requeue: true}, nil
	}

	return nil, nil
}
