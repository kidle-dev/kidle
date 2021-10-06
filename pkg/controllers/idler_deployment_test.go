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

var _ = Describe("idling/wakeup Deployments", func() {
	const (
		timeout  = time.Second * 10
		//duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	var (
		ctx = context.Background()
		irKey = types.NamespacedName{Name: "ir-idler-deploy", Namespace: "default"}
	)

	Context("Deployment suite", func() {
		var deployKey = types.NamespacedName{Name: "nginx", Namespace: "default"}
		var deploy = appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/appsv1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployKey.Name,
				Namespace: deployKey.Namespace,
				Labels: map[string]string{
					"app": "nginx",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointer.Int32(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "nginx",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "nginx",
						Namespace: "default",
						Labels: map[string]string{
							"app": "nginx",
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
				Kind:       "Deployment",
				Name:       deployKey.Name,
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
					Kind:       "Deployment",
					Name:       "nginx",
					APIVersion: "apps/appsv1",
				},
				Idle:           false,
				IdlingStrategy: nil,
				WakeupStrategy: nil,
			}))
		})

		It("Is referenced by a Deployment", func() {

			By("Creating the Deployment object")
			Expect(k8sClient.Create(ctx, &deploy)).Should(Succeed())

			d := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, deployKey, d)
			}, timeout, interval).Should(Succeed())

			By("Checking for reference to be set in annotations")
			Expect(d.ObjectMeta.GetAnnotations()[kidlev1beta1.MetadataIdlingResourceReference], irKey.Name)
		})

		It("It should not have idled the deployment", func() {

			By("Getting the Deployment")
			d := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, deployKey, d)).Should(Succeed())

			By("Checking that Replicas has not changed")
			Expect(d.Spec.Replicas).Should(Equal(deploy.Spec.Replicas))
		})

		It("Should idle the Deployment", func() {
			By("Idling the deployment")
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
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(0)))
		})

		It("Should watch the Deployment", func() {
			By("Trying to update replicas on a idled object")
			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				d := &appsv1.Deployment{}
				if err := k8sClient.Get(ctx, deployKey, d); err != nil {
					return err
				}
				d.Spec.Replicas = pointer.Int32(1)
				return k8sClient.Update(ctx, d)
			})).Should(Succeed())

			By("Checking that Replicas still equals to 0")
			Eventually(func() (*int32, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(0)))
		})

		It("Should wakeup the Deployment", func() {
			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				ir := &kidlev1beta1.IdlingResource{}
				if err := k8sClient.Get(ctx, irKey, ir); err != nil {
					return err
				}
				ir.Spec.Idle = false
				return k8sClient.Update(ctx, ir)
			})).Should(Succeed())

			// We'll need to wait until the controller has waked up the Deployment
			Eventually(func() (*int32, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(1)))
		})

		It("Should wakeup the Deployment to previous replicas", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := k8sClient.Get(ctx, irKey, ir); err != nil {
					return err
				}
				ir.Spec.Idle = false
				return k8sClient.Update(ctx, ir)
			})).Should(Succeed())

			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				d := &appsv1.Deployment{}
				if err := k8sClient.Get(ctx, deployKey, d); err != nil {
					return err
				}
				d.Spec.Replicas = pointer.Int32(2)
				return k8sClient.Update(ctx, d)
			})).Should(Succeed())

			Eventually(func() (*int32, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(2)))

			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := k8sClient.Get(ctx, irKey, ir); err != nil {
					return err
				}
				ir.Spec.Idle = true
				return k8sClient.Update(ctx, ir)
			})).Should(Succeed())
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

			// We'll need to wait until the controller has idled the Deployment
			By("idling")
			Eventually(func() (*int32, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(0)))

			Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := k8sClient.Get(ctx, irKey, ir); err != nil {
					return err
				}
				ir.Spec.Idle = false
				return k8sClient.Update(ctx, ir)
			})).Should(Succeed())
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

			// We'll need to wait until the controller has waked up the Deployment
			By("waking up")
			Eventually(func() (*int32, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(2)))
		})

		It("Should wakeup and cleanup Deployment when removing the IdlingResource", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			By("deleting the IdlingResource")
			Expect(k8sClient.Delete(ctx, ir)).Should(Succeed())

			By("checking the Deployment has not been deleted")
			d := &appsv1.Deployment{}
			Consistently(func() error {
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return err
				}
				return nil

			}, 5*time.Second, interval).Should(Succeed())

			By("checking the Deployment annotations have been removed")
			Expect(k8s.HasAnnotation(&d.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas)).ShouldNot(BeTrue())
			Expect(k8s.HasAnnotation(&d.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference)).ShouldNot(BeTrue())
			Expect(k8s.HasAnnotation(&d.ObjectMeta, kidlev1beta1.MetadataExpectedState)).ShouldNot(BeTrue())

			By("checking the Deployment has been scaled up")
			Eventually(func() (*int32, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(2)))
		})
	})
})
