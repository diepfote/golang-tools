package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	errnoKubeConfigEmpty = 10
	errnoDefault         = 1
	errnoNoError         = 0
)

func main() {
	openstackRegionName := read("/tmp/._openstack_cloud")
	re := regexp.MustCompile(`\r?\n`)
	// remove new lines
	openstackRegionName = re.ReplaceAllString(openstackRegionName, "")

	if len(openstackRegionName) > 0 {
		fmt.Printf(" :%s: ", openstackRegionName)
	}

	env_vars := os.Environ()
	home := ""
	for _, env_var := range env_vars {
		// fmt.Printf("env_var: %v", env_var)
		if strings.HasPrefix(env_var, "HOME=") {
			home = strings.Split(env_var, "=")[1]
		}
	}
	defaultKubernetesConfigFilename := home + "/.kube/config"

	kubernetesConfigFilenamePtr := read("/tmp/._kubeconfig")
	// remove new lines
	kubernetesConfigFilename := re.ReplaceAllString(kubernetesConfigFilenamePtr, "")

	if len(kubernetesConfigFilename) <= 0 {

		kubernetesConfigFilename = defaultKubernetesConfigFilename
	}
	if strings.Contains(kubernetesConfigFilename, ":") {
		// indicate more than one file referenced in KUBECONFIG env var
		fmt.Printf("KUBECONFIG+! ")
		os.Exit(errnoNoError)
	}

	kubernetesConfig := readConfigurationFile(kubernetesConfigFilename)

	if kubernetesConfig == nil {
		os.Exit(errnoKubeConfigEmpty)
	}
	if defaultKubernetesConfigFilename == kubernetesConfigFilename {
		// indicate KUBECONFIG variable is empty
		fmt.Printf("KUBECONFIG= ")
	}

	namespace := ""
	// kubernetesConfig.CurrentContext
	for _, context := range kubernetesConfig.Contexts {

		if context.Name == kubernetesConfig.CurrentContext {
			// DEBUG
			// fmt.Printf("%+v", context.Context.Namespace)
			namespace = context.Context.Namespace
		}
	}

	error, _, region := unpackSplit(kubernetesConfig.CurrentContext, "/")
	if error != nil {
		region = kubernetesConfig.CurrentContext
	} else {
		region = reverse(strings.SplitN(reverse(region), "-", 6)[5]) + "-" +
			strings.SplitN(region, "-", 8)[6]
	}

	if len(region) > 0 || len(namespace) > 0 {
		fmt.Printf("(%v) >%v< ", region, namespace)
	}
}

func unpackSplit(s, sep string) (error, string, string) {
	x := strings.SplitN(s, sep, 2)
	if len(x) > 1 {
		return nil, x[0], x[1]
	} else {
		return errors.New("unpackSplit: failed"), x[0], x[0]
	}
}

func readConfigurationFile(filePath string) *KubernetesConfig {
	// Read configuration file
	configurationYaml, err := ioutil.ReadFile(filePath)

	if err != nil || len(configurationYaml) <= 0 {
		// ignore error
		return nil
	}

	// Unmarshal yaml
	var configuration KubernetesConfig
	yaml.Unmarshal(configurationYaml, &configuration)

	return &configuration
}

type KubernetesConfig struct {
	APIVersion string `yaml:"apiVersion"`
	// Clusters   []struct {
	// 	Cluster struct {
	// 		Server string `yaml:"server"`
	// 	} `yaml:"cluster"`
	// 	Name string `yaml:"name"`
	// } `yaml:"clusters"`
	Contexts []struct {
		Context struct {
			Cluster   string `yaml:"cluster"`
			Namespace string `yaml:"namespace"`
			User      string `yaml:"user"`
		} `yaml:"context"`
		Name string `yaml:"name"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
	// Kind           string `yaml:"kind"`
	// Preferences    struct {
	// } `yaml:"preferences"`
	// Users []struct {
	// 	Name string `yaml:"name"`
	// 	User struct {
	// 		Token string `yaml:"token"`
	// 	} `yaml:"user"`
	// } `yaml:"users"`
}
