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

	newFloatingIp, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "allocate-address"}, ".PublicIp", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	prjPair.SetSshExternalIp(newFloatingIp)

	reportPublicIp(&prjPair.Live)

	return lb.Complete(nil)
}

func (*AwsDeployProvider) DeleteFloatingIp(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.DeleteFloatingIp", isVerbose)
	if prjPair.Live.SshConfig.ExternalIpAddress == "" {
		return lb.Complete(fmt.Errorf("cannot delete floating ip, ssh_config external_ip_address is already empty"))
	}

	// {
	// 	"Addresses": [
	// 		{
	// 			"PublicIp": "35.172.255.76",
	// 			"AllocationId": "eipalloc-0e4b1b8ec4eb983d4",
	// 			"Domain": "vpc",
	// 			"PublicIpv4Pool": "amazon",
	// 			"NetworkBorderGroup": "us-east-1"
	// 		}
	// 	]
	// }

	allocationId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-addresses"}, fmt.Sprintf(`.Addresses | select(any(.PublicIp == "%s")) | .[0].AllocationId`, prjPair.Live.SshConfig.ExternalIpAddress), false)
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
	return lb.Complete(nil)
}

func createAwsVpc(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createAwsVpc", isVerbose)
	if prjPair.Live.Network.Name == "" {
		return lb.Complete(fmt.Errorf("network name cannot be empty"))
	}

	// Check if it's already there
	foundNetworkIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-vpcs",
		"--filter", "Name=tag:name,Values=" + prjPair.Live.Network.Name},
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
		"--cidr-block", prjPair.Live.Network.Subnet.Cidr,
		"--tag-specification", fmt.Sprintf("ResourceType=vpc,Tags=[{Key=name,Value=%s}]", prjPair.Live.Network.Name)},
		".Vpc.VpcId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// Wait until it's there
	startWaitTs := time.Now()
	for {
		status, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-vpcs",
			"--filter", "Name=vpc-id,Values=" + newId},
			".Vpcs[0].State", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
		if status == "" {
			return lb.Complete(fmt.Errorf("aws returned empty vpc status for %s(%s)", prjPair.Live.Network.Name, newId))
		}
		if status == "available" {
			break
		}
		if status != "pending" {
			return lb.Complete(fmt.Errorf("vpc %s(%s) was built, but the status is %s", prjPair.Live.Network.Name, newId, status))
		}
		if time.Since(startWaitTs).Seconds() > float64(prjPair.Live.Timeouts.OpenstackInstanceCreation) {
			return lb.Complete(fmt.Errorf("giving up after waiting for vpc %s(%s) to be created", prjPair.Live.Network.Name, newId))
		}
		time.Sleep(10 * time.Second)
	}

	lb.Add(fmt.Sprintf("created network %s(%s)\n", prjPair.Live.Network.Name, newId))
	prjPair.SetNetworkId(newId)

	return lb.Complete(nil)
}

func createAwsSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createAwsSubnet", isVerbose)
	if prjPair.Live.Network.Subnet.Name == "" || prjPair.Live.Network.Subnet.Cidr == "" || prjPair.Live.Network.Id == "" {
		return lb.Complete(fmt.Errorf("subnet name(%s) and cidr(%s) and vpc id(%s) cannot be empty", prjPair.Live.Network.Subnet.Name, prjPair.Live.Network.Subnet.Cidr, prjPair.Live.Network.Id))
	}

	// Check if it's already there
	foundSubnetIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-subnets",
		"--filter", "Name=tag:name,Values=" + prjPair.Live.Network.Subnet.Name},
		".Subnets[0].SubnetId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if prjPair.Live.Network.Subnet.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundSubnetIdByName != "" {
			lb.Add(fmt.Sprintf("subnet %s(%s) already there, updating project", prjPair.Live.Network.Subnet.Name, foundSubnetIdByName))
			prjPair.SetSubnetId(foundSubnetIdByName)

		}
	} else {
		if foundSubnetIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested subnet id %s not present, consider removing this id from the project file", prjPair.Live.Network.Subnet.Id))
		} else if foundSubnetIdByName != prjPair.Live.Network.Subnet.Id {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested subnet id %s not matching existing id %s", prjPair.Live.Network.Subnet.Id, foundSubnetIdByName))
		}
	}

	if prjPair.Live.Network.Subnet.Id != "" {
		lb.Add(fmt.Sprintf("subnet %s(%s) already there, no need to create", prjPair.Live.Network.Subnet.Name, foundSubnetIdByName))
		return lb.Complete(nil)
	}

	// Create
	newId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-subnet",
		"--vpc-id", prjPair.Live.Network.Id,
		"--cidr-block", prjPair.Live.Network.Subnet.Cidr,
		"--tag-specification", fmt.Sprintf("ResourceType=subnet,Tags=[{Key=name,Value=%s}]", prjPair.Live.Network.Subnet.Name)},
		".Subnet.SubnetId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// TODO: dhcp options and allocation pools?

	lb.Add(fmt.Sprintf("created subnet %s(%s)", prjPair.Live.Network.Subnet.Name, newId))
	prjPair.SetSubnetId(newId)

	return lb.Complete(nil)
}

