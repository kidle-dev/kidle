package idler

import (
	"context"
	"github.com/go-logr/logr"
	kidlev1beta1 "github.com/kidle-dev/kidle/pkg/api/v1beta1"
	"github.com/kidle-dev/kidle/pkg/utils/k8s"
	"github.com/kidle-dev/kidle/pkg/utils/pointer"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CronJobIdler struct {
	client.Client
	Log     logr.Logger
	CronJob *batchv1beta1.CronJob
	ObjectIdler
}

func NewCronJobIdler(client client.Client, log logr.Logger, cronjob *batchv1beta1.CronJob) *CronJobIdler {
	return &CronJobIdler{
		Client:      client,
		Log:         log,
		CronJob:     cronjob,
		ObjectIdler: NewObjectIdler(client, log, cronjob),
	}
}

func (i *CronJobIdler) NeedIdle(instance *kidlev1beta1.IdlingResource) bool {
	return instance.Spec.Idle && !*i.CronJob.Spec.Suspend
}

func (i *CronJobIdler) NeedWakeup(instance *kidlev1beta1.IdlingResource) bool {
	return !instance.Spec.Idle && *i.CronJob.Spec.Suspend
}

func (i *CronJobIdler) Idle(ctx context.Context) error {
	if !*i.CronJob.Spec.Suspend {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := i.Get(ctx, types.NamespacedName{Namespace: i.CronJob.Namespace, Name: i.CronJob.Name}, i.CronJob); err != nil {
				i.Log.Error(err, "unable to get cronjob","name", i.CronJob.Name)
				return err
			}
			k8s.AddAnnotation(&i.CronJob.ObjectMeta, kidlev1beta1.MetadataExpectedState, "true")
			i.CronJob.Spec.Suspend = pointer.Bool(true)
			return i.Update(ctx, i.CronJob)
		})
		if err != nil {
			i.Log.Error(err, "unable to suspend cronjob", "name", i.CronJob.Name)
			return err
		}
		i.Log.V(1).Info("cronjob suspended", "name", i.CronJob.Name)
	} else {
		i.Log.V(2).Info("cronjob already suspended", "name", i.CronJob.Name)
	}
	return nil
}

func (i *CronJobIdler) Wakeup(ctx context.Context) (*int32, error) {
	if *i.CronJob.Spec.Suspend {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := i.Get(ctx, types.NamespacedName{Namespace: i.CronJob.Namespace, Name: i.CronJob.Name}, i.CronJob); err != nil {
				i.Log.Error(err, "unable to get cronjob","name", i.CronJob.Name)
				return err
			}
			k8s.AddAnnotation(&i.CronJob.ObjectMeta, kidlev1beta1.MetadataExpectedState, "false")
			i.CronJob.Spec.Suspend = pointer.Bool(false)
			return i.Update(ctx, i.CronJob)
		})
		if err != nil {
			i.Log.Error(err, "unable to wakeup cronjob", "name", i.CronJob.Name)
			return nil, err
		}
		i.Log.V(1).Info("cronjob woke up", "name", i.CronJob.Name)
	} else {
		i.Log.V(2).Info("cronjob already woke up", "name", i.CronJob.Name)
	}
	return nil, nil
}
