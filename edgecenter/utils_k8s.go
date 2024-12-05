package edgecenter

// Temporary comment by CLOUDDEV-642
// func parseCIDRFromString(cidr string) (edgecloud.CIDR, error) {
//	var ecCIDR edgecloud.CIDR
//	_, netIPNet, err := net.ParseCIDR(cidr)
//	if err != nil {
//		return ecCIDR, err
//	}
//	ecCIDR.IP = netIPNet.IP
//	ecCIDR.Mask = netIPNet.Mask
//
//	return ecCIDR, nil
// }

type K8sConfig struct {
	APIVersion     string   `yaml:"apiVersion"`
	Kind           string   `yaml:"kind"`
	CurrentContext string   `yaml:"current-context"` // nolint: tagliatelle
	Preferences    struct{} `yaml:"preferences"`

	Clusters []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			CertificateAuthorityData string `yaml:"certificate-authority-data"` // nolint: tagliatelle
			Server                   string `yaml:"server"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`

	Contexts []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
			User    string `yaml:"user"`
		} `yaml:"context"`
	} `yaml:"contexts"`

	Users []struct {
		Name string `yaml:"name"`
		User struct {
			ClientCertificateData string `yaml:"client-certificate-data"` // nolint: tagliatelle
			ClientKeyData         string `yaml:"client-key-data"`         // nolint: tagliatelle
		} `yaml:"user"`
	} `yaml:"users"`
}

// Temporary comment by CLOUDDEV-642
// func parseK8sConfig(data string) (*K8sConfig, error) {
//	var config K8sConfig
//	err := yaml.Unmarshal([]byte(data), &config)
//	if err != nil {
//		return nil, err
//	}
//	return &config, nil
// }
