/*
Copyright 2021 Jeff d'Ambly

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)
type ResourceData struct {
	CPU string
	Memory string
}

type ContainerData struct {
	Name string
	Limits ResourceData
	Requests ResourceData
}

type PodData struct {
	PodName string
	NameSpace string
	Containers []ContainerData
}

// check if the kubeconfig is set, if not use default
func getKubeConfig() *string {
	env, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		temp := "~/.kube/config"
		return &temp
	}
	return &env
}

// get the namespace for the api call request
func getNamespace(nameSpace string) string {
	// if the flag is set return the vaule given as a flag
	if *namespaceFlag != "" {
		return string(*namespaceFlag)
	}

	if nameSpace == "" {
		return "default"
	}
	return nameSpace
}


var namespaceFlag = flag.String("namespace", "", "sets the namespace for the api request")

func init(){
	flag.StringVar(namespaceFlag, "n", "", "sets the namespace for the api request")

}

func main() {
	flag.Parse()

	clientCfg, _ := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	nameSpace := getNamespace(clientCfg.Contexts[clientCfg.CurrentContext].Namespace)

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *getKubeConfig())
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	pods, err := clientset.CoreV1().Pods(nameSpace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	var podData []PodData
	// loop through the pods, and for each pod get the resources
	for _, pod := range pods.Items { // don't forget _ is there to ignore the index of the list
		var containers []ContainerData
		for _, container := range pod.Spec.InitContainers {
			containers = append(containers, ContainerData{
				Name: container.Name,
				Limits: ResourceData{
					CPU: container.Limits.Cpu(),
					Memory: container.Limits.Memory(),
				},
				Requests: ResourceData{
					CPU: container.Requests.Cpu(),
					Memory: container.Requests.Memory(),
				},

			})
		}
		podData = append(podData, PodData{
			PodName: pod.Name,
			NameSpace: nameSpace,
			Containers: containers,
		})
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAMESPACE\tPOD NAME\tCPUL\tCPUR")
	for _, v := range podData {
		for _, c := range v.Containers {
			fmt.Fprintln(tw, strings.Join([]string{v.NameSpace, v.PodName, c.Name}, "\t"))

		}
	}
	tw.Flush()
}