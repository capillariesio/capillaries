package deploy

import (
	"encoding/json"
	"fmt"
	"strings"
)

func CreateSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("CreateSubnet", isVerbose)
	if prjPair.Live.Network.Subnet.Name == "" || prjPair.Live.Network.Subnet.Cidr == "" {
		return lb.Complete(fmt.Errorf("subnet name and cidr cannot be empty"))
	}

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"subnet", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | ID                                   | Name                          | Network                              | Subnet                |
	// | 30a21631-d188-49f5-a7e5-faa3a5a5b50a | sample_deployment_name_subnet | fe181240-b89e-49c6-8b10-9fba7f4a2d6a | 10.5.0.0/16           |
	foundSubnetIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Subnet.Name)
	foundNetworkIdByName := FindOpenstackColumnValue(rows, "Network", "Name", prjPair.Live.Network.Subnet.Name)
	foundCidrByName := FindOpenstackColumnValue(rows, "Subnet", "Name", prjPair.Live.Network.Subnet.Name)
	if foundNetworkIdByName != "" && prjPair.Live.Network.Id != foundNetworkIdByName {
		return lb.Complete(fmt.Errorf("requested subnet network id %s not matching existing network id %s", prjPair.Live.Network.Id, foundNetworkIdByName))
	}
	if foundCidrByName != "" && prjPair.Live.Network.Subnet.Cidr != foundCidrByName {
		return lb.Complete(fmt.Errorf("requested subnet cidr %s not matching existing cidr %s", prjPair.Live.Network.Subnet.Cidr, foundCidrByName))
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

	subnetParams := []string{
		"subnet", "create", prjPair.Live.Network.Subnet.Name,
		"--subnet-range", prjPair.Live.Network.Subnet.Cidr,
		"--network", prjPair.Live.Network.Name}
	if prjPair.Live.Network.Subnet.AllocationPool == "" {
		subnetParams = append(subnetParams, "--no-dhcp")
	} else {
		subnetParams = append(subnetParams, "--dhcp")
		subnetParams = append(subnetParams, "--allocation-pool")
		subnetParams = append(subnetParams, prjPair.Live.Network.Subnet.AllocationPool)
	}

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", subnetParams)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | Field                | Value                                |
	// | cidr                 | 10.5.0.0/16                          |
	// | gateway_ip           | 10.5.0.1                             |
	// | id                   | 30a21631-d188-49f5-a7e5-faa3a5a5b50a |
	// | name                 | sample_deployment_name_subnet        |
	// | network_id           | fe181240-b89e-49c6-8b10-9fba7f4a2d6a |
	newId := FindOpenstackFieldValue(rows, "id")
	if newId == "" {
		return lb.Complete(fmt.Errorf("openstack returned empty subnet id"))
	}

	lb.Add(fmt.Sprintf("created subnet %s(%s)", prjPair.Live.Network.Subnet.Name, newId))
	prjPair.SetSubnetId(newId)

	return lb.Complete(nil)
}

func DeleteSubnet(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("DeleteSubnet", isVerbose)
	if prjPair.Live.Network.Subnet.Name == "" {
		return lb.Complete(fmt.Errorf("subnet name and cidr cannot be empty"))
	}

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"subnet", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | ID                                   | Name                          | Network                              | Subnet                |
	// | 30a21631-d188-49f5-a7e5-faa3a5a5b50a | sample_deployment_name_subnet | fe181240-b89e-49c6-8b10-9fba7f4a2d6a | 10.5.0.0/16           |
	foundSubnetIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Subnet.Name)
	if foundSubnetIdByName == "" {
		lb.Add(fmt.Sprintf("subnet %s not found, nothing to delete", prjPair.Live.Network.Subnet.Name))
		prjPair.SetSubnetId("")
		return lb.Complete(nil)
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"subnet", "delete", prjPair.Live.Network.Subnet.Name})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted subnet %s", prjPair.Live.Network.Subnet.Name))
	prjPair.SetSubnetId("")

	return lb.Complete(nil)
}

