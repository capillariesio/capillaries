package deploy

import (
	"fmt"
	"strings"
	"time"
)

func (*AwsDeployProvider) CreateFloatingIp(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.CreateFloatingIp", isVerbose)

	// {
	// 	"PublicIp": "35.172.255.76",
	// 	"AllocationId": "eipalloc-0e4b1b8ec4eb983d4",
	// }

	newFloatingIp, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "allocate-address",
		"--tag-specification", "ResourceType=elastic-ip,Tags=[{Key=Name,Value=bastion_ip_address}]"},
		".PublicIp", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	prjPair.SetSshExternalIp(newFloatingIp)

	reportPublicIp(&prjPair.Live)

	newNatGatewayIp, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "allocate-address",
		"--tag-specification", "ResourceType=elastic-ip,Tags=[{Key=Name,Value=natgw_ip_address}]"},
		".PublicIp", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	prjPair.SetNatGatewayExternalIp(newNatGatewayIp)

	return lb.Complete(nil)
}

func (*AwsDeployProvider) DeleteFloatingIp(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.DeleteFloatingIp", isVerbose)
	if prjPair.Live.SshConfig.ExternalIpAddress == "" {
		lb.Add("ssh_config external_ip_address is already empty, nothing to delete")
	} else {
		// {
		// 	"Addresses": [
		// 		{
		// 			"PublicIp": "35.172.255.76",
		// 			"AllocationId": "eipalloc-0e4b1b8ec4eb983d4",
		// 		}
		// 	]
		// }

		allocationId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-addresses"}, fmt.Sprintf(`.Addresses[] | select(.PublicIp == "%s") | .AllocationId`, prjPair.Live.SshConfig.ExternalIpAddress), false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "release-address", "--allocation-id", allocationId}, prjPair.Live.CliEnvVars, "")
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		prjPair.SetSshExternalIp("")
	}

	if prjPair.Live.Network.PublicSubnet.NatGatewayPublicIp == "" {
		lb.Add("nat gateway ip is already empty, nothing to delete")
	} else {
		natGatewayAllocationId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-addresses"}, fmt.Sprintf(`.Addresses[] | select(.PublicIp == "%s") | .AllocationId`, prjPair.Live.Network.PublicSubnet.NatGatewayPublicIp), false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "release-address", "--allocation-id", natGatewayAllocationId}, prjPair.Live.CliEnvVars, "")
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		prjPair.SetNatGatewayExternalIp("")
	}

	return lb.Complete(nil)
}

func createAwsVpc(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createAwsVpc", isVerbose)
	if prjPair.Live.Network.Name == "" {
		return lb.Complete(fmt.Errorf("network name cannot be empty"))
	}

	// Check if it's already there
	foundNetworkIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-vpcs",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.Name},
		".Vpcs[0].VpcId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if prjPair.Live.Network.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundNetworkIdByName != "" {
			lb.Add(fmt.Sprintf("network %s(%s) already there, updating project", prjPair.Live.Network.Name, foundNetworkIdByName))
			prjPair.SetNetworkId(foundNetworkIdByName)
		}
	} else {
		if foundNetworkIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested network id %s not present, consider removing this id from the project file", prjPair.Live.Network.Id))
		} else if prjPair.Live.Network.Id != foundNetworkIdByName {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested network id %s not matching existing network id %s", prjPair.Live.Network.Id, foundNetworkIdByName))
		}
	}

	if prjPair.Live.Network.Id != "" {
		lb.Add(fmt.Sprintf("network %s(%s) already there, no need to create", prjPair.Live.Network.Name, foundNetworkIdByName))
		return lb.Complete(nil)
	}

	// Create
	newId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-vpc",
		"--cidr-block", prjPair.Live.Network.Cidr,
		"--tag-specification", fmt.Sprintf("ResourceType=vpc,Tags=[{Key=Name,Value=%s}]", prjPair.Live.Network.Name)},
		".Vpc.VpcId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("creating network %s(%s)...\n", prjPair.Live.Network.Name, newId))
	prjPair.SetNetworkId(newId)

	return lb.Complete(nil)
}

