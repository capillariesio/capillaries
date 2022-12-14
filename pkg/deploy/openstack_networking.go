package deploy

import (
	"encoding/json"
	"fmt"
	"strings"
)

func CreateSubnet(prjPair *ProjectPair, logChan chan string) error {
	if prjPair.Live.Network.Subnet.Name == "" || prjPair.Live.Network.Subnet.Cidr == "" {
		return fmt.Errorf("subnet name and cidr cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("CreateSubnet", sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"subnet", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                          | Network                              | Subnet                |
	// | 30a21631-d188-49f5-a7e5-faa3a5a5b50a | sample_deployment_name_subnet | fe181240-b89e-49c6-8b10-9fba7f4a2d6a | 10.5.0.0/16           |
	foundSubnetIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Subnet.Name)
	foundNetworkIdByName := FindOpenstackColumnValue(rows, "Network", "Name", prjPair.Live.Network.Subnet.Name)
	foundCidrByName := FindOpenstackColumnValue(rows, "Subnet", "Name", prjPair.Live.Network.Subnet.Name)
	if foundNetworkIdByName != "" && prjPair.Live.Network.Id != foundNetworkIdByName {
		return fmt.Errorf("requested subnet network id %s not matching existing network id %s", prjPair.Live.Network.Id, foundNetworkIdByName)
	}
	if foundCidrByName != "" && prjPair.Live.Network.Subnet.Cidr != foundCidrByName {
		return fmt.Errorf("requested subnet cidr %s not matching existing cidr %s", prjPair.Live.Network.Subnet.Cidr, foundCidrByName)
	}
	if prjPair.Live.Network.Subnet.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundSubnetIdByName != "" {
			sb.WriteString(fmt.Sprintf("subnet %s(%s) already there, updating project\n", prjPair.Live.Network.Subnet.Name, foundSubnetIdByName))
			prjPair.Template.Network.Subnet.Id = foundSubnetIdByName
			prjPair.Live.Network.Subnet.Id = foundSubnetIdByName
		}
	} else {
		if foundSubnetIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return fmt.Errorf("requested subnet id %s not present, consider removing this id from the project file", prjPair.Live.Network.Subnet.Id)
		} else if foundSubnetIdByName != prjPair.Live.Network.Subnet.Id {
			// It is already there, but has different id, complain
			return fmt.Errorf("requested subnet id %s not matching existing id %s", prjPair.Live.Network.Subnet.Id, foundSubnetIdByName)
		}
	}

	if prjPair.Live.Network.Subnet.Id != "" {
		sb.WriteString(fmt.Sprintf("subnet %s(%s) already there, no need to create\n", prjPair.Live.Network.Subnet.Name, foundSubnetIdByName))
		return nil
	}

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"subnet", "create", prjPair.Live.Network.Subnet.Name, "--subnet-range", prjPair.Live.Network.Subnet.Cidr, "--network", prjPair.Live.Network.Name, "--no-dhcp"})
	if er.Error != nil {
		return er.Error
	}

	// | Field                | Value                                |
	// | cidr                 | 10.5.0.0/16                          |
	// | gateway_ip           | 10.5.0.1                             |
	// | id                   | 30a21631-d188-49f5-a7e5-faa3a5a5b50a |
	// | name                 | sample_deployment_name_subnet        |
	// | network_id           | fe181240-b89e-49c6-8b10-9fba7f4a2d6a |
	newId := FindOpenstackFieldValue(rows, "id")
	if newId == "" {
		return fmt.Errorf("openstack returned empty subnet id")
	}

	sb.WriteString(fmt.Sprintf("created subnet %s(%s)\n", prjPair.Live.Network.Subnet.Name, newId))
	prjPair.Template.Network.Subnet.Id = newId
	prjPair.Live.Network.Subnet.Id = newId

	return nil
}

