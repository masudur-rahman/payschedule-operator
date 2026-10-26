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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile makes sure a CronJob exists for every PaySchedule and matches its spec,
// then copies the CronJob's last schedule time into the PaySchedule status.
// The CronJob is owned by the PaySchedule, so it is garbage collected on delete.
func (r *PayScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	ps := &payv1.PaySchedule{}
	if err := r.Get(ctx, req.NamespacedName, ps); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.Spec.CronName,
			Namespace: ps.Namespace,
		},
	}
	op, err := r.createOrUpdateCronJob(ctx, ps, cronJob)
	if err != nil {
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		l.Info("CronJob reconciled", "name", cronJob.Name, "operation", op)
	}

	if err := r.updatePayScheduleStatus(ctx, ps, cronJob); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PayScheduleReconciler) createOrUpdateCronJob(ctx context.Context, ps *payv1.PaySchedule, cronJob *batchv1.CronJob) (controllerutil.OperationResult, error) {
	return controllerutil.CreateOrUpdate(ctx, r.Client, cronJob, func() error {
		cronJob.Spec.Schedule = ps.Spec.CronSchedule

		// only set the fields we own, so defaults filled by the api server are kept
		podSpec := &cronJob.Spec.JobTemplate.Spec.Template.Spec
		if len(podSpec.Containers) == 0 {
			podSpec.Containers = []corev1.Container{{Name: "pay"}}
		}
		podSpec.Containers[0].Image = ps.Spec.JobImage
		podSpec.Containers[0].Command = []string{"/bin/sh"}
		podSpec.Containers[0].Args = []string{"-c", ps.Spec.JobCmd}
		podSpec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
		podSpec.RestartPolicy = corev1.RestartPolicyNever

		return controllerutil.SetControllerReference(ps, cronJob, r.Scheme)
	})
}

func (r *PayScheduleReconciler) updatePayScheduleStatus(ctx context.Context, ps *payv1.PaySchedule, cronJob *batchv1.CronJob) error {
	if ps.Status.LastScheduled.Equal(cronJob.Status.LastScheduleTime) {
		return nil
	}

	ps.Status.LastScheduled = cronJob.Status.LastScheduleTime
	return r.Status().Update(ctx, ps)
}

// SetupWithManager sets up the controller with the Manager.
// Owns() re-triggers reconcile whenever an owned CronJob changes or gets deleted.
func (r *PayScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&payv1.PaySchedule{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}
