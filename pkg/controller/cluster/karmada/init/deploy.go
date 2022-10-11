package init

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"

	"captain/pkg/controller/cluster/karmada/utils"
	"captain/pkg/utils/cert"
	"captain/pkg/utils/iputil"

	"github.com/karmada-io/karmada/pkg/karmadactl/cmdinit/options"
	"k8s.io/klog"
)

var (
	imageRepositories = map[string]string{
		"global": "k8s.gcr.io",
		"cn":     "registry.cn-hangzhou.aliyuncs.com/google_containers",
	}

	certList = []string{
		options.CaCertAndKeyName,
		options.EtcdCaCertAndKeyName,
		options.EtcdServerCertAndKeyName,
		options.EtcdClientCertAndKeyName,
		options.KarmadaCertAndKeyName,
		options.ApiserverCertAndKeyName,
		options.FrontProxyCaCertAndKeyName,
		options.FrontProxyClientCertAndKeyName,
	}

	defaultKubeConfig = filepath.Join(homeDir(), ".kube", "config")

	defaultEtcdImage                  = "etcd:3.5.3-0"
	defaultKubeAPIServerImage         = "kube-apiserver:v1.24.2"
	defaultKubeControllerManagerImage = "kube-controller-manager:v1.24.2"
)

const (
	etcdStorageModePVC      = "PVC"
	etcdStorageModeEmptyDir = "emptyDir"
	etcdStorageModeHostPath = "hostPath"
)

// CommandInitOption holds all flags options for init.
type CommandInitOption struct {
	KubeImageRegistry                  string
	KubeImageMirrorCountry             string
	EtcdImage                          string
	EtcdReplicas                       int32
	EtcdInitImage                      string
	EtcdStorageMode                    string
	EtcdHostDataPath                   string
	EtcdNodeSelectorLabels             string
	EtcdPersistentVolumeSize           string
	KarmadaAPIServerImage              string
	KarmadaAPIServerReplicas           int32
	KarmadaAPIServerNodePort           int32
	KarmadaSchedulerImage              string
	KarmadaSchedulerReplicas           int32
	KubeControllerManagerImage         string
	KubeControllerManagerReplicas      int32
	KarmadaControllerManagerImage      string
	KarmadaControllerManagerReplicas   int32
	KarmadaWebhookImage                string
	KarmadaWebhookReplicas             int32
	KarmadaAggregatedAPIServerImage    string
	KarmadaAggregatedAPIServerReplicas int32
	Namespace                          string
	KubeConfig                         string
	Context                            string
	StorageClassesName                 string

	KarmadaDataPath string

	CRDs               string
	ExternalIP         string
	ExternalDNS        string
	KubeClientSet      *kubernetes.Clientset
	RestConfig         *rest.Config
	KarmadaAPIServerIP []net.IP
}

func (i *CommandInitOption) RunInit() (config []byte, tokenStr string, err error) {
	// generate certificate
	certs, err := i.genCerts()
	if err != nil {
		err = fmt.Errorf("certificate generation failed.%v", err)
		return
	}

	// prepare karmada CRD resources
	if err = i.prepareCRD(); err != nil {
		err = fmt.Errorf("prepare karmada failed.%v", err)
		return
	}

	// Create karmada kubeconfig
	config, err = i.createKarmadaConfig(certs)
	if err != nil {
		err = fmt.Errorf("create karmada kubeconfig failed.%v", err)
		return
	}

	// Create ns
	if err = i.CreateNamespace(); err != nil {
		err = fmt.Errorf("create namespace %s failed: %v", i.Namespace, err)
		return
	}

	// Create sa
	if err = i.CreateServiceAccount(); err != nil {
		return
	}

	// Create karmada-controller-manager ClusterRole and ClusterRoleBinding
	if err = i.CreateControllerManagerRBAC(); err != nil {
		return
	}

	// Create Secrets
	if err = i.createCertsSecrets(certs); err != nil {
		return
	}

	// install karmada-apiserver
	if err = i.initKarmadaAPIServer(); err != nil {
		return
	}

	// Create CRDs in karmada
	caBase64 := base64.StdEncoding.EncodeToString(certs.CA.Cert)
	if err = InitKarmadaResources(config, i.KarmadaDataPath, caBase64, i.Namespace); err != nil {
		return
	}

	// Create bootstarp token in karmada
	tokenStr, err = InitKarmadaBootstrapToken(config)
	if err != nil {
		return
	}

	// install karmada Component
	if err = i.initKarmadaComponent(); err != nil {
		return
	}

	return
}

