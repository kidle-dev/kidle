package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/k8s"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

func newIdlingResource(key types.NamespacedName, ref *kidlev1beta1.CrossVersionObjectReference) *kidlev1beta1.IdlingResource {
	return &kidlev1beta1.IdlingResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IdlingResource",
			APIVersion: "kidle.beroot.org/v1beta1", // kidlev1beta1.GroupVersion.String()
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: kidlev1beta1.IdlingResourceSpec{
			IdlingResourceRef: *ref,
			Idle:              false,
		},
	}
}

var _ = Describe("idling/wakeup Cronjobs", func() {
	const (
		timeout  = time.Second * 10
		//duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	var (
		irKey = types.NamespacedName{Name: "ir-idler-cronjob", Namespace: "default"}
		ctx = context.Background()
	)

	Context("CronJob suite", func() {
		var cronJobKey = types.NamespacedName{Name: "hello-world", Namespace: "default"}
		var cronJob = batchv1beta1.CronJob{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: "batch/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cronJobKey.Name,
				Namespace: cronJobKey.Namespace,
			},
			Spec: batchv1beta1.CronJobSpec{
				Suspend:  pointer.Bool(false),
				Schedule: "*/1 * * * *",
				JobTemplate: batchv1beta1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{},
							Spec: corev1.PodSpec{
								RestartPolicy: corev1.RestartPolicyNever,
								Containers: []corev1.Container{
									{
										Name:  "nginx",
										Image: "nginx",
									},
								},
							},
						},
					},
				},
			},
		}

		It("Has created an IdlingResource object", func() {

			By("Creating the IdlingResource object")
			ref := kidlev1beta1.CrossVersionObjectReference{
				Kind:       "CronJob",
				Name:       cronJobKey.Name,
				APIVersion: "batch/v1beta1",
			}
			Expect(k8sClient.Create(ctx, newIdlingResource(irKey, &ref))).Should(Succeed())

			By("Validation of the IdlingResource creation")
			createdIR := &kidlev1beta1.IdlingResource{}
			Eventually(func() error {
				return k8sClient.Get(ctx, irKey, createdIR)
			}, timeout, interval).Should(Succeed())
			Expect(createdIR.Spec).To(Equal(kidlev1beta1.IdlingResourceSpec{
				IdlingResourceRef: kidlev1beta1.CrossVersionObjectReference{
					Kind:       "CronJob",
					Name:       "hello-world",
					APIVersion: "batch/v1beta1",
				},
				Idle:           false,
				IdlingStrategy: nil,
				WakeupStrategy: nil,
			}))
		})

		It("Is referenced by a CronJob", func() {

			By("Creating the CronJob object")
			Expect(k8sClient.Create(ctx, &cronJob)).Should(Succeed())

			cj := &batchv1beta1.CronJob{}
			Eventually(func() error {
				return k8sClient.Get(ctx, cronJobKey, cj)
			}, timeout, interval).Should(Succeed())

			By("Checking for reference to be set in annotations")
			Expect(cj.ObjectMeta.GetAnnotations()[kidlev1beta1.MetadataIdlingResourceReference], irKey.Name)
		})

		It("It should not have idled the statefulset", func() {

			By("Getting the CronJob")
			cj := &batchv1beta1.CronJob{}
			Expect(k8sClient.Get(ctx, cronJobKey, cj)).Should(Succeed())

			By("Checking that suspend field has not changed")
			Expect(cj.Spec.Suspend).Should(Equal(cronJob.Spec.Suspend))
		})

		It("Should suspend the CronJob", func() {
			By("Idling the cronjob")
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = true
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

			By("Checking that Suspend == true")
			Eventually(func() (*bool, error) {
				cj := &batchv1beta1.CronJob{}
				err := k8sClient.Get(ctx, cronJobKey, cj)
				if err != nil {
					return nil, err
				}
				return cj.Spec.Suspend, nil
			}, timeout, interval).Should(Equal(pointer.Bool(true)))
		})

		It("Should watch the CronJob", func() {
			By("Trying to update suspend field on a idled object")
			cj := &batchv1beta1.CronJob{}
			Expect(k8sClient.Get(ctx, cronJobKey, cj)).Should(Succeed())

			cj.Spec.Suspend = pointer.Bool(false)
			Expect(k8sClient.Update(ctx, cj)).Should(Succeed())

			// We'll need to wait until the controller has idled the CronJob
			Eventually(func() (*bool, error) {
				cj := &batchv1beta1.CronJob{}
				err := k8sClient.Get(ctx, cronJobKey, cj)
				if err != nil {
					return nil, err
				}
				return cj.Spec.Suspend, nil
			}, timeout, interval).Should(Equal(pointer.Bool(true)))
		})

		It("Should wakeup the CronJob", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = false
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

			// We'll need to wait until the controller has waked up the CronJob
			Eventually(func() (*bool, error) {
				cj := &batchv1beta1.CronJob{}
				err := k8sClient.Get(ctx, cronJobKey, cj)
				if err != nil {
					return nil, err
				}
				return cj.Spec.Suspend, nil
			}, timeout, interval).Should(Equal(pointer.Bool(false)))
		})

		It("Should wakeup and cleanup Cronjob when removing the IdlingResource", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			By("deleting the IdlingResource")
			Expect(k8sClient.Delete(ctx, ir)).Should(Succeed())

			By("checking the CronJob has not been deleted")
			cj := &batchv1beta1.CronJob{}
			Consistently(func() error {
				err := k8sClient.Get(ctx, cronJobKey, cj)
				if err != nil {
					return err
				}
				return nil

			}, 5*time.Second, interval).Should(Succeed())

			By("checking the CronJob annotations have been removed")
			Expect(k8s.HasAnnotation(&cj.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas)).ShouldNot(BeTrue())
			Expect(k8s.HasAnnotation(&cj.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference)).ShouldNot(BeTrue())
			Expect(k8s.HasAnnotation(&cj.ObjectMeta, kidlev1beta1.MetadataExpectedState)).ShouldNot(BeTrue())

			By("checking the CronJob has been scaled up")
			Expect(cj.Spec.Suspend).Should(Equal(pointer.Bool(false)))
		})
	})
})
