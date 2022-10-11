package init

import (
	"context"
	"fmt"
	"os"
	"strings"

	"captain/apis/cluster/v1alpha1"
	"captain/pkg/controller/cluster/karmada/utils"
	"captain/pkg/simple/client/multicluster"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

func InstallKarmada(clusterConfig *rest.Config, cluster *v1alpha1.Cluster, options *multicluster.Options) ([]byte, string, error) {
	opts, err := initOptions(clusterConfig, options)
	if err != nil {
		return nil, "", err
	}

	return opts.RunInit()
}

func initOptions(clusterConfig *rest.Config, options *multicluster.Options) (*CommandInitOption, error) {
	i := &CommandInitOption{
		// kube image registry
		KubeImageMirrorCountry: "cn", // Country code of the kube image registry to be used. For Chinese mainland users, set it to cn
		// Kube image registry. For Chinese mainland users, you may use local gcr.io mirrors such as registry.cn-hangzhou.aliyuncs.com/google_containers to override default kube image registry
		KubeImageRegistry: options.Karmada.KubeImageRegistry,

		// Kubernetes
		Namespace: "karmada-system", //Kubernetes namespace

		// etcd
		EtcdStorageMode:          "hostPath", // etcd data storage mode(PVC, emptyDir, hostPath). value is PVC, specify --storage-classes-nam
		EtcdImage:                options.Karmada.EtcdImage,
		EtcdInitImage:            options.Karmada.EtcdInitImage,
		EtcdReplicas:             1,
		EtcdHostDataPath:         "/var/lib/karmada-etcd",
		EtcdNodeSelectorLabels:   "",    // update if need
		EtcdPersistentVolumeSize: "5Gi", // update if need, etcd data path,valid in pvc mode.

		// karmada
		CRDs:                               "/root/crds.tar.gz",                             // Karmada crds resource.(local file e.g. --crds /root/crds.tar.gz, or remote address: https://github.com/karmada-io/karmada/releases/download/v1.2.0/crds.tar.gz)
		KarmadaAPIServerNodePort:           32443,                                           // Karmada apiserver service node port
		KarmadaDataPath:                    "/etc/karmada",                                  // Karmada data path. kubeconfig cert and crds files
		KarmadaAPIServerImage:              "",                                              // Kubernetes apiserver image, default kube-apiserver:v1.24.2
		KarmadaAPIServerReplicas:           1,                                               // Karmada apiserver replica set
		KarmadaSchedulerImage:              options.Karmada.KarmadaSchedulerImage,           // Karmada scheduler image
		KarmadaSchedulerReplicas:           1,                                               // Karmada scheduler replica set
		KubeControllerManagerImage:         "",                                              // Kubernetes controller manager image
		KubeControllerManagerReplicas:      1,                                               // Karmada kube controller manager replica set
		KarmadaControllerManagerImage:      options.Karmada.KarmadaControllerManagerImage,   // Karmada controller manager image
		KarmadaControllerManagerReplicas:   1,                                               // Karmada controller manager replica set
		KarmadaWebhookImage:                options.Karmada.KarmadaWebhookImage,             // Karmada webhook image
		KarmadaWebhookReplicas:             1,                                               // Karmada webhook replica set
		KarmadaAggregatedAPIServerImage:    options.Karmada.KarmadaAggregatedAPIServerImage, // Karmada aggregated apiserver image
		KarmadaAggregatedAPIServerReplicas: 1,                                               // Karmada aggregated apiserver replica set
	}

	i.RestConfig = clusterConfig // needed？

	clientSet, err := utils.NewClientSet(clusterConfig)
	if err != nil {
		return nil, err
	}
	i.KubeClientSet = clientSet

	// 检测是否存在nodeport的service和KarmadaAPIServerNodePort冲突
	if !isNodePortExist(i) {
		return nil, fmt.Errorf("nodePort of karmada apiserver %v already exist", i.KarmadaAPIServerNodePort)

	}

	// 在etcd存储模式为hostPath且EtcdNodeSelectorLabels(用于选择节点部署etcd)为空时，自动选择node并设置nodeLable(karmada.io/etcd:"")
	// 此节点后续应用于部署etcd
	if i.EtcdStorageMode == "hostPath" && i.EtcdNodeSelectorLabels == "" {
		if err := i.AddNodeSelectorLabels(); err != nil {
			return nil, err
		}
	}

	// 通过lable(node-role.kubernetes.io/master)选择master节点
	// 若存在master节点，把所有master节点的ip添加至KarmadaAPIServerIP
	// 再随机选择三个节点的ip添加到KarmadaAPIServerIP
	// 如果最后KarmadaAPIServerIP为空，返回错误
	if err := getKubeMasterIP(i); err != nil {
		return nil, err
	}
	klog.Infof("karmada apiserver ip: %s", i.KarmadaAPIServerIP)

	//  在etcd存储模式为hostPath且EtcdNodeSelectorLabels不为空时，需要检测是否有对应的节点
	if i.EtcdStorageMode == "hostPath" && i.EtcdNodeSelectorLabels != "" {
		if !isNodeExist(i) {
			return nil, fmt.Errorf("no node found by label %s", i.EtcdNodeSelectorLabels)
		}
	}

	// 判断KarmadaDataPath是否存在，若是则需要删除
	// KarmadaDataPath ??
	if utils.IsExist(i.KarmadaDataPath) {
		if err := os.RemoveAll(i.KarmadaDataPath); err != nil {
			return nil, err
		}
	}

	return i, nil
}

func isNodePortExist(i *CommandInitOption) bool {
	svc, err := i.KubeClientSet.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Exit(err)
	}
	for _, v := range svc.Items {
		if v.Spec.Type != corev1.ServiceTypeNodePort {
			continue
		}
		if !nodePort(i.KarmadaAPIServerNodePort, v) {
			return false
		}
	}
	return true
}

func nodePort(nodePort int32, service corev1.Service) bool {
	for _, v := range service.Spec.Ports {
		if v.NodePort == nodePort {
			return false
		}
	}
	return true
}

func getKubeMasterIP(i *CommandInitOption) error {
	nodeClient := i.KubeClientSet.CoreV1().Nodes()
	masterNodes, err := nodeClient.List(context.TODO(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/master"})
	if err != nil {
		return err
	}

	if len(masterNodes.Items) == 0 {
		klog.Warning("the kubernetes cluster does not have a Master role.")
	} else {
		for _, v := range masterNodes.Items {
			i.KarmadaAPIServerIP = append(i.KarmadaAPIServerIP, utils.StringToNetIP(v.Status.Addresses[0].Address))
		}
		return nil
	}

	klog.Info("randomly select 3 Node IPs in the kubernetes cluster.")
	nodes, err := nodeClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for number := 0; number < 3; number++ {
		if number >= len(nodes.Items) {
			break
		}
		i.KarmadaAPIServerIP = append(i.KarmadaAPIServerIP, utils.StringToNetIP(nodes.Items[number].Status.Addresses[0].Address))
	}

	if len(i.KarmadaAPIServerIP) == 0 {
		return fmt.Errorf("karmada apiserver ip is empty")
	}
	return nil
}

func isNodeExist(i *CommandInitOption) bool {
	labels := i.EtcdNodeSelectorLabels
	l := strings.Split(labels, "=")
	node, err := i.KubeClientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: l[0]})
	if err != nil {
		return false
	}

	if len(node.Items) == 0 {
		return false
	}
	klog.Infof("Find the node [ %s ] by label %s", node.Items[0].Name, labels)
	return true
}
