// Copyright 2021 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by client-gen. DO NOT EDIT.

package versioned

import (
	"fmt"

	clusterinformationv1beta1 "antrea.io/antrea/pkg/legacyclient/clientset/versioned/typed/clusterinformation/v1beta1"
	controlplanev1beta2 "antrea.io/antrea/pkg/legacyclient/clientset/versioned/typed/controlplane/v1beta2"
	corev1alpha2 "antrea.io/antrea/pkg/legacyclient/clientset/versioned/typed/core/v1alpha2"
	opsv1alpha1 "antrea.io/antrea/pkg/legacyclient/clientset/versioned/typed/ops/v1alpha1"
	securityv1alpha1 "antrea.io/antrea/pkg/legacyclient/clientset/versioned/typed/security/v1alpha1"
	statsv1alpha1 "antrea.io/antrea/pkg/legacyclient/clientset/versioned/typed/stats/v1alpha1"
	systemv1beta1 "antrea.io/antrea/pkg/legacyclient/clientset/versioned/typed/system/v1beta1"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	ClusterinformationV1beta1() clusterinformationv1beta1.ClusterinformationV1beta1Interface
	ControlplaneV1beta2() controlplanev1beta2.ControlplaneV1beta2Interface
	CoreV1alpha2() corev1alpha2.CoreV1alpha2Interface
	OpsV1alpha1() opsv1alpha1.OpsV1alpha1Interface
	SecurityV1alpha1() securityv1alpha1.SecurityV1alpha1Interface
	StatsV1alpha1() statsv1alpha1.StatsV1alpha1Interface
	SystemV1beta1() systemv1beta1.SystemV1beta1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	clusterinformationV1beta1 *clusterinformationv1beta1.ClusterinformationV1beta1Client
	controlplaneV1beta2       *controlplanev1beta2.ControlplaneV1beta2Client
	coreV1alpha2              *corev1alpha2.CoreV1alpha2Client
	opsV1alpha1               *opsv1alpha1.OpsV1alpha1Client
	securityV1alpha1          *securityv1alpha1.SecurityV1alpha1Client
	statsV1alpha1             *statsv1alpha1.StatsV1alpha1Client
	systemV1beta1             *systemv1beta1.SystemV1beta1Client
}

// ClusterinformationV1beta1 retrieves the ClusterinformationV1beta1Client
func (c *Clientset) ClusterinformationV1beta1() clusterinformationv1beta1.ClusterinformationV1beta1Interface {
	return c.clusterinformationV1beta1
}

// ControlplaneV1beta2 retrieves the ControlplaneV1beta2Client
func (c *Clientset) ControlplaneV1beta2() controlplanev1beta2.ControlplaneV1beta2Interface {
	return c.controlplaneV1beta2
}

// CoreV1alpha2 retrieves the CoreV1alpha2Client
func (c *Clientset) CoreV1alpha2() corev1alpha2.CoreV1alpha2Interface {
	return c.coreV1alpha2
}

// OpsV1alpha1 retrieves the OpsV1alpha1Client
func (c *Clientset) OpsV1alpha1() opsv1alpha1.OpsV1alpha1Interface {
	return c.opsV1alpha1
}

// SecurityV1alpha1 retrieves the SecurityV1alpha1Client
func (c *Clientset) SecurityV1alpha1() securityv1alpha1.SecurityV1alpha1Interface {
	return c.securityV1alpha1
}

// StatsV1alpha1 retrieves the StatsV1alpha1Client
func (c *Clientset) StatsV1alpha1() statsv1alpha1.StatsV1alpha1Interface {
	return c.statsV1alpha1
}

// SystemV1beta1 retrieves the SystemV1beta1Client
func (c *Clientset) SystemV1beta1() systemv1beta1.SystemV1beta1Interface {
	return c.systemV1beta1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfig will generate a rate-limiter in configShallowCopy.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		if configShallowCopy.Burst <= 0 {
			return nil, fmt.Errorf("burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0")
		}
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.clusterinformationV1beta1, err = clusterinformationv1beta1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.controlplaneV1beta2, err = controlplanev1beta2.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.coreV1alpha2, err = corev1alpha2.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.opsV1alpha1, err = opsv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.securityV1alpha1, err = securityv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.statsV1alpha1, err = statsv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.systemV1beta1, err = systemv1beta1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.clusterinformationV1beta1 = clusterinformationv1beta1.NewForConfigOrDie(c)
	cs.controlplaneV1beta2 = controlplanev1beta2.NewForConfigOrDie(c)
	cs.coreV1alpha2 = corev1alpha2.NewForConfigOrDie(c)
	cs.opsV1alpha1 = opsv1alpha1.NewForConfigOrDie(c)
	cs.securityV1alpha1 = securityv1alpha1.NewForConfigOrDie(c)
	cs.statsV1alpha1 = statsv1alpha1.NewForConfigOrDie(c)
	cs.systemV1beta1 = systemv1beta1.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.clusterinformationV1beta1 = clusterinformationv1beta1.New(c)
	cs.controlplaneV1beta2 = controlplanev1beta2.New(c)
	cs.coreV1alpha2 = corev1alpha2.New(c)
	cs.opsV1alpha1 = opsv1alpha1.New(c)
	cs.securityV1alpha1 = securityv1alpha1.New(c)
	cs.statsV1alpha1 = statsv1alpha1.New(c)
	cs.systemV1beta1 = systemv1beta1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
