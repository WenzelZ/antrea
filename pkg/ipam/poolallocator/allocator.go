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

package poolallocator

import (
	"context"
	"fmt"
	"net"

	"antrea.io/antrea/pkg/apis/crd/v1alpha2"
	crdclientset "antrea.io/antrea/pkg/client/clientset/versioned"
	"antrea.io/antrea/pkg/ipam/ipallocator"
	iputil "antrea.io/antrea/pkg/util/ip"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// IPPoolAllocator is responsible for allocating IPs from IP set defined in IPPool CRD.
// The will update CRD usage accordingly.
// Pool Allocator assumes that pool with allocated IPs can not be deleted. Pool ranges can
// only be extended.
type IPPoolAllocator struct {
	// Name of IP Pool custom resource
	ipPoolName string

	// crd client to access the pool
	crdClient crdclientset.Interface
}

// NewIPPoolAllocator creates an IPPoolAllocator based on the provided IP pool.
func NewIPPoolAllocator(poolName string, client crdclientset.Interface) (*IPPoolAllocator, error) {

	// Validate the pool exists
	// This has an extra roundtrip cost, however this would allow fallback to
	// default IPAM driver if needed
	_, err := client.CrdV1alpha2().IPPools().Get(context.TODO(), poolName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	allocator := &IPPoolAllocator{
		ipPoolName: poolName,
		crdClient:  client,
	}

	return allocator, nil
}

// initAllocatorList reads IP Pool status and initializes a list of allocators based on
// IP Pool spec and state of allocation recorded in the status
func (a *IPPoolAllocator) initIPAllocators(ipPool *v1alpha2.IPPool) (ipallocator.MultiIPAllocator, error) {

	var allocators ipallocator.MultiIPAllocator

	// Initialize a list of IP allocators based on pool spec
	for _, ipRange := range ipPool.Spec.IPRanges {
		if len(ipRange.CIDR) > 0 {
			// Reserve gateway address and broadcast address
			reservedIPs := []net.IP{net.ParseIP(ipRange.SubnetInfo.Gateway)}
			_, ipNet, err := net.ParseCIDR(ipRange.CIDR)
			if err != nil {
				return nil, err
			}

			size, bits := ipNet.Mask.Size()
			if int32(size) == ipRange.SubnetInfo.PrefixLength && bits == 32 {
				// Allocation CIDR covers entire subnet, thus we need
				// to reserve broadcast IP as well for IPv4
				reservedIPs = append(reservedIPs, iputil.GetLocalBroadcastIP(ipNet))
			}

			allocator, err := ipallocator.NewCIDRAllocator(ipNet, reservedIPs)
			if err != nil {
				return nil, err
			}
			allocators = append(allocators, allocator)
		} else {
			allocator, err := ipallocator.NewIPRangeAllocator(net.ParseIP(ipRange.Start), net.ParseIP(ipRange.End))
			if err != nil {
				return allocators, err
			}
			allocators = append(allocators, allocator)
		}
	}

	// Mark allocated IPs from pool status as unavailable
	for _, ip := range ipPool.Status.IPAddresses {
		err := allocators.AllocateIP(net.ParseIP(ip.IPAddress))
		if err != nil {
			// TODO - fix state if possible
			return allocators, fmt.Errorf("inconsistent state for IP Pool %s with IP %s", ipPool.Name, ip.IPAddress)
		}
	}

	return allocators, nil
}

func (a *IPPoolAllocator) readPoolAndInitIPAllocators() (*v1alpha2.IPPool, ipallocator.MultiIPAllocator, error) {
	ipPool, err := a.crdClient.CrdV1alpha2().IPPools().Get(context.TODO(), a.ipPoolName, metav1.GetOptions{})

	if err != nil {
		return nil, ipallocator.MultiIPAllocator{}, err
	}

	allocators, err := a.initIPAllocators(ipPool)
	if err != nil {
		return nil, ipallocator.MultiIPAllocator{}, err
	}
	return ipPool, allocators, nil
}

func (a *IPPoolAllocator) appendPoolUsage(ipPool *v1alpha2.IPPool, ip net.IP, state v1alpha2.IPAddressPhase, owner v1alpha2.IPAddressOwner) error {
	newPool := ipPool.DeepCopy()
	usageEntry := v1alpha2.IPAddressState{
		IPAddress: ip.String(),
		Phase:     state,
		Owner:     owner,
	}

	newPool.Status.IPAddresses = append(newPool.Status.IPAddresses, usageEntry)
	_, err := a.crdClient.CrdV1alpha2().IPPools().UpdateStatus(context.TODO(), newPool, metav1.UpdateOptions{})
	if err != nil {
		klog.Warningf("IP Pool %s update with status %+v failed: %+v", newPool.Name, newPool.Status, err)
		return err
	}
	klog.InfoS("IP Pool update successful", "pool", newPool.Name, "allocation", newPool.Status)
	return nil

}

// Update pool status to delete released IP
func (a *IPPoolAllocator) removePoolUsage(ipPool *v1alpha2.IPPool, ip net.IP) error {

	ipString := ip.String()
	newPool := ipPool.DeepCopy()
	var newList []v1alpha2.IPAddressState
	for _, entry := range ipPool.Status.IPAddresses {
		if entry.IPAddress != ipString {
			newList = append(newList, entry)
		}
	}

	if len(newList) == len(ipPool.Status.IPAddresses) {
		return fmt.Errorf("IP address %s was not allocated from IP pool %s", ip, ipPool.Name)
	}

	newPool.Status.IPAddresses = newList

	_, err := a.crdClient.CrdV1alpha2().IPPools().UpdateStatus(context.TODO(), newPool, metav1.UpdateOptions{})
	if err != nil {
		klog.Warningf("IP Pool %s update failed: %+v", newPool.Name, err)
		return err
	}
	klog.InfoS("IP Pool update successful", "pool", newPool.Name, "allocation", newPool.Status)
	return nil

}

// AllocateIP allocates the specified IP. It returns error if the IP is not in the range or already
// allocated, or in case CRD failed to update its state.
// In case of success, IP pool CRD status is updated with allocated IP/state/resource/container.
// AllocateIP returns subnet details for the requested IP, as defined in IP pool spec.
func (a *IPPoolAllocator) AllocateIP(ip net.IP, state v1alpha2.IPAddressPhase, owner v1alpha2.IPAddressOwner) (v1alpha2.SubnetInfo, error) {
	var subnetSpec v1alpha2.SubnetInfo
	// Retry on CRD update conflict which is caused by multiple agents updating a pool at same time.
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ipPool, allocators, err := a.readPoolAndInitIPAllocators()
		if err != nil {
			return err
		}

		index := len(allocators)
		for i, allocator := range allocators {
			if allocator.Has(ip) {
				err := allocator.AllocateIP(ip)
				if err != nil {
					return err
				}
				index = i
				break
			}
		}

		if index == len(allocators) {
			// Failed to find matching range
			return fmt.Errorf("IP %v does not belong to IP pool %s", ip, a.ipPoolName)
		}

		subnetSpec = ipPool.Spec.IPRanges[index].SubnetInfo
		err = a.appendPoolUsage(ipPool, ip, state, owner)

		return err
	})

	if err != nil {
		klog.Errorf("Failed to allocate IP address %s from pool %s: %+v", ip, a.ipPoolName, err)
	}
	return subnetSpec, err
}

