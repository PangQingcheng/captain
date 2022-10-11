package cluster

import (
	"context"

	"github.com/karmada-io/karmada/pkg/karmadactl"
	"github.com/karmada-io/karmada/pkg/karmadactl/cmdinit/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const namespace = "karmada-system"

func unInstallKarmada(clusterConfig *rest.Config) error {
	opts := deIniOtptions()

	var err error
	opts.KubeClientSet, err = utils.NewClientSet(clusterConfig)
	if err != nil {
		return err
	}

	if _, err := opts.KubeClientSet.CoreV1().Namespaces().Get(context.TODO(), opts.Namespace, metav1.GetOptions{}); err != nil {
		// not install
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return opts.Run()
}

func deIniOtptions() *karmadactl.CommandDeInitOption {
	opts := karmadactl.CommandDeInitOption{
		Namespace: "karmada-system", // "namespace where Karmada components are installed.
		Force:     true,             // "Reset cluster without prompting for confirmation.
	}
	return &opts
}
