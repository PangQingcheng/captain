package persistentvolume

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"captain/pkg/bussiness/kube-resources/alpha1"
	"captain/pkg/unify/query"
	"captain/pkg/unify/response"
	"captain/pkg/utils/clusterclient"
)

type mcPersistentVolumeProvider struct {
	clusterclient.ClusterClients
}

func NewMCResProvider(clients clusterclient.ClusterClients) mcPersistentVolumeProvider {
	return mcPersistentVolumeProvider{ClusterClients: clients}
}

func (pd mcPersistentVolumeProvider) Get(region, cluster, namespace, name string) (runtime.Object, error) {
	cli, err := pd.GetClientSet(region, cluster)
	if err != nil {
		return nil, err
	}

	return cli.CoreV1().PersistentVolumes().Get(context.Background(), name, metav1.GetOptions{})
}

func (pd mcPersistentVolumeProvider) List(region, cluster, namespace string, query *query.QueryInfo) (*response.ListResult, error) {
	cli, err := pd.GetClientSet(region, cluster)
	if err != nil {
		return nil, err
	}
	list, err := cli.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{LabelSelector: query.LabelSelector})
	if err != nil {
		return nil, err
	}

	var result []runtime.Object
	if list != nil && list.Items != nil {
		for i := 0; i < len(list.Items); i++ {
			result = append(result, &list.Items[i])
		}
	}

	return alpha1.DefaultList(result, query, compareFunc, filter), nil
}
