/*
Copyright 2023 Masudur Rahman.

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

package controller

import (
	"context"
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	payv1 "github.com/masudur-rahman/payschedule-operator/api/v1"
)

// PayScheduleReconciler reconciles a PaySchedule object
type PayScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=pay.mrahman.xyz,resources=payschedules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pay.mrahman.xyz,resources=payschedules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pay.mrahman.xyz,resources=payschedules/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PaySchedule object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PayScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	ps := &payv1.PaySchedule{}
	if err := r.Get(ctx, req.NamespacedName, ps); err != nil {
		return ctrl.Result{}, err
	}

	finalizer := payv1.GroupVersion.Group + "/finalizer"
	if !ps.DeletionTimestamp.IsZero() {
		controllerutil.RemoveFinalizer(ps, finalizer)
		err := r.Update(ctx, ps)
		return ctrl.Result{}, err
	}
	if !controllerutil.ContainsFinalizer(ps, finalizer) {
		if err := r.addFinalizer(ctx, ps, finalizer); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.createCronJob(ctx, ps); err != nil {
			return ctrl.Result{}, err
		}

		l.Info("CronJob Created", "name", ps.Spec.CronName)
	}

	if err := r.updatePayScheduleStatus(ctx, ps); err != nil {
		return ctrl.Result{}, err
	}

	if ps.Status.LastScheduled != nil {
		l.Info("CronJob status Created", "last scheduled", ps.Status.LastScheduled.String())
	}

	return ctrl.Result{}, nil
}

func (r *PayScheduleReconciler) createCronJob(ctx context.Context, ps *payv1.PaySchedule) error {
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.Spec.CronName,
			Namespace: ps.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: ps.APIVersion,
					Kind:       ps.Kind,
					Name:       ps.Name,
					UID:        ps.UID,
					Controller: pointer.Bool(true),
				},
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: ps.Spec.CronSchedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "pay",
									Image:           ps.Spec.JobImage,
									Command:         []string{"/bin/sh"},
									Args:            []string{"-c", fmt.Sprintf("%s", ps.Spec.JobCmd)},
									ImagePullPolicy: corev1.PullIfNotPresent,
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			},
		},
	}

	return r.Client.Create(ctx, cronJob)
}

func (r *PayScheduleReconciler) updatePayScheduleStatus(ctx context.Context, ps *payv1.PaySchedule) error {
	cronJob := &batchv1.CronJob{}
	if err := r.Get(ctx, types.NamespacedName{Name: ps.Spec.CronName, Namespace: ps.Namespace}, cronJob); err != nil {
		return err
	}

	ps.Status.LastScheduled = cronJob.Status.LastScheduleTime
	return r.Client.Status().Update(ctx, ps)
}

func (r *PayScheduleReconciler) addFinalizer(ctx context.Context, ps *payv1.PaySchedule, finalizer string) error {
	controllerutil.AddFinalizer(ps, finalizer)
	return r.Update(ctx, ps)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PayScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&payv1.PaySchedule{}).
		Complete(r)
}