func DeleteSubnet(prjPair *ProjectPair, logChan chan string) error {
	if prjPair.Live.Network.Subnet.Name == "" {
		return fmt.Errorf("subnet name and cidr cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("DeleteSubnet", sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"subnet", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                          | Network                              | Subnet                |
	// | 30a21631-d188-49f5-a7e5-faa3a5a5b50a | sample_deployment_name_subnet | fe181240-b89e-49c6-8b10-9fba7f4a2d6a | 10.5.0.0/16           |
	foundSubnetIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Subnet.Name)
	if foundSubnetIdByName == "" {
		sb.WriteString(fmt.Sprintf("subnet %s not found, nothing to delete", prjPair.Live.Network.Subnet.Name))
		prjPair.Template.Network.Subnet.Id = ""
		prjPair.Live.Network.Subnet.Id = ""
		return nil
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"subnet", "delete", prjPair.Live.Network.Subnet.Name})
	if er.Error != nil {
		return er.Error
	}

	sb.WriteString(fmt.Sprintf("deleted subnet %s", prjPair.Live.Network.Subnet.Name))
	prjPair.Template.Network.Subnet.Id = ""
	prjPair.Live.Network.Subnet.Id = ""

	return nil
}

func CreateNetwork(prjPair *ProjectPair, logChan chan string) error {
	if prjPair.Live.Network.Name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("CreateNetwork", sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"network", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                           | Subnets                                   |
	// | e098d02f-bb35-4085-ae12-664aad3d9c52 | public                         | 0be66687-9358-46cd-9093-9ce62cb4ece7, ... |
	// | fe181240-b89e-49c6-8b10-9fba7f4a2d6a | sample_deployment_name_network | 30a21631-d188-49f5-a7e5-faa3a5a5b50a      |
	foundNetworkIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Name)
	if prjPair.Live.Network.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundNetworkIdByName != "" {
			sb.WriteString(fmt.Sprintf("network %s(%s) already there, updating project\n", prjPair.Live.Network.Name, foundNetworkIdByName))
			prjPair.Template.Network.Id = foundNetworkIdByName
			prjPair.Live.Network.Id = foundNetworkIdByName
		}
	} else {
		if foundNetworkIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return fmt.Errorf("requested network id %s not present, consider removing this id from the project file", prjPair.Live.Network.Id)
		} else if prjPair.Live.Network.Id != foundNetworkIdByName {
			// It is already there, but has different id, complain
			return fmt.Errorf("requested network id %s not matching existing network id %s", prjPair.Live.Network.Id, foundNetworkIdByName)
		}
	}

	if prjPair.Live.Network.Id != "" {
		sb.WriteString(fmt.Sprintf("network %s(%s) already there, no need to create\n", prjPair.Live.Network.Name, foundNetworkIdByName))
		return nil
	}

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"network", "create", prjPair.Live.Network.Name})
	if er.Error != nil {
		return er.Error
	}

	// | Field                     | Value                                |
	// | id                        | fe181240-b89e-49c6-8b10-9fba7f4a2d6a |
	// | name                      | sample_deployment_name_network       |
	newId := FindOpenstackFieldValue(rows, "id")
	if newId == "" {
		return fmt.Errorf("openstack returned empty network id")
	}

	sb.WriteString(fmt.Sprintf("created network %s(%s)\n", prjPair.Live.Network.Name, newId))
	prjPair.Template.Network.Id = newId
	prjPair.Live.Network.Id = newId
	return nil
}

func DeleteNetwork(prjPair *ProjectPair, logChan chan string) error {
	if prjPair.Live.Network.Name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("DeleteNetwork", sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"network", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                           | Subnets                                   |
	// | e098d02f-bb35-4085-ae12-664aad3d9c52 | public                         | 0be66687-9358-46cd-9093-9ce62cb4ece7, ... |
	// | fe181240-b89e-49c6-8b10-9fba7f4a2d6a | sample_deployment_name_network | 30a21631-d188-49f5-a7e5-faa3a5a5b50a      |
	foundNetworkIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Name)
	if foundNetworkIdByName == "" {
		sb.WriteString(fmt.Sprintf("network %s not found, nothing to delete", prjPair.Live.Network.Name))
		prjPair.Template.Network.Id = ""
		prjPair.Live.Network.Id = ""
		return nil
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"network", "delete", prjPair.Live.Network.Name})
	if er.Error != nil {
		return er.Error
	}

	sb.WriteString(fmt.Sprintf("deleted network %s, updating project file", prjPair.Live.Network.Name))
	prjPair.Template.Network.Id = ""
	prjPair.Live.Network.Id = ""

	return nil
}