func waitForAwsVpcToBeCreated(prj *Project, vpcId string, timeoutSeconds int, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("waitForAwsVpcToBeCreated:"+vpcId, isVerbose)
	startWaitTs := time.Now()
	for {
		status, er := ExecLocalAndGetJsonString(prj, "aws", []string{"ec2", "describe-vpcs",
			"--filter", "Name=vpc-id,Values=" + vpcId},
			".Vpcs[0].State", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
		if status == "" {
			return lb.Complete(fmt.Errorf("aws returned empty vpc status for %s", vpcId))
		}
		if status == "available" {
			break
		}
		if status != "pending" {
			return lb.Complete(fmt.Errorf("vpc %s was built, but the status is %s", vpcId, status))
		}
		if time.Since(startWaitTs).Seconds() > float64(prj.Timeouts.OpenstackInstanceCreation) {
			return lb.Complete(fmt.Errorf("giving up after waiting for vpc %s to be created", vpcId))
		}
		time.Sleep(10 * time.Second)
	}
	return lb.Complete(nil)
}

func createAwsPrivateSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createAwsPrivateSubnet", isVerbose)
	if prjPair.Live.Network.PrivateSubnet.Name == "" || prjPair.Live.Network.PrivateSubnet.Cidr == "" || prjPair.Live.Network.Id == "" || prjPair.Live.Network.PrivateSubnet.AvailabilityZone == "" {
		return lb.Complete(fmt.Errorf("subnet name(%s), cidr(%s), vpc id(%s), availability_zone(%s) cannot be empty", prjPair.Live.Network.PrivateSubnet.Name, prjPair.Live.Network.PrivateSubnet.Cidr, prjPair.Live.Network.Id, prjPair.Live.Network.PrivateSubnet.AvailabilityZone))
	}

	// Check if it's already there
	foundSubnetIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-subnets",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.PrivateSubnet.Name},
		".Subnets[0].SubnetId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if prjPair.Live.Network.PrivateSubnet.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundSubnetIdByName != "" {
			lb.Add(fmt.Sprintf("subnet %s(%s) already there, updating project", prjPair.Live.Network.PrivateSubnet.Name, foundSubnetIdByName))
			prjPair.SetPrivateSubnetId(foundSubnetIdByName)

		}
	} else {
		if foundSubnetIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested subnet id %s not present, consider removing this id from the project file", prjPair.Live.Network.PrivateSubnet.Id))
		} else if foundSubnetIdByName != prjPair.Live.Network.PrivateSubnet.Id {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested subnet id %s not matching existing id %s", prjPair.Live.Network.PrivateSubnet.Id, foundSubnetIdByName))
		}
	}

	if prjPair.Live.Network.PrivateSubnet.Id != "" {
		lb.Add(fmt.Sprintf("subnet %s(%s) already there, no need to create", prjPair.Live.Network.PrivateSubnet.Name, foundSubnetIdByName))
		return lb.Complete(nil)
	}

	// Create
	newId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-subnet",
		"--vpc-id", prjPair.Live.Network.Id,
		"--cidr-block", prjPair.Live.Network.PrivateSubnet.Cidr,
		"--availability-zone", prjPair.Live.Network.PrivateSubnet.AvailabilityZone,
		"--tag-specification", fmt.Sprintf("ResourceType=subnet,Tags=[{Key=Name,Value=%s}]", prjPair.Live.Network.PrivateSubnet.Name)},
		".Subnet.SubnetId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// TODO: dhcp options and allocation pools?

	lb.Add(fmt.Sprintf("created subnet %s(%s)", prjPair.Live.Network.PrivateSubnet.Name, newId))
	prjPair.SetPrivateSubnetId(newId)

	return lb.Complete(nil)
}

func createAwsPublicSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createAwsPublicSubnet", isVerbose)
	if prjPair.Live.Network.PublicSubnet.Name == "" || prjPair.Live.Network.PublicSubnet.Cidr == "" || prjPair.Live.Network.Id == "" || prjPair.Live.Network.PublicSubnet.AvailabilityZone == "" {
		return lb.Complete(fmt.Errorf("subnet name(%s), cidr(%s), vpc id(%s), availability_zone(%s) cannot be empty", prjPair.Live.Network.PublicSubnet.Name, prjPair.Live.Network.PublicSubnet.Cidr, prjPair.Live.Network.Id, prjPair.Live.Network.PublicSubnet.AvailabilityZone))
	}

	// Check if it's already there
	foundSubnetIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-subnets",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.PublicSubnet.Name},
		".Subnets[0].SubnetId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if prjPair.Live.Network.PublicSubnet.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundSubnetIdByName != "" {
			lb.Add(fmt.Sprintf("subnet %s(%s) already there, updating project", prjPair.Live.Network.PublicSubnet.Name, foundSubnetIdByName))
			prjPair.SetPublicSubnetId(foundSubnetIdByName)

		}
	} else {
		if foundSubnetIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested subnet id %s not present, consider removing this id from the project file", prjPair.Live.Network.PublicSubnet.Id))
		} else if foundSubnetIdByName != prjPair.Live.Network.PublicSubnet.Id {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested subnet id %s not matching existing id %s", prjPair.Live.Network.PublicSubnet.Id, foundSubnetIdByName))
		}
	}

	if prjPair.Live.Network.PublicSubnet.Id != "" {
		lb.Add(fmt.Sprintf("subnet %s(%s) already there, no need to create", prjPair.Live.Network.PublicSubnet.Name, foundSubnetIdByName))
		return lb.Complete(nil)
	}

	// Create
	newId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-subnet",
		"--vpc-id", prjPair.Live.Network.Id,
		"--cidr-block", prjPair.Live.Network.PublicSubnet.Cidr,
		"--availability-zone", prjPair.Live.Network.PublicSubnet.AvailabilityZone,
		"--tag-specification", fmt.Sprintf("ResourceType=subnet,Tags=[{Key=Name,Value=%s}]", prjPair.Live.Network.PublicSubnet.Name)},
		".Subnet.SubnetId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// TODO: dhcp options and allocation pools?

	lb.Add(fmt.Sprintf("created subnet %s(%s)", prjPair.Live.Network.PublicSubnet.Name, newId))
	prjPair.SetPublicSubnetId(newId)

	return lb.Complete(nil)
}

func createNatGatewayAndRoutePrivateSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createNatGateway", isVerbose)

	natGatewayAllocationId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-addresses"}, fmt.Sprintf(`.Addresses[] | select(.PublicIp == "%s") | .AllocationId`, prjPair.Live.Network.PublicSubnet.NatGatewayPublicIp), false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// Check if it's already there
	foundNatGatewayIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-nat-gateways",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.PublicSubnet.NatGatewayName},
		".NatGateways[0].NatGatewayId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if prjPair.Live.Network.PublicSubnet.NatGatewayId == "" {
		// If it was already created, save it for future use, but do not create
		if foundNatGatewayIdByName != "" {
			lb.Add(fmt.Sprintf("nat gateway %s(%s) already there, updating project\n", prjPair.Live.Network.PublicSubnet.NatGatewayName, foundNatGatewayIdByName))
			prjPair.SetNatGatewayId(foundNatGatewayIdByName)
		}
	} else {
		if foundNatGatewayIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested nat gateway id %s not present, consider removing this id from the project file", prjPair.Live.Network.PublicSubnet.NatGatewayId))
		} else if prjPair.Live.Network.PublicSubnet.NatGatewayId != foundNatGatewayIdByName {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested nat gateway id %s not matching existing nat gateway id %s", prjPair.Live.Network.PublicSubnet.NatGatewayId, foundNatGatewayIdByName))
		}
	}

	// Create NAT gateway in the public subnet
	natGatewayId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-nat-gateway",
		"--subnet-id", prjPair.Live.Network.PublicSubnet.Id,
		"--allocation-id", natGatewayAllocationId,
		"--tag-specification", fmt.Sprintf("ResourceType=natgateway,Tags=[{Key=Name,Value=%s}]", prjPair.Live.Network.PublicSubnet.NatGatewayName)},
		".NatGateway.NatGatewayId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	startWaitTs := time.Now()
	for {
		status, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-nat-gateways",
			"--filter", "Name=nat-gateway-id,Values=" + natGatewayId},
			".NatGateways[0].State", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
		if status == "" {
			return lb.Complete(fmt.Errorf("aws returned empty nat gateway status for %s", natGatewayId))
		}
		if status == "available" {
			break
		}
		if status != "pending" {
			return lb.Complete(fmt.Errorf("nat gateway %s was built, but the status is %s", natGatewayId, status))
		}
		if time.Since(startWaitTs).Seconds() > float64(prjPair.Live.Timeouts.OpenstackInstanceCreation) {
			return lb.Complete(fmt.Errorf("giving up after waiting for nat gateway %s to be created", natGatewayId))
		}
		time.Sleep(10 * time.Second)
	}

	prjPair.SetNatGatewayId(natGatewayId)

	// Create new route table id for this vpc
	routeTableId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-route-table",
		"--vpc-id", prjPair.Live.Network.Id,
		"--tag-specification", fmt.Sprintf("ResourceType=route-table,Tags=[{Key=Name,Value=%s_rt_to_natgw}]", prjPair.Live.Network.PrivateSubnet.Name)},
		".RouteTable.RouteTableId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	prjPair.SetRouteTableToNat(routeTableId)

	// Associate this route table with the private subnet
	assocId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "associate-route-table",
		"--route-table-id", routeTableId,
		"--subnet-id", prjPair.Live.Network.PrivateSubnet.Id},
		".AssociationId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	lb.Add(fmt.Sprintf("associated route table %s with subnet %s: %s", routeTableId, prjPair.Live.Network.PrivateSubnet.Id, assocId))

	// Add a record to a route table: tell all outbound 0.0.0.0/0 traffic to go through this nat gateway:
	result, er := ExecLocalAndGetJsonBool(&prjPair.Live, "aws", []string{"ec2", "create-route",
		"--route-table-id", routeTableId,
		"--destination-cidr-block", "0.0.0.0/0",
		"--nat-gateway-id", natGatewayId},
		".Return")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if result != true {
		if er.Error != nil {
			return lb.Complete(fmt.Errorf("route creation returned false"))
		}
	}
	lb.Add(fmt.Sprintf("route table %s in private subnet %s points to natgw %s", routeTableId, prjPair.Live.Network.PrivateSubnet.Id, natGatewayId))

	return lb.Complete(nil)
}

func createInternetGatewayAndRoutePublicSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createInternetGatewayAndRoutePublicSubnet", isVerbose)
	if prjPair.Live.Network.Router.Name == "" {
		return lb.Complete(fmt.Errorf("network gateway name cannot be empty"))
	}

	// Check if it's already there
	foundRouterIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-internet-gateways",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.Router.Name},
		".InternetGateways[0].InternetGatewayId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if prjPair.Live.Network.Router.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundRouterIdByName != "" {
			lb.Add(fmt.Sprintf("network gateway %s(%s) already there, updating project\n", prjPair.Live.Network.Router.Name, foundRouterIdByName))
			prjPair.SetRouterId(foundRouterIdByName)
		}
	} else {
		if foundRouterIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested network gateway id %s not present, consider removing this id from the project file", prjPair.Live.Network.Router.Id))
		} else if prjPair.Live.Network.Router.Id != foundRouterIdByName {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested network gateway id %s not matching existing network gateway id %s", prjPair.Live.Network.Router.Id, foundRouterIdByName))
		}
	}

	if prjPair.Live.Network.Router.Id == "" {
		// Create
		newId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-internet-gateway",
			"--tag-specification", fmt.Sprintf("ResourceType=internet-gateway,Tags=[{Key=Name,Value=%s}]", prjPair.Live.Network.Router.Name)},
			".InternetGateway.InternetGatewayId", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		lb.Add(fmt.Sprintf("created network gateway %s(%s)", prjPair.Live.Network.Router.Name, newId))
		prjPair.SetRouterId(newId)

	} else {
		lb.Add(fmt.Sprintf("network gateway %s(%s) already there, no need to create", prjPair.Live.Network.Router.Name, foundRouterIdByName))
	}

	// Make sure the gateway is attached to vpc
	attachedVpcId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-internet-gateways",
		"--filter", "Name=internet-gateway-id,Values=" + prjPair.Live.Network.Router.Id},
		".InternetGateways[0].Attachments[0].VpcId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	if attachedVpcId == "" {
		er := ExecLocal(&prjPair.Live, "aws", []string{"ec2", "attach-internet-gateway",
			"--internet-gateway-id", prjPair.Live.Network.Router.Id,
			"--vpc-id", prjPair.Live.Network.Id},
			prjPair.Live.CliEnvVars, "")
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
	} else if attachedVpcId != prjPair.Live.Network.Id {
		return lb.Complete(fmt.Errorf("network gateway %s seems to be attached to a wrong vpc %s already\n", prjPair.Live.Network.Router.Name, attachedVpcId))
	} else {
		lb.Add(fmt.Sprintf("network gateway %s seems to be attached to vpc already\n", prjPair.Live.Network.Router.Name))
	}

	// Obtain route table id for this vpc (it was automatically created for us and marked as 'main')
	routeTableId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-route-tables",
		"--filter", "Name=association.main,Values=true", fmt.Sprintf("Name=vpc-id,Values=%s", prjPair.Live.Network.Id)},
		".RouteTables[0].RouteTableId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// (optional) tag this route table for operator's convenience
	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "create-tags",
		"--resources", routeTableId,
		"--tags", fmt.Sprintf("Key=Name,Value=%s_rt_to_igw", prjPair.Live.Network.PublicSubnet.Name)},
		prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// Associate this route table with the public subnet
	assocId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "associate-route-table",
		"--route-table-id", routeTableId,
		"--subnet-id", prjPair.Live.Network.PublicSubnet.Id},
		".AssociationId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	lb.Add(fmt.Sprintf("associated route table %s with subnet %s: %s", routeTableId, prjPair.Live.Network.PublicSubnet.Id, assocId))

	// Add a record to a route table: tell all outbound 0.0.0.0/0 traffic to go through this internet gateway:
	result, er := ExecLocalAndGetJsonBool(&prjPair.Live, "aws", []string{"ec2", "create-route",
		"--route-table-id", routeTableId,
		"--destination-cidr-block", "0.0.0.0/0",
		"--gateway-id", prjPair.Live.Network.Router.Id},
		".Return")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if result != true {
		if er.Error != nil {
			return lb.Complete(fmt.Errorf("route creation returned false"))
		}
	}

	lb.Add(fmt.Sprintf("route table %s in public subnet %s points to igw %s", routeTableId, prjPair.Live.Network.PublicSubnet.Id, prjPair.Live.Network.Router.Id))

	return lb.Complete(nil)
}

