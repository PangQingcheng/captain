package main

import (
	"bytes"
	"captain/apis/cluster/v1alpha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var cluster = &v1alpha1.Cluster{}
var host = "http://127.0.0.1:9090"

func init() {
	cluster.APIVersion = "cluster.captain.io/v1alpha1"
	cluster.Kind = "Cluster"
	cluster.Name = "wx-tst-cke-tst"
	cluster.Labels = make(map[string]string)
	cluster.Labels[v1alpha1.ClusterRegion] = "wx-tst"
	cluster.Spec.Enable = true
	cluster.Spec.Provider = "CKE"
	cluster.Spec.Connection.Type = v1alpha1.ConnectionTypeDirect
	cluster.Spec.Connection.KubeConfig = []byte(kubeconfig)
	cluster.Spec.Connection.KubernetesAPIEndpoint = "https://10.125.176.23:44593"

	// cluster.Spec.MultiCluster.InstallKarmada = true
}

func main() {

	err := createCluster()
	if err != nil {
		panic(err)
	}
}

func get() error {
	resp, err := http.Get(host + "/capis/cluster.captain.io/v1alpha1/clusters/" + cluster.Name)
	if err != nil {
		return err
	}

	return handleResponse(resp)
}

func list() error {
	resp, err := http.Get(host + "/capis/cluster.captain.io/v1alpha1/clusters")
	if err != nil {
		return err
	}

	return handleResponse(resp)
}

func deleteCluster() error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodDelete, host+"/capis/cluster.captain.io/v1alpha1/clusters/"+cluster.Name, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	return handleResponse(resp)
}

func updateCluster() error {
	cluster.Spec.Connection.Type = v1alpha1.ConnectionTypeDirect
	cluster.Spec.Connection.KubernetesAPIEndpoint = ""
	data, err := json.Marshal(cluster)
	if err != nil {
		return err
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, host+"/capis/cluster.captain.io/v1alpha1/clusters/"+cluster.Name, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	return handleResponse(resp)
}

func createCluster() error {
	data, err := json.Marshal(cluster)
	if err != nil {
		return err
	}
	resp, err := http.Post(host+"/capis/cluster.captain.io/v1alpha1/clusters", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	return handleResponse(resp)
}

func handleResponse(resp *http.Response) error {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println("返回状态码：", resp.StatusCode)
	fmt.Println("返回内容：", string(data))

	return nil
}

var kubeconfig = `
apiVersion: v1
clusters:
  - cluster:
      insecure-skip-tls-verify: true
      server: https://10.125.176.23:44593
    name: kubernetes
contexts:
  - context:
      cluster: kubernetes
      user: admin
    name: kubernetes
current-context: kubernetes
kind: Config
preferences: {}
users:
  - name: admin
    user:
      token: 17fc67f87de25d0c5ca03ad7321f9d8f
`
