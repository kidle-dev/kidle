package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/k8s"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var irKey = types.NamespacedName{Name: "ir", Namespace: "ns"}

func newIdlingResource(ref *kidlev1beta1.CrossVersionObjectReference) *kidlev1beta1.IdlingResource {
	return &kidlev1beta1.IdlingResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IdlingResource",
			APIVersion: "kidle.beroot.org/v1beta1", // kidlev1beta1.GroupVersion.String()
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      irKey.Name,
			Namespace: irKey.Namespace,
		},
		Spec: kidlev1beta1.IdlingResourceSpec{
			IdlingResourceRef: *ref,
			Idle:              false,
		},
	}
}

var _ = Describe("IdlingResource Controller", func() {
	const (
		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	var (
		ctx = context.Background()
	)
	Context("Initially", func() {
		var deployKey = types.NamespacedName{Name: "nginx", Namespace: "ns"}
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
						Namespace: "ns",
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
			Expect(k8sClient.Create(ctx, newIdlingResource(&ref))).Should(Succeed())

			By("Validation of the IdlingResource creation")
			createdIR := &kidlev1beta1.IdlingResource{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, irKey, createdIR)
				return err == nil
			}, timeout, interval).Should(BeTrue())
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

			Eventually(func() bool {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking for reference to be set in annotations")
			Eventually(func() (string, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return "", err
				}
				return d.ObjectMeta.GetAnnotations()[kidlev1beta1.MetadataIdlingResourceReference], nil
			}, timeout, interval).Should(Equal("ir"))
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
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = true
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

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

		It("Should wakeup the Deployment", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = false
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

			// We'll need to wait until the controller has idled the Deployment
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
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = false
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

			d := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, deployKey, d)).Should(Succeed())

			d.Spec.Replicas = pointer.Int32(2)
			Expect(k8sClient.Update(ctx, d)).Should(Succeed())

			Eventually(func() (*int32, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(2)))

			ir.Spec.Idle = true
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

			ir.Spec.Idle = false
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

			By("checking the Deployment has been scaled up")
			Expect(d.Spec.Replicas).Should(Equal(pointer.Int32(2)))
		})
	})

	Context("Initially", func() {
		var stsKey = types.NamespacedName{Name: "nginx-sts", Namespace: "ns"}
		var sts = appsv1.StatefulSet{
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
						Namespace: "ns",
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
			Expect(k8sClient.Create(ctx, newIdlingResource(&ref))).Should(Succeed())

			By("Validation of the IdlingResource creation")
			createdIR := &kidlev1beta1.IdlingResource{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, irKey, createdIR)
				return err == nil
			}, timeout, interval).Should(BeTrue())
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
			Expect(k8sClient.Create(ctx, &sts)).Should(Succeed())

			Eventually(func() bool {
				sts := &appsv1.StatefulSet{}
				err := k8sClient.Get(ctx, stsKey, sts)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking for reference to be set in annotations")
			Eventually(func() (string, error) {
				sts := &appsv1.StatefulSet{}
				err := k8sClient.Get(ctx, stsKey, sts)
				if err != nil {
					return "", err
				}
				return sts.ObjectMeta.GetAnnotations()[kidlev1beta1.MetadataIdlingResourceReference], nil
			}, timeout, interval).Should(Equal("ir"))
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
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = true
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

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

		It("Should wakeup the StatefulSet", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = false
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

			// We'll need to wait until the controller has idled the StatefulSet
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
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = false
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

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

			ir.Spec.Idle = true
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

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

			ir.Spec.Idle = false
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

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

		It("Should wakeup and cleanup Deployment when removing the IdlingResource", func() {
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

			By("checking the StatefulSet has been scaled up")
			Expect(sts.Spec.Replicas).Should(Equal(pointer.Int32(2)))
		})
	})

	Context("CronJob suite", func() {
		var cronJobKey = types.NamespacedName{Name: "hello-world", Namespace: "ns"}
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
			Expect(k8sClient.Create(ctx, newIdlingResource(&ref))).Should(Succeed())

			By("Validation of the IdlingResource creation")
			createdIR := &kidlev1beta1.IdlingResource{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, irKey, createdIR)
				return err == nil
			}, timeout, interval).Should(BeTrue())
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

			Eventually(func() bool {
				cj := &batchv1beta1.CronJob{}
				err := k8sClient.Get(ctx, cronJobKey, cj)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking for reference to be set in annotations")
			Eventually(func() (string, error) {
				cj := &batchv1beta1.CronJob{}
				err := k8sClient.Get(ctx, cronJobKey, cj)
				if err != nil {
					return "", err
				}
				return cj.ObjectMeta.GetAnnotations()[kidlev1beta1.MetadataIdlingResourceReference], nil
			}, timeout, interval).Should(Equal("ir"))
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

		It("Should wakeup the CronJob", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = false
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

			// We'll need to wait until the controller has idled the CronJob
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

			By("checking the CronJob has been scaled up")
			Expect(cj.Spec.Suspend).Should(Equal(pointer.Bool(false)))
		})
	})

})
