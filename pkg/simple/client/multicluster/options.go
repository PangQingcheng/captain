package multicluster

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	DefaultResyncPeriod    = 120 * time.Second
	DefaultHostClusterName = "host"
)

type Options struct {
	// Enable
	Enable bool `json:"enable"`

	// ClusterControllerResyncPeriod is the resync period used by cluster controller.
	ClusterControllerResyncPeriod time.Duration `json:"clusterControllerResyncPeriod,omitempty" yaml:"clusterControllerResyncPeriod"`

	// HostClusterName is the name of the control plane cluster, default set to host.
	HostClusterName string `json:"hostClusterName,omitempty" yaml:"hostClusterName"`

	Karmada KarmadaConfig `json:"karmada,omitempty" yaml:"karmada"`
}

type KarmadaConfig struct {
	KubeImageRegistry               string `json:"kubeImageRegistry,omitempty" yaml:"kubeImageRegistry"`
	EtcdImage                       string `json:"etcdImage,omitempty" yaml:"etcdImage"`
	EtcdInitImage                   string `json:"etcdInitImage,omitempty" yaml:"etcdInitImage"`
	KarmadaSchedulerImage           string `json:"karmadaSchedulerImage,omitempty" yaml:"karmadaSchedulerImage"`
	KarmadaControllerManagerImage   string `json:"karmadaControllerManagerImage,omitempty" yaml:"karmadaControllerManagerImage"`
	KarmadaWebhookImage             string `json:"karmadaWebhookImage,omitempty" yaml:"karmadaWebhookImage"`
	KarmadaAggregatedAPIServerImage string `json:"karmadaAggregatedAPIServerImage,omitempty" yaml:"karmadaAggregatedAPIServerImage"`
}

// NewOptions returns a default nil options
func NewOptions() *Options {
	return &Options{
		Enable:                        false,
		ClusterControllerResyncPeriod: DefaultResyncPeriod,
		HostClusterName:               DefaultHostClusterName,
	}
}

func (o *Options) Validate() []error {
	var err []error

	res := validation.IsQualifiedName(o.HostClusterName)
	if len(res) == 0 {
		return err
	}
	err = append(err, errors.New("failed to create the host cluster because of invalid cluster name"))
	for _, str := range res {
		err = append(err, errors.New(str))
	}
	return err
}

func (o *Options) AddFlags(fs *pflag.FlagSet, s *Options) {
	fs.BoolVar(&o.Enable, "multiple-clusters", s.Enable, ""+
		"This field instructs Captain to enter multiple-cluster mode or not.")

	fs.DurationVar(&o.ClusterControllerResyncPeriod, "cluster-controller-resync-period", s.ClusterControllerResyncPeriod,
		"Cluster controller resync period to sync cluster resource. e.g. 2m 5m 10m ... default set to 2m")

	fs.StringVar(&o.HostClusterName, "host-cluster-name", s.HostClusterName, "the name of the control plane"+
		" cluster, default set to host")
}
