package main

import (
	"context"
	"fmt"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type KidleClient struct {
	client.Client
}

func NewKidleClient(kubeconfig string) (*KidleClient, error) {
	var restConfig *rest.Config
	var err error

	// load restConfig from opts.Kubeconfig
	if kubeconfig == "" {
		logf.Log.V(0).Info("using in-cluster configuration")
		restConfig, err = rest.InClusterConfig()
	} else {
		logf.Log.V(0).Info("using configuration", "kubeconfig", kubeconfig)
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, fmt.Errorf("error when creating restConfig: %v", err)
	}

	kidlev1beta1.AddToScheme(scheme.Scheme)
	client, err := client.New(restConfig, client.Options{})
	return &KidleClient{Client: client}, nil
}

func (k *KidleClient) applyDesiredIdleState(idle bool, req *client.ObjectKey) (bool, error) {

	ctx := context.Background()

	// get the IdlingResource from the req
	ir := kidlev1beta1.IdlingResource{}
	err := k.Get(ctx, *req, &ir)
	if err != nil {
		return false, fmt.Errorf("unable to get idlingresource: %v", err)
	}

	// nothing to do if current state == desired state
	if ir.Spec.Idle == idle {
		return false, nil
	}

	// update idle flag to desired state
	ir.Spec.Idle = idle

	err = k.Update(ctx, &ir)
	if err != nil {
		return false, fmt.Errorf("unable to update idlingresource: %v", err)
	}
	return true, nil
}
