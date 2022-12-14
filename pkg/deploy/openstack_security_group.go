package deploy

import (
	"fmt"
	"strings"
)

func CreateSecurityGroup(prjPair *ProjectPair, logChan chan string) error {
	if prjPair.Live.SecurityGroup.Name == "" {
		return fmt.Errorf("security group name cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("CreateSecurityGroup", sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                                  | Description                           | Project                          | Tags |
	// | c25f81ce-0db0-4b98-9c24-01543d0033bf | sample_deployment_name_security_group | sample_deployment_name_security_group | 56ac58a4903a458dbd4ea2241afc9566 | []   |
	foundGroupIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.SecurityGroup.Name)
	if prjPair.Live.SecurityGroup.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundGroupIdByName != "" {
			sb.WriteString(fmt.Sprintf("security group %s(%s) already there, updating project\n", prjPair.Live.SecurityGroup.Name, foundGroupIdByName))
			prjPair.Template.SecurityGroup.Id = foundGroupIdByName
			prjPair.Live.SecurityGroup.Id = foundGroupIdByName
		}
	} else {
		if foundGroupIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return fmt.Errorf("requested security group id %s not present, consider removing this id from the project file", prjPair.Live.SecurityGroup.Id)
		} else if prjPair.Live.SecurityGroup.Id != foundGroupIdByName {
			// It is already there, but has different id, complain
			return fmt.Errorf("requested security group id %s not matching existing security group id %s", prjPair.Live.SecurityGroup.Id, foundGroupIdByName)
		}
	}

	if prjPair.Live.SecurityGroup.Id == "" {
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "create", prjPair.Live.SecurityGroup.Name})
		if er.Error != nil {
			return er.Error
		}

		// | Field                   | Value                                |
		// | id                      | 79aa5ec5-41c2-4f15-b341-1659a783ea53 |
		// | name                    | sample_deployment_name_security_group        |
		newId := FindOpenstackFieldValue(rows, "id")
		if newId == "" {
			return fmt.Errorf("openstack returned empty security group id")
		}

		sb.WriteString(fmt.Sprintf("created security_group %s(%s)", prjPair.Live.SecurityGroup.Name, newId))
		prjPair.Template.SecurityGroup.Id = newId
		prjPair.Live.SecurityGroup.Id = newId
	}

	// Retrieve group rules

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "show", prjPair.Live.SecurityGroup.Name})
	if er.Error != nil {
		return er.Error
	}

	// | Field           | Value	|
	// | id              | c25f81ce-0db0-4b98-9c24-01543d0033bf	|
	// | name            | sample_deployment_name_security_group
	// | rules           | created_at='2022-12-02T02:52:21Z', direction='ingress', ethertype='IPv4', id='225416ee-c276-4994-b7ea-6357fe025aad', port_range_max='22', port_range_min='22', protocol='tcp', remote_ip_prefix='0.0.0.0/0', revision_number='1', updated_at='2022-12-02T02:52:21Z' |
	foundGroupRules := FindOpenstackFieldValue(rows, "rules")

	// Allow incoming ssh connections if needed

	if !(strings.Contains(foundGroupRules, "port_range_min='22'") &&
		strings.Contains(foundGroupRules, "port_range_max='22'") &&
		strings.Contains(foundGroupRules, "direction='ingress'") &&
		strings.Contains(foundGroupRules, "protocol='tcp'") &&
		strings.Contains(foundGroupRules, "remote_ip_prefix='0.0.0.0/0'")) {
		sb.WriteString(fmt.Sprintf("security group %s needs a new rule allowing ssh connections\n", prjPair.Live.SecurityGroup.Name))
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "rule", "create", "--proto", "tcp", "--dst-port", "22", prjPair.Live.SecurityGroup.Name})
		if er.Error != nil {
			return er.Error
		}
		sb.WriteString(fmt.Sprintf("updated security_group %s rules\n", prjPair.Live.SecurityGroup.Name))
	}

	return nil
}

func DeleteSecurityGroup(prjPair *ProjectPair, logChan chan string) error {
	if prjPair.Live.SecurityGroup.Name == "" {
		return fmt.Errorf("security group name cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("DeleteSecurityGroup", sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                                  | Description                           | Project                          | Tags |
	// | c25f81ce-0db0-4b98-9c24-01543d0033bf | sample_deployment_name_security_group | sample_deployment_name_security_group | 56ac58a4903a458dbd4ea2241afc9566 | []   |
	foundGroupIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.SecurityGroup.Name)
	if foundGroupIdByName == "" {
		sb.WriteString(fmt.Sprintf("security group %s not found, nothing to delete", prjPair.Live.SecurityGroup.Name))
		prjPair.Template.SecurityGroup.Id = ""
		prjPair.Live.SecurityGroup.Id = ""
		return nil
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "delete", prjPair.Live.SecurityGroup.Name})
	if er.Error != nil {
		return er.Error
	}

	sb.WriteString(fmt.Sprintf("deleted security group %s, updating project file", prjPair.Live.SecurityGroup.Name))
	prjPair.Template.SecurityGroup.Id = ""
	prjPair.Live.SecurityGroup.Id = ""

	return nil
}