func CreateNetwork(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("CreateNetwork", isVerbose)
	if prjPair.Live.Network.Name == "" {
		return lb.Complete(fmt.Errorf("network name cannot be empty"))
	}

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"network", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | ID                                   | Name                           | Subnets                                   |
	// | e098d02f-bb35-4085-ae12-664aad3d9c52 | public                         | 0be66687-9358-46cd-9093-9ce62cb4ece7, ... |
	// | fe181240-b89e-49c6-8b10-9fba7f4a2d6a | sample_deployment_name_network | 30a21631-d188-49f5-a7e5-faa3a5a5b50a      |
	foundNetworkIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Name)
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

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"network", "create", prjPair.Live.Network.Name})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | Field                     | Value                                |
	// | id                        | fe181240-b89e-49c6-8b10-9fba7f4a2d6a |
	// | name                      | sample_deployment_name_network       |
	newId := FindOpenstackFieldValue(rows, "id")
	if newId == "" {
		return lb.Complete(fmt.Errorf("openstack returned empty network id"))
	}

	lb.Add(fmt.Sprintf("created network %s(%s)\n", prjPair.Live.Network.Name, newId))
	prjPair.SetNetworkId(newId)
	return lb.Complete(nil)
}

func DeleteNetwork(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("DeleteNetwork", isVerbose)
	if prjPair.Live.Network.Name == "" {
		return lb.Complete(fmt.Errorf("network name cannot be empty"))
	}

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"network", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | ID                                   | Name                           | Subnets                                   |
	// | e098d02f-bb35-4085-ae12-664aad3d9c52 | public                         | 0be66687-9358-46cd-9093-9ce62cb4ece7, ... |
	// | fe181240-b89e-49c6-8b10-9fba7f4a2d6a | sample_deployment_name_network | 30a21631-d188-49f5-a7e5-faa3a5a5b50a      |
	foundNetworkIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Name)
	if foundNetworkIdByName == "" {
		lb.Add(fmt.Sprintf("network %s not found, nothing to delete", prjPair.Live.Network.Name))
		prjPair.SetNetworkId("")
		return lb.Complete(nil)
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"network", "delete", prjPair.Live.Network.Name})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted network %s, updating project file", prjPair.Live.Network.Name))
	prjPair.SetNetworkId("")

	return lb.Complete(nil)
}

func CreateRouter(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("CreateRouter", isVerbose)
	if prjPair.Live.Network.Router.Name == "" || prjPair.Live.Network.Router.ExternalGatewayNetworkName == "" {
		return lb.Complete(fmt.Errorf("router name and ExternalGatewayNetworkName cannot be empty"))
	}

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"router", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | ID                                   | Name                          | Status | State | Project	                      | Distributed | HA    |
	// | 79aa5ec5-41c2-4f15-b341-1659a783ea53 | sample_deployment_name_router | ACTIVE | UP    | 56ac58a4903a458dbd4ea2241afc9566 | True        | False |
	foundRouterIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Router.Name)

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
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"router", "create", prjPair.Live.Network.Router.Name})
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		// | Field                   | Value                                |
		// | id                      | 79aa5ec5-41c2-4f15-b341-1659a783ea53 |
		// | name                    | sample_deployment_name_router        |
		newId := FindOpenstackFieldValue(rows, "id")
		if newId == "" {
			return lb.Complete(fmt.Errorf("openstack returned empty router id"))
		}

		lb.Add(fmt.Sprintf("created router %s(%s)", prjPair.Live.Network.Router.Name, newId))
		prjPair.SetRouterId(newId)
	} else {
		lb.Add(fmt.Sprintf("router %s(%s) already there, no need to create", prjPair.Live.Network.Router.Name, foundRouterIdByName))
	}

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"router", "show", prjPair.Live.Network.Router.Name})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// Make sure router is associated with subnet

	// | interfaces_info         | [{"port_id": "d4c714f3-8569-47e4-8b34-4addeba02b48", "ip_address": "10.5.0.1", "subnet_id": "30a21631-d188-49f5-a7e5-faa3a5a5b50a"}]
	interfacesInfo := FindOpenstackFieldValue(rows, "interfaces_info")
	if strings.Contains(interfacesInfo, prjPair.Live.Network.Subnet.Id) {
		lb.Add(fmt.Sprintf("router %s seems to be associated with subnet\n", prjPair.Live.Network.Router.Name))
	} else {
		lb.Add(fmt.Sprintf("router %s needs to be associated with subnet\n", prjPair.Live.Network.Router.Name))
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"router", "add", "subnet", prjPair.Live.Network.Router.Name, prjPair.Live.Network.Subnet.Name})
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
	}

	// Make sure router is connected to internet

	// | external_gateway_info   | {"network_id": "e098d02f-bb35-4085-ae12-664aad3d9c52", "enable_snat": true, "external_fixed_ips": [{"subnet_id": "109e7c17-f963-4e1e-ba73-af363f59ae8f", "ip_address": "208.113.128.236"}, {"subnet_id": "5d1e9148-b023-4606-b959-2bff89412491", "ip_address": "2607:f298:5:101d:f816:3eff:fef6:f460"}]} |
	externalGatewayInfo := FindOpenstackFieldValue(rows, "external_gateway_info")
	if strings.Contains(externalGatewayInfo, "ip_address") {
		lb.Add(fmt.Sprintf("router %s seems to be connected to internet\n", prjPair.Live.Network.Router.Name))
	} else {
		lb.Add(fmt.Sprintf("router %s needs to be connected to internet\n", prjPair.Live.Network.Router.Name))
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"router", "set", "--external-gateway", prjPair.Live.Network.Router.ExternalGatewayNetworkName, prjPair.Live.Network.Router.Name})
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
	}

	return lb.Complete(nil)
}

