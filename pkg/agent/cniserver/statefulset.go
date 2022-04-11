// Copyright 2021 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cniserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	cnipb "antrea.io/antrea/pkg/apis/cni/v1beta1"
)

const (
	// The statefulSetDir is used to save static ip info
	statefulSetDir = "/var/lib/cni/antrea/statefulset/"
	// The staticIp annotation metadata key
	staticIp = "tos.network.staticIP"
)

var statefulSetApiVersions = []string{"/apis/apps.transwarp.io/v1alpha1", "/apis/apps/v1"}

type StaticIpMapSet = map[string]string

type KubeConfig struct {
	KubeconfigPath string `json:"kubeconfig,omitempty"`
}

type KubeStatefulSetList struct {
	StatefulSets []struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Replicas int `json:"replicas"`
		} `json:"spec"`
	} `json:"items"`
}

func newStaticIpMapSet(podIp string) StaticIpMapSet {
	return StaticIpMapSet{
		"IP":       podIp,
		"CniType":  "antrea",
		"Override": "true",
	}
}

// saveStatefulSet func save static ip info to local dir
func saveStatefulSet(podName, podNamespace, ip string) error {
	path := filepath.Join(statefulSetDir, podNamespace, podName)
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	path = filepath.Join(path, ip)
	log.Printf("save statefulset: %s, %s", path, ip)
	return ioutil.WriteFile(path, []byte(ip), 0600)
}

// getStatefulSet func get static ip info form local dir
func getStatefulSet(podName string, podNamespace string) (string, error) {
	path := filepath.Join(statefulSetDir, podNamespace, podName)
	dir, err := ioutil.ReadDir(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	for _, fi := range dir {
		ip := net.ParseIP(fi.Name())
		if ip != nil {
			return fi.Name(), nil
		}
	}
	return "", nil
}

// delStatefulSet func remove static ip info form local dir
func delStatefulSet(podName string, podNamespace string) error {
	path := filepath.Join(statefulSetDir, podNamespace, podName)
	//for pod not in statefulset, path not exist. removeall will error for it.
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}
	//if pod namespace is empty, we will clean it.
	//remove will remove dir if it is empty.
	//we don't use removeall, because other process may create file, during
	//we remove it. remove will leave it to os to handle it.
	path = filepath.Join(statefulSetDir, podNamespace)
	err = os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	strErr := err.Error()
	if strings.Contains(strErr, "directory not empty") {
		return nil
	}
	return err
}

// checkStatefulSet func check if the statefulset pod
func checkStatefulSet(client clientset.Interface, apiVersion, podName, podNamespace string) (bool, error) {
	url := fmt.Sprintf("%s/namespaces/%s/statefulsets", apiVersion, podNamespace)
	body, err := client.Discovery().RESTClient().Get().RequestURI(url).Do(context.TODO()).Raw()
	if err != nil {
		klog.Errorf("Failed to get statefulset info for podName:%s podNamespace:%s, %v", podName, podNamespace, err)
		return false, err
	}

	data := new(KubeStatefulSetList)
	err = json.Unmarshal(body, data)
	if err != nil {
		klog.Errorf("Failed to format statefulset info: %v", err)
		return false, err
	}
	for _, statefulset := range data.StatefulSets {
		name := statefulset.Metadata.Name
		namespace := statefulset.Metadata.Namespace
		replicas := statefulset.Spec.Replicas

		name = name + "-"
		if strings.HasPrefix(podName, name) {
			namelen := len(name)
			nameend := podName[namelen:]
			id, err := strconv.Atoi(nameend)
			if err != nil {
				continue
			}
			if id >= 0 && id < replicas && namespace == podNamespace {
				return true, nil
			}
		}
		if namespace != podNamespace {
			return false, fmt.Errorf("namespace not the same with we got. may not happen")
		}
	}
	return false, nil
}

