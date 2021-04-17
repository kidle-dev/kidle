package main

import (
	"context"
	"fmt"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type KidleClient struct {
	kidlev1beta1.IdlingResourceInterface
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

	// create a restClient from config
	restClient, err := kidlev1beta1.NewRestClient(restConfig)
	if err != nil {
		return nil, fmt.Errorf("error when creating restClient: %v", err)
	}

	// create an IdlingResource client
	client := kidlev1beta1.NewIdlingResourceClient(restClient)

	return &KidleClient{IdlingResourceInterface: client}, nil
}

func (k *KidleClient) applyDesiredIdleState(idle bool, req *types.NamespacedName) (bool, error) {

	// get the IdlingResource from the req
	ir, err := k.Get(context.TODO(), req.Namespace, req.Name)
	if err != nil {
		return false, fmt.Errorf("unable to get idlingresource: %v", err)
	}

	// nothing to do if current state == desired state
	if ir.Spec.Idle == idle {
		return false, nil
	}

	// update idle flag to desired state
	ir.Spec.Idle = idle

	ir, err = k.Update(context.TODO(), ir)
	if err != nil {
		return false, fmt.Errorf("unable to update idlingresource: %v", err)
	}
	return true, nil
}