// AllocateNext allocates the next available IP. It returns error if pool is exausted,
// or in case CRD failed to update its state.
// In case of success, IP pool CRD status is updated with allocated IP/state/resource/container.
// AllocateIP returns subnet details for the requested IP, as defined in IP pool spec.
func (a *IPPoolAllocator) AllocateNext(state v1alpha2.IPAddressPhase, owner v1alpha2.IPAddressOwner) (net.IP, v1alpha2.SubnetInfo, error) {
	var subnetSpec v1alpha2.SubnetInfo
	var ip net.IP
	// Same resource can not ask for allocation twice without release
	// This needs to be verified even at the expence of another API call
	exists, err := a.HasContainer(owner.Pod.ContainerID)
	if err != nil {
		return ip, subnetSpec, err
	}

	if exists {
		return ip, subnetSpec, fmt.Errorf("container %s was already allocated an address from IP Pool %s", owner.Pod.ContainerID, a.ipPoolName)
	}

	// Retry on CRD update conflict which is caused by multiple agents updating a pool at same time.
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ipPool, allocators, err := a.readPoolAndInitIPAllocators()
		if err != nil {
			return err
		}

		index := len(allocators)
		for i, allocator := range allocators {
			ip, err = allocator.AllocateNext()
			if err == nil {
				// successful allocation
				index = i
				break
			}
		}

		if index == len(allocators) {
			// Failed to find matching range
			return fmt.Errorf("failed to allocate IP: Pool %s is exausted", a.ipPoolName)
		}

		subnetSpec = ipPool.Spec.IPRanges[index].SubnetInfo
		return a.appendPoolUsage(ipPool, ip, state, owner)
	})

	if err != nil {
		klog.Errorf("Failed to allocate from pool %s: %+v", a.ipPoolName, err)
	}
	return ip, subnetSpec, err
}