func CreateRouter(prjPair *ProjectPair, logChan chan string) error {
	if prjPair.Live.Network.Router.Name == "" || prjPair.Live.Network.Router.ExternalGatewayNetworkName == "" {
		return fmt.Errorf("router name and ExternalGatewayNetworkName cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("CreateRouter", sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"router", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                          | Status | State | Project	                      | Distributed | HA    |
	// | 79aa5ec5-41c2-4f15-b341-1659a783ea53 | sample_deployment_name_router | ACTIVE | UP    | 56ac58a4903a458dbd4ea2241afc9566 | True        | False |
	foundRouterIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Router.Name)

	if prjPair.Live.Network.Router.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundRouterIdByName != "" {
			sb.WriteString(fmt.Sprintf("router %s(%s) already there, updating project\n", prjPair.Live.Network.Router.Name, foundRouterIdByName))
			prjPair.Template.Network.Router.Id = foundRouterIdByName
			prjPair.Live.Network.Router.Id = foundRouterIdByName
		}
	} else {
		if foundRouterIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return fmt.Errorf("requested router id %s not present, consider removing this id from the project file", prjPair.Live.Network.Router.Id)
		} else if prjPair.Live.Network.Router.Id != foundRouterIdByName {
			// It is already there, but has different id, complain
			return fmt.Errorf("requested router id %s not matching existing router id %s", prjPair.Live.Network.Router.Id, foundRouterIdByName)
		}
	}

	if prjPair.Live.Network.Router.Id == "" {
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"router", "create", prjPair.Live.Network.Router.Name})
		if er.Error != nil {
			return er.Error
		}

		// | Field                   | Value                                |
		// | id                      | 79aa5ec5-41c2-4f15-b341-1659a783ea53 |
		// | name                    | sample_deployment_name_router        |
		newId := FindOpenstackFieldValue(rows, "id")
		if newId == "" {
			return fmt.Errorf("openstack returned empty router id")
		}

		sb.WriteString(fmt.Sprintf("created router %s(%s)\n", prjPair.Live.Network.Router.Name, newId))
		prjPair.Template.Network.Router.Id = newId
		prjPair.Live.Network.Router.Id = newId
	} else {
		sb.WriteString(fmt.Sprintf("router %s(%s) already there, no need to create\n", prjPair.Live.Network.Router.Name, foundRouterIdByName))
	}

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"router", "show", prjPair.Live.Network.Router.Name})
	if er.Error != nil {
		return er.Error
	}

	// Make sure router is associated with subnet

	// | interfaces_info         | [{"port_id": "d4c714f3-8569-47e4-8b34-4addeba02b48", "ip_address": "10.5.0.1", "subnet_id": "30a21631-d188-49f5-a7e5-faa3a5a5b50a"}]
	interfacesInfo := FindOpenstackFieldValue(rows, "interfaces_info")
	if strings.Contains(interfacesInfo, prjPair.Live.Network.Subnet.Id) {
		sb.WriteString(fmt.Sprintf("router %s seems to be associated with subnet\n", prjPair.Live.Network.Router.Name))
	} else {
		sb.WriteString(fmt.Sprintf("router %s needs to be associated with subnet\n", prjPair.Live.Network.Router.Name))
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"router", "add", "subnet", prjPair.Live.Network.Router.Name, prjPair.Live.Network.Subnet.Name})
		if er.Error != nil {
			return er.Error
		}
	}

	// Make sure router is connected to internet

	// | external_gateway_info   | {"network_id": "e098d02f-bb35-4085-ae12-664aad3d9c52", "enable_snat": true, "external_fixed_ips": [{"subnet_id": "109e7c17-f963-4e1e-ba73-af363f59ae8f", "ip_address": "208.113.128.236"}, {"subnet_id": "5d1e9148-b023-4606-b959-2bff89412491", "ip_address": "2607:f298:5:101d:f816:3eff:fef6:f460"}]} |
	externalGatewayInfo := FindOpenstackFieldValue(rows, "external_gateway_info")
	if strings.Contains(externalGatewayInfo, "ip_address") {
		sb.WriteString(fmt.Sprintf("router %s seems to be connected to internet\n", prjPair.Live.Network.Router.Name))
	} else {
		sb.WriteString(fmt.Sprintf("router %s needs to be connected to internet\n", prjPair.Live.Network.Router.Name))
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"router", "set", "--external-gateway", prjPair.Live.Network.Router.ExternalGatewayNetworkName, prjPair.Live.Network.Router.Name})
		if er.Error != nil {
			return er.Error
		}
	}

	return nil
}

