package cluster

import (
	"captain/apis/cluster/v1alpha1"
	"time"

	"github.com/karmada-io/karmada/pkg/karmadactl"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

func unJoinKarmada(hostConfig, clusterConfig *rest.Config, cluster *v1alpha1.Cluster) error {
	// kubectl karmada join cke-tst1 --kubeconfig=/root/.kube/karmada.config --karmada-context=karmada-apiserver --cluster-kubeconfig=/root/.kube/cke-tst1.config --cluster-context=kubernetes
	opts := unJoinOption(cluster)
	klog.V(1).Infof("unjoining cluster. cluster name: %s", opts.ClusterName)
	klog.V(1).Infof("unjoining cluster. cluster namespace: %s", opts.ClusterNamespace)
	err := karmadactl.UnJoinCluster(hostConfig, clusterConfig, *opts)
	if !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func unJoinOption(cluster *v1alpha1.Cluster) *karmadactl.CommandUnjoinOption {
	opts := &karmadactl.CommandUnjoinOption{
		ClusterNamespace: "karmada-cluster",
		ClusterName:      cluster.Name,
		Wait:             60 * time.Second,
	}
	return opts
}
