package init

import (
	"captain/pkg/utils/cert"

	"github.com/karmada-io/karmada/pkg/karmadactl/cmdinit/options"
)

type KarmadaCerts struct {
	CA               *cert.CertPair
	Karmada          *cert.CertPair
	Apiserver        *cert.CertPair
	FrontProxyCA     *cert.CertPair
	FrontProxyClient *cert.CertPair
	EtcdCa           *cert.CertPair
	EtcdServerCert   *cert.CertPair
	EtcdClientCert   *cert.CertPair
}

// genKarmadaCerts Create CA certificate and sign etcd karmada certificate.
func genKarmadaCerts(etcdServerCertCfg, etcdClientCertCfg, karmadaCertCfg, apiserverCertCfg, frontProxyClientCertCfg *cert.CertsConfig) (*KarmadaCerts, error) {
	certs := &KarmadaCerts{}
	caCert, caKey, err := cert.NewCACertAndKey("karmada")
	if err != nil {
		return nil, err
	}
	if certs.CA, err = cert.EncodeCertAndKey(options.CaCertAndKeyName, caCert, caKey); err != nil {
		return nil, err
	}

	karmadaCert, karmadaKey, err := cert.NewCertAndKey(caCert, *caKey, karmadaCertCfg)
	if err != nil {
		return nil, err
	}
	if certs.Karmada, err = cert.EncodeCertAndKey(options.KarmadaCertAndKeyName, karmadaCert, &karmadaKey); err != nil {
		return nil, err
	}

	apiserverCert, apiserverKey, err := cert.NewCertAndKey(caCert, *caKey, apiserverCertCfg)
	if err != nil {
		return nil, err
	}
	if certs.Apiserver, err = cert.EncodeCertAndKey(options.ApiserverCertAndKeyName, apiserverCert, &apiserverKey); err != nil {
		return nil, err
	}

	frontProxyCaCert, frontProxyCaKey, err := cert.NewCACertAndKey("front-proxy-ca")
	if err != nil {
		return nil, err
	}
	if certs.FrontProxyCA, err = cert.EncodeCertAndKey(options.FrontProxyCaCertAndKeyName, frontProxyCaCert, frontProxyCaKey); err != nil {
		return nil, err
	}

	frontProxyClientCert, frontProxyClientKey, err := cert.NewCertAndKey(frontProxyCaCert, *frontProxyCaKey, frontProxyClientCertCfg)
	if err != nil {
		return nil, err
	}
	if certs.FrontProxyClient, err = cert.EncodeCertAndKey(options.FrontProxyClientCertAndKeyName, frontProxyClientCert, &frontProxyClientKey); err != nil {
		return nil, err
	}

	return genEtcdCerts(certs, etcdServerCertCfg, etcdClientCertCfg)
}

func genEtcdCerts(certs *KarmadaCerts, etcdServerCertCfg, etcdClientCertCfg *cert.CertsConfig) (*KarmadaCerts, error) {
	etcdCaCert, etcdCaKey, err := cert.NewCACertAndKey("etcd-ca")
	if err != nil {
		return nil, err
	}
	if certs.EtcdCa, err = cert.EncodeCertAndKey(options.EtcdCaCertAndKeyName, etcdCaCert, etcdCaKey); err != nil {
		return nil, err
	}

	etcdServerCert, etcdServerKey, err := cert.NewCertAndKey(etcdCaCert, *etcdCaKey, etcdServerCertCfg)
	if err != nil {
		return nil, err
	}
	if certs.EtcdServerCert, err = cert.EncodeCertAndKey(options.EtcdServerCertAndKeyName, etcdServerCert, &etcdServerKey); err != nil {
		return nil, err
	}

	etcdClientCert, etcdClientKey, err := cert.NewCertAndKey(etcdCaCert, *etcdCaKey, etcdClientCertCfg)
	if err != nil {
		return nil, err
	}
	if certs.EtcdClientCert, err = cert.EncodeCertAndKey(options.EtcdClientCertAndKeyName, etcdClientCert, &etcdClientKey); err != nil {
		return nil, err
	}
	return certs, nil
}
