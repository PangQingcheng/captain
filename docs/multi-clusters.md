# 多集群
## 接口
GET /capis/cluster.captain.io/v1alpha1/clusters\
GET /capis/cluster.captain.io/v1alpha1/clusters/{name}\
POST /capis/cluster.captain.io/v1alpha1/clusters\
DELETE /capis/cluster.captain.io/v1alpha1/clusters/{name}\
PUT /capis/cluster.captain.io/v1alpha1/clusters/{name}

## Cluster
```yaml
apiVersion: cluster.captain.io/v1alpha1
kind: Cluster
metadata:
  annotations:
    captainManaged.io/description: The description was created by Captain automatically.
      It is recommended that you use the Host Cluster to manage clusters only and
      deploy workloads on Member Clusters.
  creationTimestamp: "2022-10-09T07:11:46Z"
  finalizers:
  - finalizer.cluster.captain.io
  generation: 5
  labels:
    captain.io/managed: "true"
    cluster-role.captain.io/host: ""
  name: host
  resourceVersion: "9366214"
  uid: faded5d5-a3b2-462b-b006-954e92ab047c
spec:
  connection:
    kubeconfig: XXXX(base64)
    kubernetesAPIEndpoint: https://xxx.xx.xx.xx:6443
    type: direct
  enable: true
  provider: captain
status:
  conditions:
  - lastTransitionTime: "2022-10-09T07:15:46Z"
    lastUpdateTime: "2022-10-09T07:15:46Z"
    message: Cluster is available now
    reason: Ready
    status: "True"
    type: Ready
  kubernetesVersion: v1.23.6
  nodeCount: 7
```

## 多集群代理接口
/regions/{region}/cluster/{name}/...\
eg. 
```bash
curl http://127.0.0.1:9090/regions/wx-tst/clusters/cke-tst/api/v1/namespaces
```

## 注意
创建Cluster时：
+ cluster.Name添加前缀 {region}-， 如cluster1->xxtst-cluster1。
+ 添加region的label，cluster.captain.io/region: {region}，如cluster.captain.io/region: xxtst

*Why?*

不同region下可能存在同名的cluster，为了避免冲突，在存在region的情况下，使用region作为前缀。\
再此情况下为了能获取到正确的集群名称，同时需要在label中标明region字段，前端或代码中获取cluster名称的逻辑为当存在`cluster.captain.io/region` Label，且value不为空的情况下，集群名称为cluster.Name再去除{region}-前缀。label不存在或value为空直接返回cluster.Name

多集群功能应该允许region为空的情况，比如私有云单云池场景。所以也应该支持这种场景下对应的api路径，例如/clusters/{cluster}/...