// Release releases the provided IP. It returns error if the IP is not in the range or not allocated,
// or in case CRD failed to update its state.
// In case of success, IP pool CRD status is updated with released IP/state/resource.
func (a *IPPoolAllocator) Release(ip net.IP) error {

	// Retry on CRD update conflict which is caused by multiple agents updating a pool at same time.
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ipPool, allocators, err := a.readPoolAndInitIPAllocators()
		if err != nil {
			return err
		}

		err = allocators.Release(ip)

		if err != nil {
			// Failed to find matching range
			return fmt.Errorf("IP %v does not belong to IP pool %s", ip, a.ipPoolName)
		}

		return a.removePoolUsage(ipPool, ip)
	})

	if err != nil {
		klog.Errorf("Failed to release IP address %s from pool %s: %+v", ip, a.ipPoolName, err)
	}
	return err
}

// ReleaseResource releases the IP associated with specified Pod. It returns error if the resource is not present in state or in case CRD failed to update its state.
// In case of success, IP pool CRD status is updated with released entry.
func (a *IPPoolAllocator) ReleasePod(namespace, podName string) error {

	// Retry on CRD update conflict which is caused by multiple agents updating a pool at same time.
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ipPool, err := a.crdClient.CrdV1alpha2().IPPools().Get(context.TODO(), a.ipPoolName, metav1.GetOptions{})

		if err != nil {
			return err
		}

		// Mark allocated IPs from pool status as unavailable
		for _, ip := range ipPool.Status.IPAddresses {
			if ip.Owner.Pod != nil && ip.Owner.Pod.Namespace == namespace && ip.Owner.Pod.Name == podName {
				return a.removePoolUsage(ipPool, net.ParseIP(ip.IPAddress))

			}
		}

		klog.V(4).InfoS("IP Pool state:", "name", a.ipPoolName, "allocation", ipPool.Status.IPAddresses)
		return fmt.Errorf("failed to find record of IP allocated to Pod:%s/%s in pool %s", namespace, podName, a.ipPoolName)
	})

	if err != nil {
		klog.Errorf("Failed to release IP address for Pod:%s/%s from pool %s: %+v", namespace, podName, a.ipPoolName, err)
	}
	return err
}

// ReleaseContainerIfPresent releases the IP associated with specified container ID if present in state.
// It returns error in case CRD failed to update its state, or if pool does not exist.
// In case of success, IP pool CRD status is updated with released entry.
func (a *IPPoolAllocator) ReleaseContainerIfPresent(containerID string) error {

	// Retry on CRD update conflict which is caused by multiple agents updating a pool at same time.
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ipPool, err := a.crdClient.CrdV1alpha2().IPPools().Get(context.TODO(), a.ipPoolName, metav1.GetOptions{})

		if err != nil {
			return err
		}

		// Mark allocated IPs from pool status as unavailable
		for _, ip := range ipPool.Status.IPAddresses {
			if ip.Owner.Pod != nil && ip.Owner.Pod.ContainerID == containerID {
				return a.removePoolUsage(ipPool, net.ParseIP(ip.IPAddress))

			}
		}

		klog.V(4).InfoS("Failed to find allocation record in pool", "container", containerID, "pool", a.ipPoolName, "allocation", ipPool.Status.IPAddresses)
		return nil
	})

	if err != nil {
		klog.Errorf("Failed to release IP address for container %s from pool %s: %+v", containerID, a.ipPoolName, err)
	}
	return err
}

// HasResource checks whether an IP was associated with specified pod. It returns error if the resource is crd fails to be retrieved.
func (a *IPPoolAllocator) HasPod(namespace, podName string) (bool, error) {

	ipPool, err := a.crdClient.CrdV1alpha2().IPPools().Get(context.TODO(), a.ipPoolName, metav1.GetOptions{})

	if err != nil {
		return false, err
	}

	for _, ip := range ipPool.Status.IPAddresses {
		if ip.Owner.Pod != nil && ip.Owner.Pod.Namespace == namespace && ip.Owner.Pod.Name == podName {
			return true, nil
		}
	}
	return false, nil
}

// HasResource checks whether an IP was associated with specified container. It returns error if the resource is crd fails to be retrieved.
func (a *IPPoolAllocator) HasContainer(containerID string) (bool, error) {

	ipPool, err := a.crdClient.CrdV1alpha2().IPPools().Get(context.TODO(), a.ipPoolName, metav1.GetOptions{})

	if err != nil {
		return false, err
	}

	for _, ip := range ipPool.Status.IPAddresses {
		if ip.Owner.Pod != nil && ip.Owner.Pod.ContainerID == containerID {
			return true, nil
		}
	}
	return false, nil
}
