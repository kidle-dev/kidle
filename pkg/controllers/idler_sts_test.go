package controllers

import (
	"context"
	kidlev1beta1 "github.com/kidle-dev/kidle/pkg/api/v1beta1"
	"github.com/kidle-dev/kidle/pkg/utils/k8s"
	"github.com/kidle-dev/kidle/pkg/utils/pointer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"time"
)

var _ = Describe("idling/wakeup StatefulSets", func() {
	const (
		timeout  = time.Second * 10
		//duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	var (
		ctx = context.Background()
		irKey = types.NamespacedName{Name: "ir-idler-sts", Namespace: "default"}
	)

	Context("StatefulSet suite", func() {
		var stsKey = types.NamespacedName{Name: "nginx-sts", Namespace: "default"}
		var statefulSet = appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: "apps/appsv1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      stsKey.Name,
				Namespace: stsKey.Namespace,
				Labels: map[string]string{
					"app": "nginx-sts",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointer.Int32(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "nginx-sts",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "nginx",
						Namespace: "default",
						Labels: map[string]string{
							"app": "nginx-sts",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "nginx",
								Image: "nginx",
							},
						},
					},
				},
			},
		}

		It("Has created an IdlingResource object", func() {

			By("Creating the IdlingResource object")
			ref := kidlev1beta1.CrossVersionObjectReference{
				Kind:       "StatefulSet",
				Name:       stsKey.Name,
				APIVersion: "apps/appsv1",
			}
			Expect(k8sClient.Create(ctx, newIdlingResource(irKey, &ref))).Should(Succeed())

			By("Validation of the IdlingResource creation")
			createdIR := &kidlev1beta1.IdlingResource{}
			Eventually(func() error {
				return k8sClient.Get(ctx, irKey, createdIR)
			}, timeout, interval).Should(Succeed())
			Expect(createdIR.Spec).To(Equal(kidlev1beta1.IdlingResourceSpec{
				IdlingResourceRef: kidlev1beta1.CrossVersionObjectReference{
					Kind:       "StatefulSet",
					Name:       "nginx-sts",
					APIVersion: "apps/appsv1",
				},
				Idle:           false,
				IdlingStrategy: nil,
				WakeupStrategy: nil,
			}))
		})

		It("Is referenced by a StatefulSet", func() {

			By("Creating the StatefulSet object")
			Expect(k8sClient.Create(ctx, &statefulSet)).Should(Succeed())

			sts := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, stsKey, sts)
			}, timeout, interval).Should(Succeed())

			By("Checking for reference to be set in annotations")
			Expect(sts.ObjectMeta.GetAnnotations()[kidlev1beta1.MetadataIdlingResourceReference], irKey.Name)
		})

		It("It should not have idled the statefulset", func() {

			By("Getting the StatefulSet")
			sts := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, stsKey, sts)).Should(Succeed())

			By("Checking that Replicas has not changed")
			Expect(sts.Spec.Replicas).Should(Equal(sts.Spec.Replicas))
		})

		It("Should idle the StatefulSet", func() {
			By("Idling the statefulset")
			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				ir := &kidlev1beta1.IdlingResource{}
				if err := k8sClient.Get(ctx, irKey, ir); err != nil {
					return err
				}
				ir.Spec.Idle = true
				return k8sClient.Update(ctx, ir)
			})).Should(Succeed())

			By("Checking that Replicas == 0")
			Eventually(func() (*int32, error) {
				sts := &appsv1.StatefulSet{}
				err := k8sClient.Get(ctx, stsKey, sts)
				if err != nil {
					return nil, err
				}
				return sts.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(0)))
		})

		It("Should watch the StatefulSet", func() {
			By("Trying to update replicas on a idled object")
			s := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, stsKey, s)).Should(Succeed())

			s.Spec.Replicas = pointer.Int32(1)
			Expect(k8sClient.Update(ctx, s)).Should(Succeed())

			By("Checking that Replicas still equals to 0")
			Eventually(func() (*int32, error) {
				s := &appsv1.StatefulSet{}
				err := k8sClient.Get(ctx, stsKey, s)
				if err != nil {
					return nil, err
				}
				return s.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(0)))
		})

		It("Should wakeup the StatefulSet", func() {
			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				ir := &kidlev1beta1.IdlingResource{}
				if err := k8sClient.Get(ctx, irKey, ir); err != nil {
					return err
				}
				ir.Spec.Idle = false
				return k8sClient.Update(ctx, ir)
			})).Should(Succeed())


			// We'll need to wait until the controller has waked up the StatefulSet
			Eventually(func() (*int32, error) {
				sts := &appsv1.StatefulSet{}
				err := k8sClient.Get(ctx, stsKey, sts)
				if err != nil {
					return nil, err
				}
				return sts.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(1)))
		})

		It("Should wakeup the StatefulSet to previous replicas", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := k8sClient.Get(ctx, irKey, ir); err != nil {
					return err
				}
				ir.Spec.Idle = false
				return k8sClient.Update(ctx, ir)
			})).Should(Succeed())

			sts := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, stsKey, sts)).Should(Succeed())

			sts.Spec.Replicas = pointer.Int32(2)
			Expect(k8sClient.Update(ctx, sts)).Should(Succeed())

			Eventually(func() (*int32, error) {
				sts := &appsv1.StatefulSet{}
				err := k8sClient.Get(ctx, stsKey, sts)
				if err != nil {
					return nil, err
				}
				return sts.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(2)))

			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := k8sClient.Get(ctx, irKey, ir); err != nil {
					return err
				}
				ir.Spec.Idle = true
				return k8sClient.Update(ctx, ir)
			})).Should(Succeed())

			// We'll need to wait until the controller has idled the StatefulSet
			By("idling")
			Eventually(func() (*int32, error) {
				sts := &appsv1.StatefulSet{}
				err := k8sClient.Get(ctx, stsKey, sts)
				if err != nil {
					return nil, err
				}
				return sts.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(0)))

			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := k8sClient.Get(ctx, irKey, ir); err != nil {
					return err
				}
				ir.Spec.Idle = false
				return k8sClient.Update(ctx, ir)
			})).Should(Succeed())

			// We'll need to wait until the controller has waked up the StatefulSet
			By("waking up")
			Eventually(func() (*int32, error) {
				sts := &appsv1.StatefulSet{}
				err := k8sClient.Get(ctx, stsKey, sts)
				if err != nil {
					return nil, err
				}
				return sts.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(2)))
		})

		It("Should wakeup and cleanup StatefulSet when removing the IdlingResource", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			By("deleting the IdlingResource")
			Expect(k8sClient.Delete(ctx, ir)).Should(Succeed())

			By("checking the StatefulSet has not been deleted")
			sts := &appsv1.StatefulSet{}
			Consistently(func() error {
				err := k8sClient.Get(ctx, stsKey, sts)
				if err != nil {
					return err
				}
				return nil

			}, 5*time.Second, interval).Should(Succeed())

			By("checking the StatefulSet annotations have been removed")
			Expect(k8s.HasAnnotation(&sts.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas)).ShouldNot(BeTrue())
			Expect(k8s.HasAnnotation(&sts.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference)).ShouldNot(BeTrue())
			Expect(k8s.HasAnnotation(&sts.ObjectMeta, kidlev1beta1.MetadataExpectedState)).ShouldNot(BeTrue())

			By("checking the StatefulSet has been scaled up")
			Expect(sts.Spec.Replicas).Should(Equal(pointer.Int32(2)))
		})
	})
})
