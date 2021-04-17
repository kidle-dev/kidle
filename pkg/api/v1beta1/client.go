package v1beta1

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func NewRestClient(cfg *rest.Config) (*rest.RESTClient, error) {
	_ = AddToScheme(scheme.Scheme)
	cfg.ContentConfig.GroupVersion = &GroupVersion
	cfg.APIPath = "/apis"
	cfg.ContentType = runtime.ContentTypeJSON
	cfg.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	cfg.UserAgent = rest.DefaultKubernetesUserAgent()

	return rest.RESTClientFor(cfg)
}

type IdlingResourceInterface interface {
	Get(ctx context.Context, namespace, name string) (*IdlingResource, error)
	Create(ctx context.Context, obj *IdlingResource) (*IdlingResource, error)
	Update(ctx context.Context, obj *IdlingResource) (*IdlingResource, error)
	Delete(ctx context.Context, obj *IdlingResource, options *metav1.DeleteOptions) error
}

type IdlingResourceV1Beta1Client struct {
	restClient *rest.RESTClient
}

func NewIdlingResourceClient(restClient *rest.RESTClient) IdlingResourceInterface {
	return &IdlingResourceV1Beta1Client{
		restClient: restClient,
	}
}

func (i *IdlingResourceV1Beta1Client) Get(ctx context.Context, namespace string, name string) (*IdlingResource, error) {
	ir := &IdlingResource{}
	err := i.restClient.
		Get().
		Namespace(namespace).
		Resource(IdlingResources).
		Name(name).
		Do(ctx).
		Into(ir)
	return ir, err
}

func (i *IdlingResourceV1Beta1Client) Create(ctx context.Context, obj *IdlingResource) (*IdlingResource, error) {
	ir := &IdlingResource{}
	err := i.restClient.
		Post().
		Namespace(obj.Namespace).
		Resource(IdlingResources).
		Body(obj).
		Do(ctx).
		Into(ir)
	return ir, err
}

func (i *IdlingResourceV1Beta1Client) Update(ctx context.Context, obj *IdlingResource) (*IdlingResource, error) {
	ir := &IdlingResource{}
	err := i.restClient.
		Put().
		Namespace(obj.Namespace).
		Resource(IdlingResources).
		Name(obj.Name).
		Body(obj).
		Do(ctx).
		Into(ir)
	return ir, err
}

func (i *IdlingResourceV1Beta1Client) Delete(ctx context.Context, obj *IdlingResource, options *metav1.DeleteOptions) error {
	return i.restClient.
		Delete().
		Namespace(obj.Namespace).
		Resource(IdlingResources).
		Name(obj.Name).
		Body(options).
		Do(ctx).
		Error()
}