type RouterInterfaceInfo struct {
	PortId    string `json:"port_id"`
	IpAddress string `json:"ip_address"`
	SubnetId  string `json:"subnet_id"`
}

func DeleteRouter(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("DeleteRouter", isVerbose)
	if prjPair.Live.Network.Router.Name == "" {
		return lb.Complete(fmt.Errorf("router name and ExternalGatewayNetworkName cannot be empty"))
	}

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"router", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | ID                                   | Name                          | Status | State | Project	                      | Distributed | HA    |
	// | 79aa5ec5-41c2-4f15-b341-1659a783ea53 | sample_deployment_name_router | ACTIVE | UP    | 56ac58a4903a458dbd4ea2241afc9566 | True        | False |
	foundRouterIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Router.Name)
	if foundRouterIdByName == "" {
		lb.Add(fmt.Sprintf("router %s not found, nothing to delete", prjPair.Live.Network.Router.Name))
		prjPair.SetRouterId("")
		return lb.Complete(nil)
	}

	// Retrieve interface info

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"router", "show", prjPair.Live.Network.Router.Name})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | interfaces_info         | [{"port_id": "d4c714f3-8569-47e4-8b34-4addeba02b48", "ip_address": "10.5.0.1", "subnet_id": "30a21631-d188-49f5-a7e5-faa3a5a5b50a"}]
	iterfacesJson := FindOpenstackFieldValue(rows, "interfaces_info")
	if iterfacesJson != "" {
		var interfaces []RouterInterfaceInfo
		err := json.Unmarshal([]byte(iterfacesJson), &interfaces)
		if err != nil {
			return lb.Complete(err)
		}

		for _, iface := range interfaces {
			_, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"router", "remove", "port", prjPair.Live.Network.Router.Name, iface.PortId})
			lb.Add(er.ToString())
			if er.Error != nil {
				return lb.Complete(fmt.Errorf("cannot remove router %s port %s: %s", prjPair.Live.Network.Router.Name, iface.PortId, er.Error))
			}
			lb.Add(fmt.Sprintf("removed router %s port %s", prjPair.Live.Network.Router.Name, iface.PortId))
		}
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"router", "delete", prjPair.Live.Network.Router.Name})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted router %s", prjPair.Live.Network.Router.Name))
	prjPair.SetRouterId("")

	return lb.Complete(nil)
}

func CreateNetworking(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	sb := strings.Builder{}

	logMsg, err := CreateNetwork(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = WaitForEntityToBeCreated(&prjPair.Live, "network", prjPair.Live.Network.Name, prjPair.Live.Network.Id, prjPair.Live.Timeouts.OpenstackInstanceCreation, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = CreateSubnet(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = CreateRouter(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = WaitForEntityToBeCreated(&prjPair.Live, "router", prjPair.Live.Network.Router.Name, prjPair.Live.Network.Router.Id, prjPair.Live.Timeouts.OpenstackInstanceCreation, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	return LogMsg(sb.String()), nil
}

func DeleteNetworking(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	sb := strings.Builder{}

	logMsg, err := DeleteRouter(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = DeleteSubnet(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = DeleteNetwork(prjPair, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	return LogMsg(sb.String()), nil
}
