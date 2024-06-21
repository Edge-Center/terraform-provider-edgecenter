//go:build cloud_data_source || cloud_resource

package edgecenter_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/clusters"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/loadbalancers"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/availablenetworks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/router/v1/routers"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/volume/v1/volumes"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func createTestNetwork(client *edgecloud.ServiceClient, opts networks.CreateOpts) (string, error) {
	result, err := networks.Create(client, opts).Extract()
	if err != nil {
		return "", err
	}

	taskID := result.Tasks[0]
	networkID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, int(edgecenter.NetworkCreatingTimeout.Seconds()), func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		networkID, err := networks.ExtractNetworkIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve network ID from task info: %w", err)
		}
		return networkID, nil
	})
	if err != nil {
		return "", err
	}

	return networkID.(string), nil
}

func deleteTestNetwork(client *edgecloud.ServiceClient, networkID string) error {
	result, err := networks.Delete(client, networkID).Extract()
	if err != nil {
		return err
	}

	taskID := result.Tasks[0]
	err = tasks.WaitTaskAndProcessResult(client, taskID, true, int(edgecenter.NetworkDeletingTimeout.Seconds()), func(task tasks.TaskID) error {
		_, err := networks.Get(client, networkID).Extract()
		if err == nil {
			return fmt.Errorf("cannot delete network with ID: %s", networkID)
		}

		var errDefault404 edgecloud.Default404Error
		if errors.As(err, &errDefault404) {
			return nil
		}
		return fmt.Errorf("extracting Network resource error: %w", err)
	})

	return err
}

func createTestSubnet(client *edgecloud.ServiceClient, opts subnets.CreateOpts, extra ...string) (string, error) {
	subCidr := cidrTest
	if extra != nil {
		subCidr = extra[0]
	}

	var eccidr edgecloud.CIDR
	_, netIPNet, err := net.ParseCIDR(subCidr)
	if err != nil {
		return "", err
	}
	eccidr.IP = netIPNet.IP
	eccidr.Mask = netIPNet.Mask
	opts.CIDR = eccidr

	result, err := subnets.Create(client, opts).Extract()
	if err != nil {
		return "", err
	}

	taskID := result.Tasks[0]
	subnetID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, int(edgecenter.SubnetCreatingTimeout.Seconds()), func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		subnet, err := subnets.ExtractSubnetIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve Subnet ID from task info: %w", err)
		}
		return subnet, nil
	})

	return subnetID.(string), err
}

func patchRouterForK8S(provider *edgecloud.ProviderClient, networkID string) error {
	routersClient, err := createTestClient(provider, edgecenter.RouterPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}

	aNetClient, err := createTestClient(provider, edgecenter.SharedNetworksPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}

	availableNetworks, err := availablenetworks.ListAll(aNetClient, nil)
	if err != nil {
		return err
	}
	var extNet availablenetworks.Network
	for _, an := range availableNetworks {
		if an.External {
			extNet = an
			break
		}
	}

	rs, err := routers.ListAll(routersClient, nil)
	if err != nil {
		return err
	}

	var router routers.Router
	for _, r := range rs {
		if strings.Contains(r.Name, networkID) {
			router = r
			break
		}
	}

	extSubnet := extNet.Subnets[0]
	routerOpts := routers.UpdateOpts{Routes: extSubnet.HostRoutes}
	if _, err = routers.Update(routersClient, router.ID, routerOpts).Extract(); err != nil {
		return err
	}

	return nil
}

func createTestCluster(client *edgecloud.ServiceClient, opts clusters.CreateOpts) (string, error) {
	result, err := clusters.Create(client, opts).Extract()
	if err != nil {
		return "", err
	}

	taskID := result.Tasks[0]
	clusterID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, edgecenter.K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		clusterID, err := clusters.ExtractClusterIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve cluster ID from task info: %w", err)
		}
		return clusterID, nil
	})
	if err != nil {
		return "", err
	}

	return clusterID.(string), nil
}

func deleteTestCluster(client *edgecloud.ServiceClient, clusterID string) error {
	result, err := clusters.Delete(client, clusterID).Extract()
	if err != nil {
		return err
	}

	taskID := result.Tasks[0]
	err = tasks.WaitTaskAndProcessResult(client, taskID, true, edgecenter.K8sCreateTimeout, func(task tasks.TaskID) error {
		_, err := clusters.Get(client, clusterID).Extract()
		if err == nil {
			return fmt.Errorf("cannot delete k8s cluster with ID: %s", clusterID)
		}

		var errDefault404 edgecloud.Default404Error
		if errors.As(err, &errDefault404) {
			return nil
		}
		return fmt.Errorf("extracting k8s cluster resource error: %w", err)
	})

	return err
}

func createTestLoadBalancerWithListener(client *edgecloud.ServiceClient, opts loadbalancers.CreateOpts) (string, error) {
	result, err := loadbalancers.Create(client, opts).Extract()
	if err != nil {
		return "", err
	}

	taskID := result.Tasks[0]
	lbID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, int(edgecenter.LoadBalancerCreateTimeout.Seconds()), func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		lbID, err := loadbalancers.ExtractLoadBalancerIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve LoadBalancer ID from task info: %w", err)
		}
		return lbID, nil
	})
	if err != nil {
		return "", err
	}

	return lbID.(string), nil
}

func createTestLoadBalancerWithListenerV2(ctx context.Context, client edgecloudV2.Client, opts edgecloudV2.LoadbalancerCreateRequest) (string, error) {
	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.Loadbalancers.Create, &opts, &client, edgecenter.LoadBalancerCreateTimeout)
	if err != nil {
		return "", err
	}
	lbID := taskResult.Loadbalancers[0]
	return lbID, nil
}

func createTestVolume(client *edgecloud.ServiceClient, opts volumes.CreateOpts) (string, error) {
	result, err := volumes.Create(client, opts).Extract()
	if err != nil {
		return "", err
	}

	taskID := result.Tasks[0]
	volumeID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, int(edgecenter.VolumeCreatingTimeout.Seconds()), func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		volumeID, err := volumes.ExtractVolumeIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve volume ID from task info: %w", err)
		}
		return volumeID, nil
	})
	if err != nil {
		return "", err
	}

	return volumeID.(string), nil
}

func createTestVolumeV2(ctx context.Context, client edgecloudV2.Client, opts *edgecloudV2.VolumeCreateRequest) (string, error) {
	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.Volumes.Create, opts, &client)
	if err != nil {
		return "", err
	}

	return taskResult.Volumes[0], nil
}
