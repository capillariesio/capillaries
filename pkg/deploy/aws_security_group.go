package deploy

import (
	"fmt"
	"strings"
)

func createAwsSecurityGroup(prjPair *ProjectPair, sgNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createAwsSecurityGroup:"+sgNickname, isVerbose)
	liveGroupDef := prjPair.Live.SecurityGroups[sgNickname]
	if liveGroupDef.Name == "" {
		return lb.Complete(fmt.Errorf("security group name cannot be empty"))
	}

	foundGroupIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-security-groups",
		"--filter", "Name=tag:Name,Values=" + liveGroupDef.Name},
		".SecurityGroups[0].GroupId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if liveGroupDef.Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundGroupIdByName != "" {
			lb.Add(fmt.Sprintf("security group %s(%s) already there, updating project", liveGroupDef.Name, foundGroupIdByName))
			prjPair.SetSecurityGroupId(sgNickname, foundGroupIdByName)

		}
	} else {
		if foundGroupIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested security group id %s not present, consider removing this id from the project file", liveGroupDef.Id))
		} else if liveGroupDef.Id != foundGroupIdByName {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested security group id %s not matching existing security group id %s", liveGroupDef.Id, foundGroupIdByName))
		}
	}

	if liveGroupDef.Id == "" {
		newId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-security-group",
			"--vpc-id", prjPair.Live.Network.Id,
			"--group-name", liveGroupDef.Name,
			"--description", liveGroupDef.Name,
			"--tag-specification", fmt.Sprintf("ResourceType=security-group,Tags=[{Key=Name,Value=%s}]", liveGroupDef.Name)},
			".GroupId", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		lb.Add(fmt.Sprintf("created security_group %s(%s)", liveGroupDef.Name, newId))
		prjPair.SetSecurityGroupId(sgNickname, newId)
	}

	groupId := prjPair.Live.SecurityGroups[sgNickname].Id

	for ruleIdx, rule := range liveGroupDef.Rules {
		result, er := ExecLocalAndGetJsonBool(&prjPair.Live, "aws", []string{"ec2", "authorize-security-group-ingress",
			"--group-id", groupId,
			"--protocol", rule.Protocol,
			"--port", fmt.Sprintf("%d", rule.Port),
			"--cidr", rule.RemoteIp},
			".Return")
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
		if !result {
			if er.Error != nil {
				return lb.Complete(fmt.Errorf("rule creation returned false: %v", rule))
			}
		}

		// AWS does not assign ids to rules, so use the port
		prjPair.SetSecurityGroupRuleId(sgNickname, ruleIdx, fmt.Sprintf("%d", rule.Port))
	}

	return lb.Complete(nil)
}

func (*AwsDeployProvider) CreateSecurityGroups(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	sb := strings.Builder{}
	for sgNickname := range prjPair.Live.SecurityGroups {
		logMsg, err := createAwsSecurityGroup(prjPair, sgNickname, isVerbose)
		AddLogMsg(&sb, logMsg)
		if err != nil {
			return LogMsg(sb.String()), err
		}
	}
	return LogMsg(sb.String()), nil
}

func deleteAwsSecurityGroup(prjPair *ProjectPair, sgNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("deleteAwsSecurityGroup", isVerbose)
	liveGroupDef := prjPair.Live.SecurityGroups[sgNickname]
	if liveGroupDef.Name == "" {
		return lb.Complete(fmt.Errorf("security group name cannot be empty"))
	}

	foundGroupIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-security-groups",
		"--filter", "Name=tag:Name,Values=" + liveGroupDef.Name},
		".SecurityGroups[0].GroupId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if foundGroupIdByName == "" {
		lb.Add(fmt.Sprintf("security group %s not found, nothing to delete", liveGroupDef.Name))
		prjPair.CleanSecurityGroup(sgNickname)
		return lb.Complete(nil)
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-security-group",
		"--group-id", foundGroupIdByName}, prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted security group %s, updating project file", liveGroupDef.Name))
	prjPair.CleanSecurityGroup(sgNickname)

	return lb.Complete(nil)
}

func (*AwsDeployProvider) DeleteSecurityGroups(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	sb := strings.Builder{}
	for sgNickname := range prjPair.Live.SecurityGroups {
		logMsg, err := deleteAwsSecurityGroup(prjPair, sgNickname, isVerbose)
		AddLogMsg(&sb, logMsg)
		if err != nil {
			return LogMsg(sb.String()), err
		}
	}
	return LogMsg(sb.String()), nil
}
