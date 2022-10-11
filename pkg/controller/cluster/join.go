package cluster

import (
	"captain/apis/cluster/v1alpha1"

	"github.com/karmada-io/karmada/pkg/karmadactl"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

func joinKarmada(hostConfig, clusterConfig *rest.Config, cluster *v1alpha1.Cluster) error {
	// kubectl karmada join cke-tst1 --kubeconfig=/root/.kube/karmada.config --karmada-context=karmada-apiserver --cluster-kubeconfig=/root/.kube/cke-tst1.config --cluster-context=kubernetes
	opts := joinOption(cluster)

	klog.V(1).Infof("joining cluster. cluster name: %s", opts.ClusterName)
	klog.V(1).Infof("joining cluster. cluster namespace: %s", opts.ClusterNamespace)

	return karmadactl.JoinCluster(hostConfig, clusterConfig, *opts)
}

func joinOption(cluster *v1alpha1.Cluster) *karmadactl.CommandJoinOption {
	opts := &karmadactl.CommandJoinOption{
		ClusterNamespace: "karmada-cluster",     // Namespace in the control plane where member cluster secrets are stored.
		ClusterProvider:  cluster.Spec.Provider, //  "Provider of the joining cluster. The Karmada scheduler can use this information to spread workloads across providers for higher availability.
		ClusterRegion:    "",                    // The region of the joining cluster. The Karmada scheduler can use this information to spread workloads across regions for higher availability.
		ClusterZone:      "",                    // The zone of the joining cluster
	}
	opts.ClusterName = cluster.Name
	if cluster.Labels != nil && len(cluster.Labels[v1alpha1.ClusterRegion]) > 0 {
		opts.ClusterRegion = cluster.Labels[v1alpha1.ClusterRegion]
	}
	return opts
}