func (*AwsDeployProvider) CreateNetworking(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	sb := strings.Builder{}

	logMsg, err := createAwsVpc(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = waitForAwsVpcToBeCreated(&prjPair.Live, prjPair.Live.Network.Id, prjPair.Live.Timeouts.OpenstackInstanceCreation, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = createAwsPrivateSubnet(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = createAwsPublicSubnet(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = createInternetGatewayAndRoutePublicSubnet(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = createNatGatewayAndRoutePrivateSubnet(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	return LogMsg(sb.String()), nil
}

func deleteInternetGateway(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("deleteInternetGateway", isVerbose)
	if prjPair.Live.Network.Router.Name == "" {
		lb.Add("internet gateway empty, nothing to delete")
		return lb.Complete(nil)
	}

	// Check if it's already there
	foundRouterIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-internet-gateways",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.Router.Name},
		".InternetGateways[0].InternetGatewayId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	if foundRouterIdByName == "" {
		lb.Add(fmt.Sprintf("network gateway %s not found, nothing to delete", prjPair.Live.Network.Router.Name))
		prjPair.SetRouterId("")
		return lb.Complete(nil)
	}

	if prjPair.Live.Network.Router.Id != "" && foundRouterIdByName != prjPair.Live.Network.Router.Id {
		lb.Add(fmt.Sprintf("network gateway %s not found, but another network gateway with proper name found %s, not sure what to delete", prjPair.Live.Network.Router.Name, foundRouterIdByName))
		return lb.Complete(nil)
	}

	// Verify it's attached
	// Check if it's already there
	state, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-internet-gateways",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.Router.Name},
		".InternetGateways[0].Attachments[0].State", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if state == "available" {
		// This may fail with: Network vpc-... has some mapped public address(es). Please unmap those public address(es) before detaching the gateway.
		// This is probably because the delition of the natgw is still in progress. Call again after a while.
		er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "detach-internet-gateway",
			"--internet-gateway-id", prjPair.Live.Network.Router.Id,
			"--vpc-id", prjPair.Live.Network.Id},
			prjPair.Live.CliEnvVars, "")
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
		lb.Add(fmt.Sprintf("detached internet gateway %s from vpc %s", prjPair.Live.Network.Router.Id, prjPair.Live.Network.Id))
	} else {
		lb.Add(fmt.Sprintf("internet gateway %s was not attached, nothing to do", prjPair.Live.Network.Router.Id))
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-internet-gateway",
		"--internet-gateway-id", prjPair.Live.Network.Router.Id},
		prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted internet gateway %s", prjPair.Live.Network.Router.Name))
	prjPair.SetRouterId("")

	return lb.Complete(nil)
}

func deleteNatGateway(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("deleteNatGateway", isVerbose)
	if prjPair.Live.Network.PublicSubnet.NatGatewayId == "" {
		lb.Add("nat gateway id empty, nothing to delete")
		return lb.Complete(nil)
	}

	// Check if it's already there
	foundNatGatewayIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-nat-gateways",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.PublicSubnet.NatGatewayName, "Name=state,Values=available"},
		".NatGateways[0].NatGatewayId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	if foundNatGatewayIdByName == "" {
		lb.Add(fmt.Sprintf("nat gateway %s not found, nothing to delete", prjPair.Live.Network.PublicSubnet.NatGatewayName))
		prjPair.SetNatGatewayId("")
		return lb.Complete(nil)
	}

	if prjPair.Live.Network.Router.Id != "" && foundNatGatewayIdByName != prjPair.Live.Network.PublicSubnet.NatGatewayId {
		lb.Add(fmt.Sprintf("nat gateway %s not found, but another nat gateway with proper name found %s, not sure what to delete", prjPair.Live.Network.PublicSubnet.NatGatewayName, foundNatGatewayIdByName))
		return lb.Complete(nil)
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-nat-gateway",
		"--nat-gateway-id", prjPair.Live.Network.PublicSubnet.NatGatewayId},
		prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// Wait until natgw is trully gone, otherwise vpc deletion will choke
	startWaitTs := time.Now()
	for {
		status, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-nat-gateways",
			"--nat-gateway-ids", foundNatGatewayIdByName},
			".NatGateways[0].State", true)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
		if status == "" {
			return lb.Complete(fmt.Errorf("aws returned empty natgw status for %s", foundNatGatewayIdByName))
		}
		if status == "deleted" {
			break
		}
		if status != "deleting" {
			return lb.Complete(fmt.Errorf("natgw %s was being deleted, but the status is %s", foundNatGatewayIdByName, status))
		}
		if time.Since(startWaitTs).Seconds() > float64(prjPair.Live.Timeouts.OpenstackInstanceCreation) {
			return lb.Complete(fmt.Errorf("giving up after waiting for natgw %s to be deleted", foundNatGatewayIdByName))
		}
		time.Sleep(10 * time.Second)
	}

	lb.Add(fmt.Sprintf("deleted nat gateway %s", prjPair.Live.Network.Router.Name))
	prjPair.SetNatGatewayId("")

	return lb.Complete(nil)
}

func deleteAwsPrivateSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("deleteAwsPrivateSubnet", isVerbose)
	if prjPair.Live.Network.PrivateSubnet.Name == "" {
		return lb.Complete(fmt.Errorf("subnet name cannot be empty"))
	}

	foundSubnetIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-subnets",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.PrivateSubnet.Name},
		".Subnets[0].SubnetId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if foundSubnetIdByName == "" {
		lb.Add(fmt.Sprintf("subnet %s not found, nothing to delete", prjPair.Live.Network.PrivateSubnet.Name))
		prjPair.SetPrivateSubnetId("")
		return lb.Complete(nil)
	}

	if prjPair.Live.Network.PrivateSubnet.Id != "" && foundSubnetIdByName != prjPair.Live.Network.PrivateSubnet.Id {
		lb.Add(fmt.Sprintf("subnet %s not found, but another subnet with proper name found %s, not sure what to delete", prjPair.Live.Network.PrivateSubnet.Name, foundSubnetIdByName))
		return lb.Complete(nil)
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-subnet",
		"--subnet-id", prjPair.Live.Network.PrivateSubnet.Id},
		prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted subnet %s", prjPair.Live.Network.PrivateSubnet.Name))
	prjPair.SetPrivateSubnetId("")

	return lb.Complete(nil)
}

func deleteAwsPublicSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("deleteAwsPublicSubnet", isVerbose)
	if prjPair.Live.Network.PublicSubnet.Name == "" {
		return lb.Complete(fmt.Errorf("subnet name cannot be empty"))
	}

	foundSubnetIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-subnets",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.PublicSubnet.Name},
		".Subnets[0].SubnetId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if foundSubnetIdByName == "" {
		lb.Add(fmt.Sprintf("subnet %s not found, nothing to delete", prjPair.Live.Network.PublicSubnet.Name))
		prjPair.SetPublicSubnetId("")
		return lb.Complete(nil)
	}

	if prjPair.Live.Network.PublicSubnet.Id != "" && foundSubnetIdByName != prjPair.Live.Network.PublicSubnet.Id {
		lb.Add(fmt.Sprintf("subnet %s not found, but another subnet with proper name found %s, not sure what to delete", prjPair.Live.Network.PublicSubnet.Name, foundSubnetIdByName))
		return lb.Complete(nil)
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-subnet",
		"--subnet-id", prjPair.Live.Network.PublicSubnet.Id},
		prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted subnet %s", prjPair.Live.Network.PublicSubnet.Name))
	prjPair.SetPublicSubnetId("")

	return lb.Complete(nil)
}

func deleteAwsVpc(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("deleteAwsVpc", isVerbose)
	if prjPair.Live.Network.Name == "" {
		return lb.Complete(fmt.Errorf("network name cannot be empty"))
	}

	foundNetworkIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-vpcs",
		"--filter", "Name=tag:Name,Values=" + prjPair.Live.Network.Name},
		".Vpcs[0].VpcId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if foundNetworkIdByName == "" {
		lb.Add(fmt.Sprintf("network %s not found, nothing to delete", prjPair.Live.Network.Name))
		prjPair.SetNetworkId("")
		return lb.Complete(nil)
	}

	if prjPair.Live.Network.Id != "" && foundNetworkIdByName != prjPair.Live.Network.Id {
		lb.Add(fmt.Sprintf("network %s not found, but another network with proper name found %s, not sure what to delete", prjPair.Live.Network.Name, foundNetworkIdByName))
		return lb.Complete(nil)
	}

	if prjPair.Live.Network.PrivateSubnet.RouteTableToNat != "" {
		// Delete the route table pointing to natgw (if we don't, AWS will consider them as dependencies and will not delete vpc)
		er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-route-table",
			"--route-table-id", prjPair.Live.Network.PrivateSubnet.RouteTableToNat},
			prjPair.Live.CliEnvVars, "")
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-vpc", "--vpc-id", prjPair.Live.Network.Id}, prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted network %s, updating project file", prjPair.Live.Network.Name))
	prjPair.SetNetworkId("")

	return lb.Complete(nil)
}

func (*AwsDeployProvider) DeleteNetworking(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	sb := strings.Builder{}

	logMsg, err := deleteNatGateway(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = deleteInternetGateway(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = deleteAwsPublicSubnet(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = deleteAwsPrivateSubnet(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = deleteAwsVpc(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	return LogMsg(sb.String()), nil
}