// genCerts create ca etcd karmada cert
func (i *CommandInitOption) genCerts() (*KarmadaCerts, error) {
	notAfter := time.Now().Add(cert.Duration365d).UTC()

	var etcdServerCertDNS = []string{
		"localhost",
	}
	for number := int32(0); number < i.EtcdReplicas; number++ {
		etcdServerCertDNS = append(etcdServerCertDNS, fmt.Sprintf("%s-%v.%s.%s.svc.cluster.local", etcdStatefulSetAndServiceName, number, etcdStatefulSetAndServiceName, i.Namespace))
	}
	etcdServerAltNames := certutil.AltNames{
		DNSNames: etcdServerCertDNS,
		IPs:      []net.IP{utils.StringToNetIP("127.0.0.1")},
	}
	etcdServerCertConfig := cert.NewCertConfig("karmada-etcd-server", []string{}, etcdServerAltNames, &notAfter)
	etcdClientCertCfg := cert.NewCertConfig("karmada-etcd-client", []string{}, certutil.AltNames{}, &notAfter)

	var karmadaDNS = []string{
		"localhost",
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		karmadaAPIServerDeploymentAndServiceName,
		webhookDeploymentAndServiceAccountAndServiceName,
		karmadaAggregatedAPIServerDeploymentAndServiceName,
		fmt.Sprintf("%s.%s.svc.cluster.local", karmadaAPIServerDeploymentAndServiceName, i.Namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", webhookDeploymentAndServiceAccountAndServiceName, i.Namespace),
		fmt.Sprintf("%s.%s.svc", webhookDeploymentAndServiceAccountAndServiceName, i.Namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", karmadaAggregatedAPIServerDeploymentAndServiceName, i.Namespace),
		fmt.Sprintf("*.%s.svc.cluster.local", i.Namespace),
		fmt.Sprintf("*.%s.svc", i.Namespace),
	}
	karmadaDNS = append(karmadaDNS, utils.FlagsDNS(i.ExternalDNS)...)

	karmadaIPs := utils.FlagsIP(i.ExternalIP)
	karmadaIPs = append(
		karmadaIPs,
		utils.StringToNetIP("127.0.0.1"),
		utils.StringToNetIP("10.254.0.1"),
	)
	karmadaIPs = append(karmadaIPs, i.KarmadaAPIServerIP...)

	internetIP, err := utils.InternetIP()
	if err != nil {
		klog.Warningln("Failed to obtain internet IP. ", err)
	} else {
		karmadaIPs = append(karmadaIPs, internetIP)
	}

	karmadaAltNames := certutil.AltNames{
		DNSNames: karmadaDNS,
		IPs:      karmadaIPs,
	}
	karmadaCertCfg := cert.NewCertConfig("system:admin", []string{"system:masters"}, karmadaAltNames, &notAfter)
	apiserverCertCfg := cert.NewCertConfig("karmada-apiserver", []string{""}, karmadaAltNames, &notAfter)
	frontProxyClientCertCfg := cert.NewCertConfig("front-proxy-client", []string{}, certutil.AltNames{}, &notAfter)
	certs, err := genKarmadaCerts(etcdServerCertConfig, etcdClientCertCfg, karmadaCertCfg, apiserverCertCfg, frontProxyClientCertCfg)
	if err != nil {
		return nil, err
	}
	return certs, nil
}

// prepareCRD download or unzip `crds.tar.gz` to `options.DataPath`
func (i *CommandInitOption) prepareCRD() error {
	if strings.HasPrefix(i.CRDs, "http") {
		filename := i.KarmadaDataPath + "/" + path.Base(i.CRDs)
		klog.Infoln("download crds file name:", filename)
		if err := utils.DownloadFile(i.CRDs, filename); err != nil {
			return err
		}
		if err := utils.DeCompress(filename, i.KarmadaDataPath); err != nil {
			return err
		}
		return nil
	}
	klog.Infoln("local crds file name:", i.CRDs)
	return utils.DeCompress(i.CRDs, i.KarmadaDataPath)
}

func (i *CommandInitOption) createKarmadaConfig(certs *KarmadaCerts) ([]byte, error) {
	serverIP := i.KarmadaAPIServerIP[0].String()
	serverURL, err := generateServerURL(serverIP, i.KarmadaAPIServerNodePort)
	if err != nil {
		return nil, err
	}
	config, err := utils.GenKubeConfigFromSpec(serverURL, options.UserName, options.ClusterName, options.KarmadaKubeConfigName,
		certs.CA.Cert, certs.Karmada.Cert, certs.Karmada.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create karmada kubeconfig file. %v", err)
	}
	klog.Info("Create karmada kubeconfig success.")
	return config, nil
}

func generateServerURL(serverIP string, nodePort int32) (string, error) {
	_, ipType, err := iputil.ParseIP(serverIP)
	if err != nil {
		return "", err
	}
	if ipType == 4 {
		return fmt.Sprintf("https://%s:%v", serverIP, nodePort), nil
	}
	return fmt.Sprintf("https://[%s]:%v", serverIP, nodePort), nil
}

func (i *CommandInitOption) createCertsSecrets(certs *KarmadaCerts) error {
	// Create kubeconfig Secret
	karmadaServerURL := fmt.Sprintf("https://%s.%s.svc.cluster.local:%v", karmadaAPIServerDeploymentAndServiceName, i.Namespace, karmadaAPIServerContainerPort)
	config := utils.CreateWithCerts(karmadaServerURL, options.UserName, options.UserName, certs.CA.Cert, certs.Karmada.Key, certs.Karmada.Cert)
	configBytes, err := clientcmd.Write(*config)
	if err != nil {
		return fmt.Errorf("failure while serializing admin kubeConfig. %v", err)
	}

	kubeConfigSecret := i.SecretFromSpec(KubeConfigSecretAndMountName, corev1.SecretTypeOpaque, map[string]string{KubeConfigSecretAndMountName: string(configBytes)})
	if err = i.CreateSecret(kubeConfigSecret); err != nil {
		return err
	}
	// Create certs Secret
	etcdCert := map[string]string{
		fmt.Sprintf("%s.crt", options.EtcdCaCertAndKeyName):     string(certs.EtcdCa.Cert),
		fmt.Sprintf("%s.key", options.EtcdCaCertAndKeyName):     string(certs.EtcdCa.Key),
		fmt.Sprintf("%s.crt", options.EtcdServerCertAndKeyName): string(certs.EtcdServerCert.Cert),
		fmt.Sprintf("%s.key", options.EtcdServerCertAndKeyName): string(certs.EtcdServerCert.Key),
	}
	etcdSecret := i.SecretFromSpec(etcdCertName, corev1.SecretTypeOpaque, etcdCert)
	if err := i.CreateSecret(etcdSecret); err != nil {
		return err
	}

	karmadaCert := map[string]string{
		fmt.Sprintf("%s.crt", options.CaCertAndKeyName):               string(certs.CA.Cert),
		fmt.Sprintf("%s.key", options.CaCertAndKeyName):               string(certs.CA.Key),
		fmt.Sprintf("%s.crt", options.EtcdCaCertAndKeyName):           string(certs.EtcdCa.Cert),
		fmt.Sprintf("%s.key", options.EtcdCaCertAndKeyName):           string(certs.EtcdCa.Key),
		fmt.Sprintf("%s.crt", options.EtcdServerCertAndKeyName):       string(certs.EtcdServerCert.Cert),
		fmt.Sprintf("%s.key", options.EtcdServerCertAndKeyName):       string(certs.EtcdServerCert.Key),
		fmt.Sprintf("%s.crt", options.EtcdClientCertAndKeyName):       string(certs.EtcdClientCert.Cert),
		fmt.Sprintf("%s.key", options.EtcdClientCertAndKeyName):       string(certs.EtcdClientCert.Key),
		fmt.Sprintf("%s.crt", options.KarmadaCertAndKeyName):          string(certs.Karmada.Cert),
		fmt.Sprintf("%s.key", options.KarmadaCertAndKeyName):          string(certs.Karmada.Key),
		fmt.Sprintf("%s.crt", options.ApiserverCertAndKeyName):        string(certs.Apiserver.Cert),
		fmt.Sprintf("%s.key", options.ApiserverCertAndKeyName):        string(certs.Apiserver.Key),
		fmt.Sprintf("%s.crt", options.FrontProxyCaCertAndKeyName):     string(certs.FrontProxyCA.Cert),
		fmt.Sprintf("%s.key", options.FrontProxyCaCertAndKeyName):     string(certs.FrontProxyCA.Key),
		fmt.Sprintf("%s.crt", options.FrontProxyClientCertAndKeyName): string(certs.FrontProxyClient.Cert),
		fmt.Sprintf("%s.key", options.FrontProxyClientCertAndKeyName): string(certs.FrontProxyClient.Key),
	}

	karmadaSecret := i.SecretFromSpec(karmadaCertsName, corev1.SecretTypeOpaque, karmadaCert)
	if err := i.CreateSecret(karmadaSecret); err != nil {
		return err
	}

	karmadaWebhookCert := map[string]string{
		"tls.crt": string(certs.Karmada.Cert),
		"tls.key": string(certs.Karmada.Key),
	}
	karmadaWebhookSecret := i.SecretFromSpec(webhookCertsName, corev1.SecretTypeOpaque, karmadaWebhookCert)
	if err := i.CreateSecret(karmadaWebhookSecret); err != nil {
		return err
	}

	return nil
}

func (i *CommandInitOption) initKarmadaAPIServer() error {
	if err := i.CreateService(i.makeEtcdService(etcdStatefulSetAndServiceName)); err != nil {
		return err
	}
	klog.Info("create etcd StatefulSets")
	if _, err := i.KubeClientSet.AppsV1().StatefulSets(i.Namespace).Create(context.TODO(), i.makeETCDStatefulSet(), metav1.CreateOptions{}); err != nil {
		klog.Warning(err)
	}
	if err := WaitEtcdReplicasetInDesired(i.EtcdReplicas, i.KubeClientSet, i.Namespace, utils.MapToString(etcdLabels), 30); err != nil {
		klog.Warning(err)
	}
	if err := WaitPodReady(i.KubeClientSet, i.Namespace, utils.MapToString(etcdLabels), 30); err != nil {
		klog.Warning(err)
	}

	klog.Info("create karmada ApiServer Deployment")
	if err := i.CreateService(i.makeKarmadaAPIServerService()); err != nil {
		return err
	}
	if _, err := i.KubeClientSet.AppsV1().Deployments(i.Namespace).Create(context.TODO(), i.makeKarmadaAPIServerDeployment(), metav1.CreateOptions{}); err != nil {
		klog.Warning(err)
	}
	if err := WaitPodReady(i.KubeClientSet, i.Namespace, utils.MapToString(apiServerLabels), 120); err != nil {
		return err
	}

	// Create karmada-aggregated-apiserver
	// https://github.com/karmada-io/karmada/blob/master/artifacts/deploy/karmada-aggregated-apiserver.yaml
	klog.Info("create karmada aggregated apiserver Deployment")
	if err := i.CreateService(i.karmadaAggregatedAPIServerService()); err != nil {
		klog.Exitln(err)
	}
	if _, err := i.KubeClientSet.AppsV1().Deployments(i.Namespace).Create(context.TODO(), i.makeKarmadaAggregatedAPIServerDeployment(), metav1.CreateOptions{}); err != nil {
		klog.Warning(err)
	}
	if err := WaitPodReady(i.KubeClientSet, i.Namespace, utils.MapToString(aggregatedAPIServerLabels), 30); err != nil {
		klog.Warning(err)
	}
	return nil
}

func (i *CommandInitOption) initKarmadaComponent() error {
	// wait pod ready timeout 30s
	waitPodReadyTimeout := 30

	deploymentClient := i.KubeClientSet.AppsV1().Deployments(i.Namespace)
	// Create karmada-kube-controller-manager
	// https://github.com/karmada-io/karmada/blob/master/artifacts/deploy/kube-controller-manager.yaml
	klog.Info("create karmada kube controller manager Deployment")
	if err := i.CreateService(i.kubeControllerManagerService()); err != nil {
		klog.Exitln(err)
	}
	if _, err := deploymentClient.Create(context.TODO(), i.makeKarmadaKubeControllerManagerDeployment(), metav1.CreateOptions{}); err != nil {
		klog.Warning(err)
	}
	if err := WaitPodReady(i.KubeClientSet, i.Namespace, utils.MapToString(kubeControllerManagerLabels), waitPodReadyTimeout); err != nil {
		klog.Warning(err)
	}

	// Create karmada-scheduler
	// https://github.com/karmada-io/karmada/blob/master/artifacts/deploy/karmada-scheduler.yaml
	klog.Info("create karmada scheduler Deployment")
	if _, err := deploymentClient.Create(context.TODO(), i.makeKarmadaSchedulerDeployment(), metav1.CreateOptions{}); err != nil {
		klog.Warning(err)
	}
	if err := WaitPodReady(i.KubeClientSet, i.Namespace, utils.MapToString(schedulerLabels), waitPodReadyTimeout); err != nil {
		klog.Warning(err)
	}

	// Create karmada-controller-manager
	// https://github.com/karmada-io/karmada/blob/master/artifacts/deploy/karmada-controller-manager.yaml
	klog.Info("create karmada controller manager Deployment")
	if _, err := deploymentClient.Create(context.TODO(), i.makeKarmadaControllerManagerDeployment(), metav1.CreateOptions{}); err != nil {
		klog.Warning(err)
	}
	if err := WaitPodReady(i.KubeClientSet, i.Namespace, utils.MapToString(controllerManagerLabels), waitPodReadyTimeout); err != nil {
		klog.Warning(err)
	}

	// Create karmada-webhook
	// https://github.com/karmada-io/karmada/blob/master/artifacts/deploy/karmada-webhook.yaml
	klog.Info("create karmada webhook Deployment")
	if err := i.CreateService(i.karmadaWebhookService()); err != nil {
		klog.Exitln(err)
	}
	if _, err := deploymentClient.Create(context.TODO(), i.makeKarmadaWebhookDeployment(), metav1.CreateOptions{}); err != nil {
		klog.Warning(err)
	}
	if err := WaitPodReady(i.KubeClientSet, i.Namespace, utils.MapToString(webhookLabels), waitPodReadyTimeout); err != nil {
		klog.Warning(err)
	}
	return nil
}

// get kube components registry
func (i *CommandInitOption) kubeRegistry() string {
	registry := i.KubeImageRegistry
	mirrorCountry := strings.ToLower(i.KubeImageMirrorCountry)

	if registry != "" {
		return registry
	}

	if mirrorCountry != "" {
		value, ok := imageRepositories[mirrorCountry]
		if ok {
			return value
		}
	}

	return imageRepositories["global"]
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// get etcd image
func (i *CommandInitOption) etcdImage() string {
	if i.EtcdImage != "" {
		return i.EtcdImage
	}
	return i.kubeRegistry() + "/" + defaultEtcdImage
}

// get kube-apiserver image
func (i *CommandInitOption) kubeAPIServerImage() string {
	if i.KarmadaAPIServerImage != "" {
		return i.KarmadaAPIServerImage
	}
	return i.kubeRegistry() + "/" + defaultKubeAPIServerImage
}

// get kube-controller-manager image
func (i *CommandInitOption) kubeControllerManagerImage() string {
	if i.KubeControllerManagerImage != "" {
		return i.KubeControllerManagerImage
	}
	return i.kubeRegistry() + "/" + defaultKubeControllerManagerImage
}
