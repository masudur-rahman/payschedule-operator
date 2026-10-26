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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	payv1 "github.com/masudur-rahman/payschedule-operator/api/v1"
)

var _ = Describe("PaySchedule Controller", func() {
	ctx := context.Background()
	key := types.NamespacedName{Name: "payschedule-test", Namespace: "default"}
	cronKey := types.NamespacedName{Name: "pay-cron-test", Namespace: "default"}

	var reconciler *PayScheduleReconciler

	BeforeEach(func() {
		reconciler = &PayScheduleReconciler{Client: k8sClient, Scheme: scheme.Scheme}

		ps := &payv1.PaySchedule{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Spec: payv1.PayScheduleSpec{
				CronName:     cronKey.Name,
				CronSchedule: "*/5 * * * *",
				JobImage:     "busybox:latest",
				JobCmd:       "echo hello",
			},
		}
		Expect(k8sClient.Create(ctx, ps)).To(Succeed())
	})

	AfterEach(func() {
		// envtest has no garbage collector, so delete the CronJob ourselves
		Expect(k8sClient.Delete(ctx, &payv1.PaySchedule{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}})).To(Succeed())
		_ = k8sClient.Delete(ctx, &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: cronKey.Name, Namespace: cronKey.Namespace}})
	})

	It("creates an owned CronJob from the spec", func() {
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
		Expect(err).NotTo(HaveOccurred())

		cronJob := &batchv1.CronJob{}
		Expect(k8sClient.Get(ctx, cronKey, cronJob)).To(Succeed())
		Expect(cronJob.Spec.Schedule).To(Equal("*/5 * * * *"))

		containers := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers
		Expect(containers).To(HaveLen(1))
		Expect(containers[0].Image).To(Equal("busybox:latest"))
		Expect(containers[0].Args).To(Equal([]string{"-c", "echo hello"}))

		Expect(cronJob.OwnerReferences).To(HaveLen(1))
		Expect(cronJob.OwnerReferences[0].Kind).To(Equal("PaySchedule"))
		Expect(cronJob.OwnerReferences[0].Name).To(Equal(key.Name))
	})

	It("updates the CronJob when the spec changes", func() {
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
		Expect(err).NotTo(HaveOccurred())

		ps := &payv1.PaySchedule{}
		Expect(k8sClient.Get(ctx, key, ps)).To(Succeed())
		ps.Spec.CronSchedule = "0 * * * *"
		ps.Spec.JobCmd = "echo updated"
		Expect(k8sClient.Update(ctx, ps)).To(Succeed())

		_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
		Expect(err).NotTo(HaveOccurred())

		cronJob := &batchv1.CronJob{}
		Expect(k8sClient.Get(ctx, cronKey, cronJob)).To(Succeed())
		Expect(cronJob.Spec.Schedule).To(Equal("0 * * * *"))
		Expect(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args).To(Equal([]string{"-c", "echo updated"}))
	})

	It("recreates the CronJob if it gets deleted", func() {
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
		Expect(err).NotTo(HaveOccurred())

		Expect(k8sClient.Delete(ctx, &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: cronKey.Name, Namespace: cronKey.Namespace}})).To(Succeed())

		_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
		Expect(err).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, cronKey, &batchv1.CronJob{})).To(Succeed())
	})

	It("ignores a PaySchedule that no longer exists", func() {
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "missing", Namespace: "default"}})
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects a PaySchedule with an empty schedule", func() {
		ps := &payv1.PaySchedule{
			ObjectMeta: metav1.ObjectMeta{Name: "invalid", Namespace: "default"},
			Spec: payv1.PayScheduleSpec{
				CronName: "invalid",
				JobImage: "busybox:latest",
				JobCmd:   "echo hello",
			},
		}
		Expect(k8sClient.Create(ctx, ps)).NotTo(Succeed())
	})
})