type RouterInterfaceInfo struct {
	PortId    string `json:"port_id"`
	IpAddress string `json:"ip_address"`
	SubnetId  string `json:"subnet_id"`
}

func DeleteRouter(prjPair *ProjectPair, logChan chan string) error {
	if prjPair.Live.Network.Router.Name == "" {
		return fmt.Errorf("router name and ExternalGatewayNetworkName cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("DeleteRouter", sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"router", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                          | Status | State | Project	                      | Distributed | HA    |
	// | 79aa5ec5-41c2-4f15-b341-1659a783ea53 | sample_deployment_name_router | ACTIVE | UP    | 56ac58a4903a458dbd4ea2241afc9566 | True        | False |
	foundRouterIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Network.Router.Name)
	if foundRouterIdByName == "" {
		sb.WriteString(fmt.Sprintf("router %s not found, nothing to delete", prjPair.Live.Network.Router.Name))
		prjPair.Template.Network.Router.Id = ""
		prjPair.Live.Network.Router.Id = ""
		return nil
	}

	// Retrieve interface info

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"router", "show", prjPair.Live.Network.Router.Name})
	if er.Error != nil {
		return er.Error
	}

	// | interfaces_info         | [{"port_id": "d4c714f3-8569-47e4-8b34-4addeba02b48", "ip_address": "10.5.0.1", "subnet_id": "30a21631-d188-49f5-a7e5-faa3a5a5b50a"}]
	iterfacesJson := FindOpenstackFieldValue(rows, "interfaces_info")
	if iterfacesJson != "" {
		var interfaces []RouterInterfaceInfo
		err := json.Unmarshal([]byte(iterfacesJson), &interfaces)
		if err != nil {
			return err
		}

		for _, iface := range interfaces {
			_, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"router", "remove", "port", prjPair.Live.Network.Router.Name, iface.PortId})
			if er.Error != nil {
				return fmt.Errorf("cannot remove router %s port %s: %s", prjPair.Live.Network.Router.Name, iface.PortId, er.Error)
			}
			sb.WriteString(fmt.Sprintf("removed router %s port %s", prjPair.Live.Network.Router.Name, iface.PortId))
		}
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"router", "delete", prjPair.Live.Network.Router.Name})
	if er.Error != nil {
		return er.Error
	}

	sb.WriteString(fmt.Sprintf("deleted router %s", prjPair.Live.Network.Router.Name))
	prjPair.Template.Network.Router.Id = ""
	prjPair.Live.Network.Router.Id = ""

	return nil
}

func CreateNetworking(prjPair *ProjectPair, logChan chan string) error {
	if err := CreateNetwork(prjPair, logChan); err != nil {
		return err
	}
	if err := CreateSubnet(prjPair, logChan); err != nil {
		return err
	}
	if err := CreateRouter(prjPair, logChan); err != nil {
		return err
	}
	return nil
}

func DeleteNetworking(prjPair *ProjectPair, logChan chan string) error {
	if err := DeleteRouter(prjPair, logChan); err != nil {
		return err
	}
	if err := DeleteSubnet(prjPair, logChan); err != nil {
		return err
	}
	if err := DeleteNetwork(prjPair, logChan); err != nil {
		return err
	}
	return nil
}
