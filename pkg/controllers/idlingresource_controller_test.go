package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

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

var irKey = types.NamespacedName{Name: "ir", Namespace: "ns"}
var ir = kidlev1beta1.IdlingResource{
	TypeMeta: metav1.TypeMeta{
		Kind:       "IdlingResource",
		APIVersion: "kidle.beroot.org/v1beta1", // kidlev1beta1.GroupVersion.String()
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      irKey.Name,
		Namespace: irKey.Namespace,
	},
	Spec: kidlev1beta1.IdlingResourceSpec{
		IdlingResourceRef: kidlev1beta1.CrossVersionObjectReference{
			Kind:       "Deployment",
			Name:       "nginx",
			APIVersion: "apps/appsv1",
		},
		Idle: false,
	},
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
		It("Has created an IdlingResource object", func() {

			By("Creating the IdlingResource object")
			Expect(k8sClient.Create(ctx, &ir)).Should(Succeed())

			// Wait for the IdlingResource object to be created
			createdIR := &kidlev1beta1.IdlingResource{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, irKey, createdIR)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdIR.Spec.Idle).Should(Equal(false))
			Expect(createdIR.Spec.IdlingResourceRef.Name).Should(Equal("nginx"))
			Expect(createdIR.Spec.IdlingResourceRef.Kind).Should(Equal("Deployment"))
			Expect(createdIR.Spec.IdlingResourceRef.APIVersion).Should(Equal("apps/appsv1"))
		})

		It("Owns a Deployment", func() {

			By("Creating the Deployment object")
			Expect(k8sClient.Create(ctx, &deploy)).Should(Succeed())

			Eventually(func() bool {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() ([]metav1.OwnerReference, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.ObjectMeta.GetOwnerReferences(), nil
			}, timeout, interval).Should(Not(BeEmpty()))
		})

		It("It should not have idled the deployment", func() {

			By("Getting the Deployment")
			d := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, deployKey, d)).Should(Succeed())
			Expect(d.Spec.Replicas).Should(Equal(deploy.Spec.Replicas))
		})

		It("Should idle the Deployment", func() {
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())

			ir.Spec.Idle = true
			Expect(k8sClient.Update(ctx, ir)).Should(Succeed())

			// We'll need to wait until the controller has idled the Deployment
			Eventually(func() (*int32, error) {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deployKey, d)
				if err != nil {
					return nil, err
				}
				return d.Spec.Replicas, nil
			}, timeout, interval).Should(Equal(pointer.Int32(0)))
		})
	})
})
