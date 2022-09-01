package resource

import (
	"captain/pkg/bussiness/kube-resources/alpha1"
	"captain/pkg/bussiness/kube-resources/alpha1/deployment"
	"captain/pkg/unify/query"
	"captain/pkg/unify/response"
	"errors"

	"captain/pkg/informers"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

var (
	DeploymentGVR           = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployment"}
	ErrResourceNotSupported = errors.New("resource is not supported")
)

//ResourceProcessor ... processing resources including kube-native, sevice mesh , others kinds of cloud-native resources
type ResourceProcessor struct {
	clusterResourceProcessors    map[schema.GroupVersionResource]alpha1.KubeResProvider
	namespacedResourceProcessors map[schema.GroupVersionResource]alpha1.KubeResProvider
}

func NewResourceProcessor(factory informers.CapInformerFactory, cache cache.Cache) *ResourceProcessor {
	namespacedResourceProcessors := make(map[schema.GroupVersionResource]alpha1.KubeResProvider)
	clusterResourceProcessors := make(map[schema.GroupVersionResource]alpha1.KubeResProvider)

	//native kube resources
	namespacedResourceProcessors[DeploymentGVR] = deployment.New(factory.KubernetesSharedInformerFactory())

	return &ResourceProcessor{
		namespacedResourceProcessors: namespacedResourceProcessors,
		clusterResourceProcessors:    clusterResourceProcessors,
	}
}

// TryResource will retrieve a getter with resource name, it doesn't guarantee find resource with correct group version
// need to refactor this use schema.GroupVersionResource
func (r *ResourceProcessor) TryResource(clusterScope bool, resource string) alpha1.KubeResProvider {
	if clusterScope {
		for k, v := range r.clusterResourceProcessors {
			if k.Resource == resource {
				return v
			}
		}
	}
	for k, v := range r.namespacedResourceProcessors {
		if k.Resource == resource {
			return v
		}
	}
	return nil
}

func (r *ResourceProcessor) Get(resource, namespace, name string) (runtime.Object, error) {
	clusterScope := namespace == ""
	getter := r.TryResource(clusterScope, resource)
	if getter == nil {
		return nil, ErrResourceNotSupported
	}
	return getter.Get(namespace, name)
}

func (r *ResourceProcessor) List(resource, namespace string, query *query.QueryInfo) (*response.ListResult, error) {
	// parse cluster scope or not
	clusterScope := namespace == ""

	provider := r.TryResource(clusterScope, resource)
	if provider == nil {
		return nil, ErrResourceNotSupported
	}
	return provider.List(namespace, query)
}
