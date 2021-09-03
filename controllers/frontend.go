package controllers

import (
	"context"
	"time"

	examplecomv1beta1 "github.com/ringdrx/visitors-operator/api/v1beta1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

const frontendPort = 3000
const frontendImage = "jdob/visitors-webui:1.0.0"

func frontendDeploymentName(v *examplecomv1beta1.VisitorsApp) string {
	return v.Name + "-frontend"
}

func frontendServiceName(v *examplecomv1beta1.VisitorsApp) string {
	return v.Name + "-frontend-service"
}

func (r *VisitorsAppReconciler) frontendDeployment(v *examplecomv1beta1.VisitorsApp) *appsv1.Deployment {
	labels := labels(v, "frontend")
	frontendTitle := v.Spec.FrontendTitle
	frontendSize := v.Spec.FrontendSize

	// If the header was specified, add it as an env variable
	env := []corev1.EnvVar{}
	env = append(env, corev1.EnvVar{
		Name:  "REACT_APP_TITLE",
		Value: frontendTitle,
	})
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      frontendDeploymentName(v),
			Namespace: v.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &frontendSize,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: frontendImage,
						Name:  "visitors-webui",
						Ports: []corev1.ContainerPort{{
							ContainerPort: frontendPort,
							Name:          "visitors",
						}},
						Env: env,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								"cpu": resource.MustParse("500m"),
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

func (r *VisitorsAppReconciler) frontendService(v *examplecomv1beta1.VisitorsApp) *corev1.Service {
	labels := labels(v, "frontend")
	frontendServiceNodePort := v.Spec.FrontendServiceNodePort

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      frontendServiceName(v),
			Namespace: v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       frontendPort,
				TargetPort: intstr.FromInt(int(frontendPort)),
				NodePort:   frontendServiceNodePort,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	controllerutil.SetControllerReference(v, s, r.Scheme)
	return s
}

func (r *VisitorsAppReconciler) updateFrontendStatus(ctx context.Context, v *examplecomv1beta1.VisitorsApp) error {
	v.Status.FrontendImage = frontendImage
	err := r.Update(ctx, v)
	return err
}

func (r *VisitorsAppReconciler) handleFrontendChanges(ctx context.Context, v *examplecomv1beta1.VisitorsApp) (*ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	foundDeployment := &appsv1.Deployment{}
	foundService := &corev1.Service{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      frontendDeploymentName(v),
		Namespace: v.Namespace,
	}, foundDeployment)
	if err != nil {
		// The deployment may not have been created yet, so requeue
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	err = r.Get(ctx, types.NamespacedName{
		Name:      frontendServiceName(v),
		Namespace: v.Namespace,
	}, foundService)
	if err != nil {
		// The service may not have been created yet, so requeue
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	frontendAutoScaling := v.Spec.FrontendAutoScaling
	frontendTitle := v.Spec.FrontendTitle
	frontendSize := v.Spec.FrontendSize
	frontendServiceNodePort := v.Spec.FrontendServiceNodePort

	existingFrontendTitle := (*foundDeployment).Spec.Template.Spec.Containers[0].Env[0].Value
	existingFrontendSize := *foundDeployment.Spec.Replicas
	existingFrontendServiceNodePort := (*foundService).Spec.Ports[0].NodePort

	if frontendTitle != existingFrontendTitle {
		(*foundDeployment).Spec.Template.Spec.Containers[0].Env[0].Value = frontendTitle
		err = r.Update(ctx, foundDeployment)
		if err != nil {
			log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
			return &ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return &ctrl.Result{Requeue: true}, nil
	}

	if !frontendAutoScaling {
		if frontendSize != existingFrontendSize {
			foundDeployment.Spec.Replicas = &frontendSize
			err = r.Update(ctx, foundDeployment)
			if err != nil {
				log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
				return &ctrl.Result{}, err
			}
			// Spec updated - return and requeue
			return &ctrl.Result{Requeue: true}, nil
		}
	}

	if frontendServiceNodePort != existingFrontendServiceNodePort {
		(*foundService).Spec.Ports[0].NodePort = frontendServiceNodePort
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