// checkStaticIp func check the configured isStaticIP metadata for pod
func checkStaticIp(client clientset.Interface, podName, podNamespace string) (bool, error) {
	pod, err := client.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get pod info, podName:%s podNamespace:%s, %v", podName, podNamespace, err)
		return false, err
	}
	isStaticIP, ok := pod.ObjectMeta.Annotations[staticIp]
	if !ok || strings.ToLower(isStaticIP) != "true" {
		return false, nil
	}
	return true, nil
}

// getConfiguredStaticIp func check isStaticIP metadata and isStatefulSet, then get static ip from local dir
func getConfiguredStaticIp(client clientset.Interface, podName, podNamespace string, checkCache bool) (bool, bool, string, error) {
	var configuredPodIp = ""
	isStaticIP, err := checkStaticIp(client, podName, podNamespace)
	if err != nil {
		return false, false, configuredPodIp, err
	}
	if !isStaticIP {
		return isStaticIP, false, configuredPodIp, nil
	}

	isStatefulSet := false
	for _, apiVersion := range statefulSetApiVersions {
		isStatefulSet, err = checkStatefulSet(client, apiVersion, podName, podNamespace)
		if err != nil {
			klog.Warningf("Failed to check StatefulSet resource, %v", err)
		}
		if isStatefulSet {
			break
		}
	}
	if !checkCache && !isStatefulSet {
		return isStaticIP, isStatefulSet, configuredPodIp, err
	}
	configuredPodIp, err = getStatefulSet(podName, podNamespace)
	if err != nil {
		return isStaticIP, isStatefulSet, configuredPodIp, err
	}
	return isStaticIP, isStatefulSet, configuredPodIp, nil
}

func prepareCheckStaticIpForDel(client clientset.Interface, podName, podNamespace string) (bool, bool) {

	isStaticIP := false
	isStatefulSet := false

	configuredPodIp, _ := getStatefulSet(podName, podNamespace)
	if configuredPodIp != "" {
		isStaticIP = true
	}

	var err error
	for _, apiVersion := range statefulSetApiVersions {
		isStatefulSet, err = checkStatefulSet(client, apiVersion, podName, podNamespace)
		if err != nil {
			klog.Warningf("Failed to check StatefulSet resource, %v", err)
		}
		if isStatefulSet {
			break
		}
	}

	return isStaticIP, isStatefulSet
}

// configIPAMForStaticIp func config static ip to ipam args
func configIPAMForStaticIp(staticIpSet StaticIpMapSet, ipamArgs *cnipb.CniCmdArgs) {
	for key, value := range staticIpSet {
		configCNIArgs(key, value, ipamArgs)
	}
}

// configKubeConfig func config kubeconfig file path for ipam args
func configKubeConfig(netConf []byte, ipamArgs *cnipb.CniCmdArgs) {
	kubeConfig := &KubeConfig{}
	if err := json.Unmarshal(netConf, kubeConfig); err != nil {
		klog.Errorf("Failed to parse kubeconfig from network configuration %v", err)
	}
	if kubeConfig.KubeconfigPath != "" {
		configCNIArgs("Kubeconfig", kubeConfig.KubeconfigPath, ipamArgs)
	}
}

func configCNIArgs(key, value string, ipamArgs *cnipb.CniCmdArgs) {
	if len(ipamArgs.Args) > 0 && !strings.HasSuffix(ipamArgs.Args, ";") {
		ipamArgs.Args += ";"
	}
	ipamArgs.Args += fmt.Sprintf("%s=%s", key, value)
}

func removeStaticPodStaleFlows(podConfigurator *podConfigurator, podName, podNamespace string) error {
	ifaces := podConfigurator.ifaceStore.GetInterfacesByEntity(podName, podNamespace)
	for _, iface := range ifaces {
		if err := podConfigurator.ofClient.UninstallPodFlows(iface.InterfaceName); err != nil {
			return err
		}
		klog.Infof("remove stale flows for %s/%s", podName, podNamespace)
	}
	return nil
}
