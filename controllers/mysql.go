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

func mysqlStatefulSetName() string {
	return "mysql"
}

func mysqlServiceName() string {
	return "mysql-service"
}

func mysqlAuthName() string {
	return "mysql-auth"
}

func mysqlVolumeName() string {
	return "mysql-volume"
}

func mysqlMountPath() string {
	return "//var/lib/mysql"
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

func (r *VisitorsAppReconciler) mysqlStatefulSet(v *examplecomv1beta1.VisitorsApp) *appsv1.StatefulSet {
	labels := labels(v, "mysql")
	size := int32(1)
	databaseImage := "mysql:" + v.Spec.DatabaseVersion
	databaseHostPath := v.Spec.DatabaseHostPath
	databaseMySQLRootPassword := v.Spec.DatabaseMySQLRootPassword
	hostPathType := corev1.HostPathDirectoryOrCreate

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

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqlStatefulSetName(),
			Namespace: v.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: mysqlServiceName(),
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
								Value: databaseMySQLRootPassword,
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
						VolumeMounts: []corev1.VolumeMount{{
							Name:      mysqlVolumeName(),
							MountPath: mysqlMountPath(),
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: mysqlVolumeName(),
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: databaseHostPath,
								Type: &hostPathType,
							},
						},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(v, sts, r.Scheme)
	return sts
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

// Returns whether or not the MySQL statefulset is running
func (r *VisitorsAppReconciler) isMysqlUp(ctx context.Context, v *examplecomv1beta1.VisitorsApp) bool {
	log := ctrllog.FromContext(ctx)
	statefulset := &appsv1.StatefulSet{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      mysqlStatefulSetName(),
		Namespace: v.Namespace,
	}, statefulset)

	if err != nil {
		log.Error(err, "StatefulSet mysql not found")
		return false
	}

	if statefulset.Status.ReadyReplicas == 1 {
		return true
	}

	return false
}

func (r *VisitorsAppReconciler) updateDatabaseStatus(ctx context.Context, v *examplecomv1beta1.VisitorsApp) error {
	v.Status.DatabaseImage = "mysql:" + v.Spec.DatabaseVersion
	err := r.Update(ctx, v)
	return err
}

func (r *VisitorsAppReconciler) handleDatabaseChanges(ctx context.Context, v *examplecomv1beta1.VisitorsApp) (*ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	foundService := &corev1.Service{}
	foundStatefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      mysqlServiceName(),
		Namespace: v.Namespace,
	}, foundService)
	if err != nil {
		// The service may not have been created yet, so requeue
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	err = r.Get(ctx, types.NamespacedName{
		Name:      mysqlStatefulSetName(),
		Namespace: v.Namespace,
	}, foundStatefulSet)
	if err != nil {
		// The statefulset may not have been created yet, so requeue
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	databaseMySQLRootPassword := v.Spec.DatabaseMySQLRootPassword
	databaseImage := "mysql:" + v.Spec.DatabaseVersion
	databaseHostPath := v.Spec.DatabaseHostPath

	existingDatabaseMySQLRootPassword := (*foundStatefulSet).Spec.Template.Spec.Containers[0].Env[0].Value
	existingDatabaseImage := (*foundStatefulSet).Spec.Template.Spec.Containers[0].Image
	existingDatabaseHostPath := (*foundStatefulSet).Spec.Template.Spec.Volumes[0].VolumeSource.HostPath.Path

	if databaseMySQLRootPassword != existingDatabaseMySQLRootPassword {
		(*foundStatefulSet).Spec.Template.Spec.Containers[0].Env[0].Value = databaseMySQLRootPassword
		err = r.Update(ctx, foundStatefulSet)
		if err != nil {
			log.Error(err, "Failed to update StatefulSet.", "StatefulSet.Namespace", foundStatefulSet.Namespace, "StatefulSet.Name", foundStatefulSet.Name)
			return &ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return &ctrl.Result{Requeue: true}, nil
	}

	if databaseImage != existingDatabaseImage {
		(*foundStatefulSet).Spec.Template.Spec.Containers[0].Image = databaseImage
		err = r.Update(ctx, foundStatefulSet)
		if err != nil {
			log.Error(err, "Failed to update StatefulSet.", "StatefulSet.Namespace", foundStatefulSet.Namespace, "StatefulSet.Name", foundStatefulSet.Name)
			return &ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return &ctrl.Result{Requeue: true}, nil
	}

	if databaseHostPath != existingDatabaseHostPath {
		(*foundStatefulSet).Spec.Template.Spec.Volumes[0].VolumeSource.HostPath.Path = databaseHostPath
		err = r.Update(ctx, foundStatefulSet)
		if err != nil {
			log.Error(err, "Failed to update StatefulSet.", "StatefulSet.Namespace", foundStatefulSet.Namespace, "StatefulSet.Name", foundStatefulSet.Name)
			return &ctrl.Result{}, err
		}

		// Updata Backend When Database HostPath Is Modified
		log.Info("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
		foundBackendDeployment := &appsv1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      backendDeploymentName(v),
			Namespace: v.Namespace,
		}, foundBackendDeployment)
		if err != nil {
			// The deployment may not have been created yet, so requeue
			return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
		zero := int32(0)
		foundBackendDeployment.Spec.Replicas = &zero
		err = r.Update(ctx, foundBackendDeployment)
		if err != nil {
			log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", foundBackendDeployment.Namespace, "Deployment.Name", foundBackendDeployment.Name)
			return &ctrl.Result{}, err
		}

		// Spec updated - return and requeue
		return &ctrl.Result{Requeue: true}, nil
	}

	return nil, nil
}
