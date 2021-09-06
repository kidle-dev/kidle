package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kidlev1beta1 "github.com/kidle-dev/kidle/pkg/api/v1beta1"
	"github.com/kidle-dev/kidle/pkg/utils/k8s"
	"github.com/kidle-dev/kidle/pkg/utils/pointer"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("cronjob strategy", func() {
	const (
		timeout  = time.Second * 10
		//duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	var (
		ctx                  = context.Background()
		assertServiceAccount = func(types.NamespacedName) {}
		assertRole           = func(types.NamespacedName) {}
		assertRoleBinding    = func(types.NamespacedName) {}
		assertCronJob        = func(types.NamespacedName, string, string) {}
	)

	Context("Idling cronjob strategy suite", func() {
		var (
			irKey          = types.NamespacedName{Name: "ir-idling-cronjob-strategy", Namespace: "default"}
			cron           = "1 2 3 4 5"
			idlingStrategy = &kidlev1beta1.IdlingStrategy{
				CronStrategy: &kidlev1beta1.CronStrategy{
					Schedule: cron,
				},
			}
			idlingResource = newIdlingResource(irKey, &kidlev1beta1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "none",
				APIVersion: "apps/appsv1",
			})
		)
		idlingResource.Spec.IdlingStrategy = idlingStrategy

		It("Has created an IdlingResource object", func() {

			By("Creating the IdlingResource object")
			Expect(k8sClient.Create(ctx, idlingResource)).Should(Succeed())
		})

		It("Has created a service account", func() { assertServiceAccount(irKey) })
		It("Has created a role", func() { assertRole(irKey) })
		It("Has created a role binding", func() { assertRoleBinding(irKey) })
		It("Has created a cronjob", func() { assertCronJob(irKey, CommandIdle, cron) })

		XIt("Has removed cronjob strategy", func() {
			By("Removing the cronjob strategy")
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())
			ir.Spec.IdlingStrategy = nil
			Expect(k8sClient.Update(ctx, ir, &client.UpdateOptions{})).Should(Succeed())

			By("Has deleted the cronjob")
			cjKey := types.NamespacedName{Name: k8s.ToDNSName("kidle", irKey.Name, CommandIdle), Namespace: irKey.Namespace}
			Eventually(func() bool {
				cj := &batchv1beta1.CronJob{}
				err := k8sClient.Get(ctx, cjKey, cj)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Wakeup cronjob strategy suite", func() {
		var (
			irKey          = types.NamespacedName{Name: "ir-wakeup-cronjob-strategy", Namespace: "default"}
			cron           = "6 7 8 9 0"
			wakeupStrategy = &kidlev1beta1.WakeupStrategy{
				CronStrategy: &kidlev1beta1.CronStrategy{
					Schedule: cron,
				},
			}
			idlingResource = newIdlingResource(irKey, &kidlev1beta1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "none",
				APIVersion: "apps/appsv1",
			})
		)
		idlingResource.Spec.WakeupStrategy = wakeupStrategy

		It("Has created an IdlingResource object", func() {

			By("Creating the IdlingResource object")
			Expect(k8sClient.Create(ctx, idlingResource)).Should(Succeed())
		})

		It("Has created a service account", func() { assertServiceAccount(irKey) })
		It("Has created a role", func() { assertRole(irKey) })
		It("Has created a role binding", func() { assertRoleBinding(irKey) })
		It("Has created a cronjob", func() { assertCronJob(irKey, CommandWakeup, cron) })

		XIt("Has removed cronjob strategy", func() {
			By("Removing the cronjob strategy")
			ir := &kidlev1beta1.IdlingResource{}
			Expect(k8sClient.Get(ctx, irKey, ir)).Should(Succeed())
			ir.Spec.WakeupStrategy = nil
			Expect(k8sClient.Update(ctx, ir, &client.UpdateOptions{})).Should(Succeed())

			By("Has deleted the cronjob")
			cjKey := types.NamespacedName{Name: k8s.ToDNSName("kidle", irKey.Name, CommandWakeup), Namespace: irKey.Namespace}
			Eventually(func() bool {
				cj := &batchv1beta1.CronJob{}
				err := k8sClient.Get(ctx, cjKey, cj)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(Succeed())
		})
	})

	assertServiceAccount = func(irKey types.NamespacedName) {
		By("Validation of the service account creation")
		sa := &corev1.ServiceAccount{}
		saKey := types.NamespacedName{Name: k8s.ToDNSName("kidle", irKey.Name, "sa"), Namespace: irKey.Namespace}
		Eventually(func() error {
			return k8sClient.Get(ctx, saKey, sa)
		}, timeout, interval).Should(Succeed())

		By("Validation of the service account metadata")
		Expect(sa.ObjectMeta.Name).To(Equal(saKey.Name))
		Expect(sa.ObjectMeta.Namespace).To(Equal(saKey.Namespace))

		By("Validation of the service account owner reference")
		Expect(sa.ObjectMeta.OwnerReferences).To(HaveLen(1))
		Expect(sa.ObjectMeta.OwnerReferences[0].APIVersion).To(Equal("kidle.beroot.org/v1beta1"))
		Expect(sa.ObjectMeta.OwnerReferences[0].Kind).To(Equal("IdlingResource"))
		Expect(sa.ObjectMeta.OwnerReferences[0].Name).To(Equal(irKey.Name))
		Expect(sa.ObjectMeta.OwnerReferences[0].Controller).To(Equal(pointer.Bool(true)))
	}

	assertRole = func(irKey types.NamespacedName) {
		By("Validation of the role creation")
		role := &rbacv1.Role{}
		roleKey := types.NamespacedName{Name: k8s.ToDNSName("kidle", irKey.Name, "role"), Namespace: irKey.Namespace}
		Eventually(func() error {
			return k8sClient.Get(ctx, roleKey, role)
		}, timeout, interval).Should(Succeed())

		By("Validation of the role metadata")
		Expect(role.ObjectMeta.Name).To(Equal(roleKey.Name))
		Expect(role.ObjectMeta.Namespace).To(Equal(roleKey.Namespace))

		By("Validation of the role policy")
		Expect(role.Rules).To(HaveLen(1))
		Expect(role.Rules[0]).To(Equal(rbacv1.PolicyRule{
			Verbs:         []string{"get", "patch", "update"},
			APIGroups:     []string{"kidle.beroot.org"},
			Resources:     []string{"idlingresources"},
			ResourceNames: []string{irKey.Name},
		}))

		By("Validation of the role owner reference")
		Expect(role.ObjectMeta.OwnerReferences).NotTo(BeEmpty())
		Expect(role.ObjectMeta.OwnerReferences[0].APIVersion).To(Equal("kidle.beroot.org/v1beta1"))
		Expect(role.ObjectMeta.OwnerReferences[0].Kind).To(Equal("IdlingResource"))
		Expect(role.ObjectMeta.OwnerReferences[0].Name).To(Equal(irKey.Name))
		Expect(role.ObjectMeta.OwnerReferences[0].Controller).To(Equal(pointer.Bool(true)))
	}

	assertRoleBinding = func(irKey types.NamespacedName) {
		By("Validation of the role binding creation")
		rb := &rbacv1.RoleBinding{}
		rbKey := types.NamespacedName{Name: k8s.ToDNSName("kidle", irKey.Name, "rb"), Namespace: irKey.Namespace}
		Eventually(func() error {
			return k8sClient.Get(ctx, rbKey, rb)
		}, timeout, interval).Should(Succeed())

		By("Validation of the role binding metadata")
		Expect(rb.ObjectMeta.Name).To(Equal(rbKey.Name))
		Expect(rb.ObjectMeta.Namespace).To(Equal(rbKey.Namespace))

		By("Validation of the role binding subjects")
		Expect(rb.Subjects).To(HaveLen(1))
		Expect(rb.Subjects[0]).To(Equal(rbacv1.Subject{
			APIGroup: "",
			Kind:     "ServiceAccount",
			Name:     k8s.ToDNSName("kidle", irKey.Name, "sa"),
		}))

		By("Validation of the role binding role ref")
		Expect(rb.RoleRef).To(Equal(rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     k8s.ToDNSName("kidle", irKey.Name, "role"),
		}))

		By("Validation of the role binding owner reference")
		Expect(rb.ObjectMeta.OwnerReferences).NotTo(BeEmpty())
		Expect(rb.ObjectMeta.OwnerReferences[0].APIVersion).To(Equal("kidle.beroot.org/v1beta1"))
		Expect(rb.ObjectMeta.OwnerReferences[0].Kind).To(Equal("IdlingResource"))
		Expect(rb.ObjectMeta.OwnerReferences[0].Name).To(Equal(irKey.Name))
		Expect(rb.ObjectMeta.OwnerReferences[0].Controller).To(Equal(pointer.Bool(true)))
	}

	assertCronJob = func(irKey types.NamespacedName, command string, cron string) {
		By("Validation of the cronjob creation")
		cj := &batchv1beta1.CronJob{}
		cjKey := types.NamespacedName{Name: k8s.ToDNSName("kidle", irKey.Name, command), Namespace: irKey.Namespace}
		Eventually(func() error {
			return k8sClient.Get(ctx, cjKey, cj)
		}, timeout, interval).Should(Succeed())

		By("Validation of the cronjob metadata")
		Expect(cj.ObjectMeta.Name).To(Equal(cjKey.Name))
		Expect(cj.ObjectMeta.Namespace).To(Equal(cjKey.Namespace))

		By("Validation of the cronjob spec")
		Expect(cj.Spec.Suspend).To(Equal(pointer.Bool(false)))
		Expect(cj.Spec.Schedule).To(Equal(cron))

		By("Validation of the cronjob job spec")
		Expect(cj.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName).To(Equal(k8s.ToDNSName("kidle", irKey.Name, "sa")))
		Expect(cj.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy).To(Equal(corev1.RestartPolicyOnFailure))

		Expect(cj.Spec.JobTemplate.Spec.Template.Spec.Containers).To(HaveLen(1))
		containers := k8s.ContainersToMap(cj.Spec.JobTemplate.Spec.Template.Spec.Containers)
		Expect(containers).To(HaveKey(CronJobContainerName))
		c := containers[CronJobContainerName]
		Expect(c.Image).To(Equal(KidlectlImage))
		Expect(c.Args).To(HaveLen(2))
		Expect(c.Args).To(ContainElements(command, irKey.Name))

		By("Validation of the cronjob owner reference")
		Expect(cj.ObjectMeta.OwnerReferences).NotTo(BeEmpty())
		Expect(cj.ObjectMeta.OwnerReferences[0].APIVersion).To(Equal("kidle.beroot.org/v1beta1"))
		Expect(cj.ObjectMeta.OwnerReferences[0].Kind).To(Equal("IdlingResource"))
		Expect(cj.ObjectMeta.OwnerReferences[0].Name).To(Equal(irKey.Name))
		Expect(cj.ObjectMeta.OwnerReferences[0].Controller).To(Equal(pointer.Bool(true)))
	}
})
