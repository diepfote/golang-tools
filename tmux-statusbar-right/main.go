package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func getReader(filename string) (*bufio.Reader, *os.File) {
	file, _ := os.Open(filename)
	// file, error := os.Open(filename)
	// if error != nil {
	// 	fmt.Printf("file error: %v", error)
	// }
	reader := bufio.NewReader(file)

	return reader, file
}

func readContent(filename string) string {
	reader, file := getReader(filename)
	defer file.Close()

	bytes, _ := ioutil.ReadAll(reader)
	// bytes, error := ioutil.ReadAll(reader)
	// if error != nil {
	// 	fmt.Printf("read error: %v", error)
	// }

	return string(bytes)
}

func main() {
	openstackRegionName := readContent("/tmp/._openstack_cloud")
	re := regexp.MustCompile(`\r?\n`)
	// remove new lines
	openstackRegionName = re.ReplaceAllString(openstackRegionName, "")

	if len(openstackRegionName) > 0 {
		fmt.Printf(" :%s: ", openstackRegionName)
	}

	pulumiEnv := readContent("/tmp/._pulumi_env")
	// remove new lines
	pulumiEnv = re.ReplaceAllString(pulumiEnv, "")
	if len(pulumiEnv) > 0 {
		fmt.Printf("`%v` ", pulumiEnv)
	}

	env_vars := os.Environ()
	home := ""
	for _, env_var := range env_vars {
		// fmt.Printf("env_var: %v", env_var)
		if strings.HasPrefix(env_var, "HOME") {
			home = strings.Split(env_var, "=")[1]
		}
	}
	defaultKubernetesConfigFilename := home + "/.kube/config"

	kubernetesConfigFilenamePtr := readContent("/tmp/._kubeconfig")
	// remove new lines
	kubernetesConfigFilename := re.ReplaceAllString(kubernetesConfigFilenamePtr, "")

	if len(kubernetesConfigFilename) <= 0 {

		kubernetesConfigFilename = defaultKubernetesConfigFilename
	}

	kubernetesConfig := readConfigurationFile(kubernetesConfigFilename)

	if kubernetesConfig == nil {
		os.Exit(0)
	}
	if defaultKubernetesConfigFilename == kubernetesConfigFilename {
		// indicate KUBECONFIG variabl is empty
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

func reverse(s string) string {
	chars := []rune(s)
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
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
