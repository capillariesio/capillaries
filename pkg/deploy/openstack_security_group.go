package deploy

import (
	"fmt"
	"strings"
)

func CreateSecurityGroups(prjPair *ProjectPair, logChan chan string) error {
	for sgNickname, _ := range prjPair.Live.SecurityGroups {
		if err := CreateSecurityGroup(prjPair, logChan, sgNickname); err != nil {
			return err
		}
	}
	return nil
}

func CreateSecurityGroup(prjPair *ProjectPair, logChan chan string, sgNickname string) error {
	liveGroupDef := prjPair.Live.SecurityGroups[sgNickname]
	if liveGroupDef.Name == "" {
		return fmt.Errorf("security group name cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("CreateSecurityGroups", sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                                  | Description                           | Project                          | Tags |
	// | c25f81ce-0db0-4b98-9c24-01543d0033bf | sample_deployment_name_security_group | sample_deployment_name_security_group | 56ac58a4903a458dbd4ea2241afc9566 | []   |
	foundGroupIdByName := FindOpenstackColumnValue(rows, "ID", "Name", liveGroupDef.Name)
	if liveGroupDef.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundGroupIdByName != "" {
			sb.WriteString(fmt.Sprintf("security group %s(%s) already there, updating project\n", liveGroupDef.Name, foundGroupIdByName))
			prjPair.SetSecurityGroupId(sgNickname, foundGroupIdByName)

		}
	} else {
		if foundGroupIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return fmt.Errorf("requested security group id %s not present, consider removing this id from the project file", liveGroupDef.Id)
		} else if liveGroupDef.Id != foundGroupIdByName {
			// It is already there, but has different id, complain
			return fmt.Errorf("requested security group id %s not matching existing security group id %s", liveGroupDef.Id, foundGroupIdByName)
		}
	}

	if liveGroupDef.Id == "" {
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "create", liveGroupDef.Name})
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

		sb.WriteString(fmt.Sprintf("created security_group %s(%s)", liveGroupDef.Name, newId))
		prjPair.SetSecurityGroupId(sgNickname, newId)
	}

	// Retrieve group rules

	// rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "show", prjPair.Live.SecurityGroup.Name})
	// if er.Error != nil {
	// 	return er.Error
	// }

	// // | Field           | Value	|
	// // | id              | c25f81ce-0db0-4b98-9c24-01543d0033bf	|
	// // | name            | sample_deployment_name_security_group
	// // | rules           | created_at='2022-12-02T02:52:21Z', direction='ingress', ethertype='IPv4', id='225416ee-c276-4994-b7ea-6357fe025aad', port_range_max='22', port_range_min='22', protocol='tcp', remote_ip_prefix='0.0.0.0/0', revision_number='1', updated_at='2022-12-02T02:52:21Z' |
	// foundGroupRules := FindOpenstackFieldValue(rows, "rules")

	// // Allow incoming ssh connections if needed

	// if !(strings.Contains(foundGroupRules, "port_range_min='22'") &&
	// 	strings.Contains(foundGroupRules, "port_range_max='22'") &&
	// 	strings.Contains(foundGroupRules, "direction='ingress'") &&
	// 	strings.Contains(foundGroupRules, "protocol='tcp'") &&
	// 	strings.Contains(foundGroupRules, "remote_ip_prefix='0.0.0.0/0'")) {
	// 	sb.WriteString(fmt.Sprintf("security group %s needs a new rule allowing ssh connections\n", prjPair.Live.SecurityGroup.Name))
	// 	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "rule", "create", "--proto", "tcp", "--dst-port", "22", prjPair.Live.SecurityGroup.Name})
	// 	if er.Error != nil {
	// 		return er.Error
	// 	}
	// 	sb.WriteString(fmt.Sprintf("updated security_group %s rules\n", prjPair.Live.SecurityGroup.Name))
	// }

	// | ID                                   | IP Protocol | Ethertype | IP Range    | Port Range | Direction | Remote Security Group | Remote Address Group |
	// | 434a77d8-ad42-48d5-8d8c-d2d2cc41efdd | tcp         | IPv4      | 10.5.0.0/24 | 9090:9090  | ingress   | None                  | None                 |
	// | 86159116-4acf-4d78-9901-4602bc455dbd | None        | IPv4      | 0.0.0.0/0   |            | egress    | None                  | None                 |
	// | b3158642-996c-40f3-9729-121b47984a8c | None        | IPv6      | ::/0        |            | egress    | None                  | None                 |
	// | c58f1583-d4f2-48f1-bc60-1a28b768316c | tcp         | IPv4      | 0.0.0.0/0   | 22:22      | ingress   | None                  | None                 |
	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "rule", "list", liveGroupDef.Name})
	if er.Error != nil {
		return er.Error
	}

	for ruleIdx, rule := range liveGroupDef.Rules {
		// Search by port
		portRange := fmt.Sprintf("%d:%d", rule.Port, rule.Port)
		foundProtocol := FindOpenstackColumnValue(rows, "IP Protocol", "Port Range", portRange)
		foundEthertype := FindOpenstackColumnValue(rows, "Ethertype", "Port Range", portRange)
		foundRemoteIp := FindOpenstackColumnValue(rows, "IP Range", "Port Range", portRange)
		foundDirection := FindOpenstackColumnValue(rows, "Direction", "Port Range", portRange)
		foundId := FindOpenstackColumnValue(rows, "ID", "Port Range", portRange)
		if rule.Protocol == foundProtocol &&
			rule.Ethertype == foundEthertype &&
			rule.RemoteIp == foundRemoteIp &&
			rule.Direction == foundDirection {
			prjPair.SetSecurityGroupRuleId(sgNickname, ruleIdx, foundId)
		} else {
			sb.WriteString(fmt.Sprintf("security group %s needs a new rule for port %d, adding...\n", liveGroupDef.Name, rule.Port))
			rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "rule", "create", "--ethertype", rule.Ethertype, "--proto", rule.Protocol, "--remote-ip", rule.RemoteIp, "--dst-port", fmt.Sprintf("%d", rule.Port), fmt.Sprintf("--%s", rule.Direction), liveGroupDef.Name})
			if er.Error != nil {
				return er.Error
			}
			newId := FindOpenstackFieldValue(rows, "id")
			prjPair.SetSecurityGroupRuleId(sgNickname, ruleIdx, newId)
		}
	}

	return nil
}

func DeleteSecurityGroups(prjPair *ProjectPair, logChan chan string) error {
	for sgNickname, _ := range prjPair.Live.SecurityGroups {
		if err := DeleteSecurityGroup(prjPair, logChan, sgNickname); err != nil {
			return err
		}
	}
	return nil
}

func DeleteSecurityGroup(prjPair *ProjectPair, logChan chan string, sgNickname string) error {
	liveGroupDef := prjPair.Live.SecurityGroups[sgNickname]
	if liveGroupDef.Name == "" {
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
	foundGroupIdByName := FindOpenstackColumnValue(rows, "ID", "Name", liveGroupDef.Name)
	if foundGroupIdByName == "" {
		sb.WriteString(fmt.Sprintf("security group %s not found, nothing to delete", liveGroupDef.Name))
		prjPair.CleanSecurityGroup(sgNickname)
		return nil
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"security", "group", "delete", liveGroupDef.Name})
	if er.Error != nil {
		return er.Error
	}

	sb.WriteString(fmt.Sprintf("deleted security group %s, updating project file", liveGroupDef.Name))
	prjPair.CleanSecurityGroup(sgNickname)

	return nil
}
