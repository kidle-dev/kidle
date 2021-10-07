package pkg

import (
	"context"
	"fmt"
	"sort"
	"strings"

	kidlev1beta1 "github.com/kidle-dev/kidle/pkg/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KidleClient defines the client
type KidleClient struct {
	client.Client
	DiscoveryClient *discovery.DiscoveryClient
	Namespace       string
}

// NewKidleClient creates a kubernetes client for kidle.
// It can connect inside a k8s cluster from a pod into its current namespace
// or outside as a remote client on a specified namespace.
// If namespace == "", the namespace from the current context is used.
func NewKidleClient(namespace string) (*KidleClient, error) {
	var restConfig *rest.Config
	var err error

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	configOverrides := &clientcmd.ConfigOverrides{
		Context: clientcmdapi.Context{
			Namespace: namespace,
		},
	}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	currentNamespace, _, err := kubeConfig.Namespace()
	if err != nil {
		return nil, fmt.Errorf("error when getting current namespace: %v", err)
	}

	restConfig, err = kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error when creating restConfig: %v", err)
	}

	// Create a client with kidle scheme registered
	err = kidlev1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("error when adding kidle to Scheme: %v", err)
	}
	c, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("error when creating client: %v", err)
	}

	// Create a discoveryClient
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("error when creating discoveryclient: %v", err)
	}

	return &KidleClient{
		Client:          c,
		DiscoveryClient: discoveryClient,
		Namespace:       currentNamespace,
	}, nil
}

// AllowedGVK represents the allowed Group Version Kind
var AllowedGVK = []v1.GroupVersionKind{
	{
		Group:   "apps",
		Version: "v1",
		Kind:    "deployments",
	},
	{
		Group:   "apps",
		Version: "v1",
		Kind:    "statefulsets",
	},
	{
		Group:   "batch",
		Version: "v1beta1",
		Kind:    "cronjobs",
	},
	{
		Group:   "batch",
		Version: "v1",
		Kind:    "cronjobs",
	},
}

// GetAllowedResources gives the possible resources to create
// by calling the k8s discovery client
func (k *KidleClient) GetAllowedResources() (map[string]bool, error) {

	var allowedPrefixes []string

	// call the k8s discovery client to get allowed resources
	_, resourcesListSlice, err := k.DiscoveryClient.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	// build a list of allowed prefixes
	for _, resourceList := range resourcesListSlice {
		allowedPrefixes, err = getAllowedPrefixes(resourceList, allowedPrefixes)
		if err != nil {
			return nil, fmt.Errorf("unable to get allowed prefixes; %s", err.Error())
		}
	}

	// build the map of allowed prefixes
	allowedPrefixesMap := make(map[string]bool)
	for _, a := range allowedPrefixes {
		allowedPrefixesMap[a] = true
	}
	return allowedPrefixesMap, nil
}

func getAllowedPrefixes(resourceList *metav1.APIResourceList, allowedPrefixes []string) ([]string, error) {
	for _, gvk := range AllowedGVK {
		gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, err
		}

		if gvk.Group == gv.Group && gvk.Version == gv.Version {
			for _, resource := range resourceList.APIResources {
				if gvk.Kind == resource.Name {
					_, singular := meta.UnsafeGuessKindToResource(gv.WithKind(resource.Kind))
					allowedPrefixes = append(allowedPrefixes, resource.ShortNames...)
					allowedPrefixes = append(allowedPrefixes, singular.Resource)
					allowedPrefixes = append(allowedPrefixes, resource.Name)
				}
			}
		}
	}
	return allowedPrefixes, nil
}

// ApplyDesiredIdleState make sure that the referenced object has the proper idling state
func (k *KidleClient) ApplyDesiredIdleState(idle bool, req *client.ObjectKey) (bool, error) {

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

// CreateIdlingResource creates an IdlingResource with the given values
func (k *KidleClient) CreateIdlingResource(idle bool, ref string, req *client.ObjectKey) (bool, error) {
	ctx := context.Background()

	values := strings.Split(ref, "/")
	if len(values) < 2 {
		return false, fmt.Errorf("invalid idlingresource ref; expected <kind>/<name> got %s", ref)
	}

	// validate the passed kind
	resources, err := k.GetAllowedResources()
	if err != nil {
		return false, fmt.Errorf("unable to get allowed resources; %s", err.Error())
	}

	kind := values[0]
	ok := resources[kind]
	if !ok {
		return false, fmt.Errorf(fmt.Sprintf("invalid resource kind; got `%s` expected one of: %v", kind, printAllowedResources(resources)))
	}

	// build the IdlingResource
	ir := kidlev1beta1.IdlingResource{
		TypeMeta: metav1.TypeMeta{
			Kind: "IdlingResource",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: kidlev1beta1.IdlingResourceSpec{
			IdlingResourceRef: kidlev1beta1.CrossVersionObjectReference{
				Kind:       values[0],
				Name:       values[1],
				APIVersion: "apps/v1",
			},
			Idle: idle,
		},
	}

	// create the IdlingResource
	err = k.Create(ctx, &ir)
	if err != nil {
		return false, fmt.Errorf("unable to create idling resource: %s", err.Error())
	}
	return true, nil
}

// print the allowed resources in alphabetical order
func printAllowedResources(resources map[string]bool) string {
	var keys []string
	for k, _ := range resources {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := keys[0]
	for _, v := range keys[1:] {
		s += fmt.Sprintf(", %s", v)
	}
	return s
}
