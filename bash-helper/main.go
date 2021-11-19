package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func getReader(filename string) (*bufio.Reader, *os.File) {
	file, error := os.Open(filename)
	if error != nil {
		fmt.Printf("file error: %v", error)
		return nil, nil
	}
	reader := bufio.NewReader(file)

	return reader, file
}

func read(filename string) []byte {
	reader, file := getReader(filename)
	if reader == nil {
		fmt.Printf("no reader")
		return nil
	}
	defer file.Close()

	bytes, error := ioutil.ReadAll(reader)
	if error != nil {
		fmt.Printf("read error: %v", error)
		return nil
	}

	// fmt.Printf("[THIS] %v", string(bytes))
	return bytes
}

func removeTrailingWhiteSpace(home string, inContainer string) {
	filename := home

	if len(inContainer) > 0 {
		filename += "/.container"
	}
	filename += "/.bash_history"

	content := read(filename)
	if content == nil {
		return
	}

	re := regexp.MustCompile(`(?m)[\t ]+$`)
	content = re.ReplaceAll(content, []byte(""))

	// fmt.Printf("[ſðđæ] %v", string(content))
	err := ioutil.WriteFile(filename, content, 0600)
	if err != nil {
		fmt.Printf("%v", err)
		return // no need to refresh tmux
	}
}

// Turns $HOME/Documents/golang/tools into
//       ~/D/golang/tools
// and leaves /usr/local/bin etc. as is
func printShortenedPath(path string, home string, color string,
	noColor string, optionals ...string) {

	pathSplit := strings.Split(path, "/")
	prefix := ""
	var inContainer string

	if len(optionals) > 0 {
		inContainer = optionals[0]

		if len(inContainer) > 0 {
			prefix += "IN_CONTAINER: "
		}
	}

	if strings.HasPrefix(path, home) {
		prefix += "~/"

		if strings.Compare(path, home) == 0 {
			pathSplit = make([]string, 0)
		} else if len(pathSplit) >= 3 {
			pathSplit = pathSplit[3:]
		}
	}

	for index, element := range pathSplit {
		if index == 1 {
			// do not shorten first dir below root
			continue
		}
		if index == len(pathSplit)-2 {
			// do not shorten directory above CWD
			continue
		}

		if index == len(pathSplit)-1 {
			break
		}

		if len(element) > 1 {
			if strings.HasPrefix(element, ".") {
				pathSplit[index] = element[0:2]
			} else {
				pathSplit[index] = element[0:1]
			}
		}
	}

	fmt.Printf("%v%v%v%v", color, prefix, strings.Join(pathSplit, "/"), noColor)
}

func updateOpenStackAndKubeConfigInfo(osCloud string, kubeConfig string) {

	var err error

	err = ioutil.WriteFile("/tmp/._openstack_cloud", []byte(osCloud), 0600)
	if err != nil {
		fmt.Printf("%v", err)
		return // no need to refresh tmux
	}

	err = ioutil.WriteFile("/tmp/._kubeconfig", []byte(kubeConfig), 0600)
	if err != nil {
		fmt.Printf("%v", err)
		return // no need to refresh tmux
	}

	cmd := exec.Command("tmux", "refresh-client")
	cmd.Start()
}

func main() {

	env_vars := os.Environ()
	pwd := ""
	virtualEnv := ""
	home := ""
	inContainer := ""
	osCloud := ""
	kubeConfig := ""
	green := ""
	blue := ""
	noColor := ""

	for _, env_var := range env_vars {
		// fmt.Printf("env_var: %v", env_var)
		switch {
			case strings.HasPrefix(env_var, "PWD"): pwd = strings.Split(env_var, "=")[1]
			case strings.HasPrefix(env_var, "HOME"): home = strings.Split(env_var, "=")[1]
			case strings.HasPrefix(env_var, "IN_CONTAINER"): inContainer = strings.Split(env_var, "=")[1]
			case strings.HasPrefix(env_var, "OS_CLOUD"): osCloud = strings.Split(env_var, "=")[1]
			case strings.HasPrefix(env_var, "KUBECONFIG"): kubeConfig = strings.Split(env_var, "=")[1]
			case strings.HasPrefix(env_var, "GREEN"): green = strings.Split(env_var, "=")[1]
			case strings.HasPrefix(env_var, "BLUE"): blue = strings.Split(env_var, "=")[1]
			case strings.HasPrefix(env_var, "NC"): noColor = strings.Split(env_var, "=")[1]
			case strings.HasPrefix(env_var, "VIRTUAL_ENV"): virtualEnv = strings.Split(env_var, "=")[1]	
		}
	}

	printShortenedPath(pwd, home, green, noColor, inContainer)

	if len(virtualEnv) > 0 {
		fmt.Printf(" (")
		printShortenedPath(virtualEnv, home, blue, noColor)
		fmt.Printf(")")
	}

	fmt.Printf("\n$ ")

	updateOpenStackAndKubeConfigInfo(osCloud, kubeConfig)

	removeTrailingWhiteSpace(home, inContainer)
}