func createAwsGateway(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createAwsGateway", isVerbose)
	if prjPair.Live.Network.Router.Name == "" {
		return lb.Complete(fmt.Errorf("router name and ExternalGatewayNetworkName cannot be empty"))
	}

	// Check if it's already there
	foundRouterIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-internet-gateways",
		"--filter", "Name=tag:name,Values=" + prjPair.Live.Network.Router.Name},
		".InternetGateways[0].InternetGatewayId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if prjPair.Live.Network.Router.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundRouterIdByName != "" {
			lb.Add(fmt.Sprintf("router %s(%s) already there, updating project\n", prjPair.Live.Network.Router.Name, foundRouterIdByName))
			prjPair.SetRouterId(foundRouterIdByName)
		}
	} else {
		if foundRouterIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested router id %s not present, consider removing this id from the project file", prjPair.Live.Network.Router.Id))
		} else if prjPair.Live.Network.Router.Id != foundRouterIdByName {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested router id %s not matching existing router id %s", prjPair.Live.Network.Router.Id, foundRouterIdByName))
		}
	}

	if prjPair.Live.Network.Router.Id == "" {
		// Create
		newId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-internet-gateway",
			"--tag-specification", fmt.Sprintf("ResourceType=internet-gateway,Tags=[{Key=name,Value=%s}]", prjPair.Live.Network.Router.Name)},
			".InternetGateway.InternetGatewayId", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		lb.Add(fmt.Sprintf("created router %s(%s)", prjPair.Live.Network.Router.Name, newId))
		prjPair.SetRouterId(newId)

	} else {
		lb.Add(fmt.Sprintf("router %s(%s) already there, no need to create", prjPair.Live.Network.Router.Name, foundRouterIdByName))
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
		return lb.Complete(fmt.Errorf("router %s seems to be attached to a wrong vpc %s already\n", prjPair.Live.Network.Router.Name, attachedVpcId))
	} else {
		lb.Add(fmt.Sprintf("router %s seems to be attached to vpc already\n", prjPair.Live.Network.Router.Name))
	}

	// Obtain route table id for this vpc (it was automatically created for us)
	routeTableId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-route-tables",
		"--filter", "Name=vpc-id,Values=" + prjPair.Live.Network.Id},
		".RouteTables[0].RouteTableId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// Add a record to a route table: tell all outbound 0.0.0.0/0 traffic to go through this gateway:
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

	return lb.Complete(nil)
}

func (*AwsDeployProvider) CreateNetworking(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	sb := strings.Builder{}

	logMsg, err := createAwsVpc(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = createAwsSubnet(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = createAwsGateway(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	return LogMsg(sb.String()), nil
}

func deleteAwsGateway(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("deleteAwsGateway", isVerbose)
	if prjPair.Live.Network.Router.Name == "" {
		return lb.Complete(fmt.Errorf("router name and ExternalGatewayNetworkName cannot be empty"))
	}

	// Check if it's already there
	foundRouterIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-internet-gateways",
		"--filter", "Name=tag:name,Values=" + prjPair.Live.Network.Router.Name},
		".InternetGateways[0].InternetGatewayId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	if foundRouterIdByName == "" {
		lb.Add(fmt.Sprintf("router %s not found, nothing to delete", prjPair.Live.Network.Router.Name))
		prjPair.SetRouterId("")
		return lb.Complete(nil)
	}

	if prjPair.Live.Network.Router.Id != "" && foundRouterIdByName != prjPair.Live.Network.Router.Id {
		lb.Add(fmt.Sprintf("router %s not found, but another router with proper name found %s, not sure what to delete", prjPair.Live.Network.Router.Name, foundRouterIdByName))
		return lb.Complete(nil)
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "detach-internet-gateway",
		"--internet-gateway-id", prjPair.Live.Network.Router.Id,
		"--vpc-id", prjPair.Live.Network.Id},
		prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-internet-gateway",
		"--internet-gateway-id", prjPair.Live.Network.Router.Id},
		prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted router %s", prjPair.Live.Network.Router.Name))
	prjPair.SetRouterId("")

	return lb.Complete(nil)
}

func deleteAwsSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("deleteAwsSubnet", isVerbose)
	if prjPair.Live.Network.Subnet.Name == "" {
		return lb.Complete(fmt.Errorf("subnet name cannot be empty"))
	}

	foundSubnetIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-subnets",
		"--filter", "Name=tag:name,Values=" + prjPair.Live.Network.Subnet.Name},
		".Subnets[0].SubnetId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if foundSubnetIdByName == "" {
		lb.Add(fmt.Sprintf("subnet %s not found, nothing to delete", prjPair.Live.Network.Subnet.Name))
		prjPair.SetSubnetId("")
		return lb.Complete(nil)
	}

	if prjPair.Live.Network.Subnet.Id != "" && foundSubnetIdByName != prjPair.Live.Network.Subnet.Id {
		lb.Add(fmt.Sprintf("subnet %s not found, but another subnet with proper name found %s, not sure what to delete", prjPair.Live.Network.Subnet.Name, foundSubnetIdByName))
		return lb.Complete(nil)
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-subnet",
		"--subnet-id", prjPair.Live.Network.Subnet.Id},
		prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted subnet %s", prjPair.Live.Network.Subnet.Name))
	prjPair.SetSubnetId("")

	return lb.Complete(nil)
}

func deleteAwsVpc(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("deleteAwsVpc", isVerbose)
	if prjPair.Live.Network.Name == "" {
		return lb.Complete(fmt.Errorf("network name cannot be empty"))
	}

	foundNetworkIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-vpcs",
		"--filter", "Name=tag:name,Values=" + prjPair.Live.Network.Name},
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

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-vpc",
		"--vpc-id", prjPair.Live.Network.Id},
		prjPair.Live.CliEnvVars, "")
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

	logMsg, err := deleteAwsGateway(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = deleteAwsSubnet(prjPair, isVerbose)
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
