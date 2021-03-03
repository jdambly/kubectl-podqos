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

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ResourceData containts CPU/Memory quantity
type ResourceData struct {
	CPU *resource.Quantity
	Memory *resource.Quantity
}
// ContainerData holds container information
type ContainerData struct {
	Name string
	Limits ResourceData
	Requests ResourceData
}
// PodData holds pod infromation, and list of containers in pod
type PodData struct {
	PodName string
	NameSpace string
	Containers []ContainerData
}

// PodQosPolicy describes the QosClass for each container
// see: https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/
// for more inforamtion
type PodQosPolicy  string

const (
	// BestEffort class when no resource requests or limits are specified.
	BestEffort PodQosPolicy = "BestEffort"

	// Burstable class when requests are less then limits
	Burstable PodQosPolicy = "Burstable"

	// Guaranteed class when requests are equal to limits
	Guaranteed PodQosPolicy = "Guaranteed"
)

// check if the kubeconfig is set, if not use default
func getKubeConfig() *string {
	env, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		temp := "~/.kube/config"
		return &temp
	}
	return &env
}

func (c *ContainerData) getQosClass() PodQosPolicy {

	if c.Limits.CPU.MilliValue() == 0 && c.Requests.CPU.MilliValue() == 0 {
		return BestEffort
	}
	if c.Limits.CPU.MilliValue() == c.Requests.CPU.MilliValue() {
		return Guaranteed
	}
	if c.Requests.CPU.MilliValue() < c.Limits.CPU.MilliValue(){
		return Burstable
	}
	return BestEffort
}

func main() {
	var namespaceFlag = flag.String("namespace", "", "sets the namespace for the api request")
	flag.StringVar(namespaceFlag, "n", "", "sets the namespace for the api request")
	allNameSpaces := flag.Bool("A", false, "Query all namespaces")
	flag.Parse()
	clientCfg, _ := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	namespace := clientCfg.Contexts[clientCfg.CurrentContext].Namespace

	// if the flag is set return the vaule given as a flag
	if *namespaceFlag != "" {
		namespace = *namespaceFlag
	}

	if namespace == "" {
		namespace = "default"
	}

	// if the -A flag is set use empty string
	if *allNameSpaces {
		namespace = ""
	}

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
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	var podData []PodData
	// loop through the pods, and for each pod get the resources
	for _, pod := range pods.Items { // don't forget _ is there to ignore the index of the list
		var containers []ContainerData
		for _, container := range pod.Spec.Containers {
			cpuLimit := container.Resources.Limits.Cpu()
			cpuRequest := container.Resources.Requests.Cpu()

			containers = append(containers, ContainerData{
				Name: container.Name,
				Limits: ResourceData{
					CPU: cpuLimit,
					Memory: container.Resources.Limits.Memory(),
				},
				Requests: ResourceData{
					CPU: cpuRequest,
					Memory: container.Resources.Requests.Memory(),
				},

			})
		}
		podData = append(podData, PodData{
			PodName: pod.Name,
			NameSpace: pod.Namespace,
			Containers: containers,
		})
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAMESPACE\tPOD NAME\tCONTAINER\tCPUl\tCPUr\tCLASS")
	for _, v := range podData {
		for _, c := range v.Containers {
			fmt.Fprintln(tw, strings.Join([]string{v.NameSpace, v.PodName, c.Name, c.Limits.CPU.String(), c.Requests.CPU.String(), string(c.getQosClass())}, "\t"))

		}
	}
	tw.Flush()
}