package deploy

import (
	"fmt"
	"strings"
	"time"
)

func (*AwsDeployProvider) GetFlavorIds(prjPair *ProjectPair, flavorMap map[string]string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.GetFlavorIds", isVerbose)

	for flavorName := range flavorMap {
		foundInstanceType, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-instance-types",
			"--filter", "Name=instance-type,Values=" + flavorName},
			".InstanceTypes[0].InstanceType", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(fmt.Errorf("cannot find flavor %s:%s", flavorName, er.Error.Error()))
		}
		flavorMap[flavorName] = foundInstanceType
	}

	return lb.Complete(nil)
}

func (*AwsDeployProvider) GetImageIds(prjPair *ProjectPair, imageMap map[string]string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.GetImageIds", isVerbose)

	for imageId := range imageMap {
		foundImageId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-images",
			"--filter", "Name=image-id,Values=" + imageId},
			".Images[0].ImageId", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(fmt.Errorf("cannot find image %s:%s", imageId, er.Error.Error()))
		}
		imageMap[imageId] = foundImageId
	}
	return lb.Complete(nil)
}

func (*AwsDeployProvider) GetKeypairs(prjPair *ProjectPair, keypairMap map[string]struct{}, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.GetKeypairs", isVerbose)

	for keypairName := range keypairMap {
		_, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-key-pairs",
			"--filter", "Name=key-name,Values=" + keypairName},
			".KeyPairs[0].KeyPairId", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(fmt.Errorf("cannot find keypair %s:%s", keypairName, er.Error.Error()))
		}
	}

	return lb.Complete(nil)
}

func createAwsInstance(prjPair *ProjectPair, iNickname string, flavorId string, imageId string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createAwsInstance:"+prjPair.Live.Instances[iNickname].HostName, isVerbose)
	if prjPair.Live.Instances[iNickname].HostName == "" ||
		prjPair.Live.Instances[iNickname].IpAddress == "" {
		return lb.Complete(fmt.Errorf("instance hostname(%s), ip address(%s) cannot be empty", prjPair.Live.Instances[iNickname].HostName, prjPair.Live.Instances[iNickname].IpAddress))
	}

	// If floating ip is requested and it's already assigned, fail
	if prjPair.Live.Instances[iNickname].ExternalIpAddress != "" {

		attachedToInstanceId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-addresses",
			"--filter", "Name=public-ip,Values=" + prjPair.Live.Instances[iNickname].ExternalIpAddress},
			".Addresses[0].InstanceId", true)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		if attachedToInstanceId != "" {
			return lb.Complete(fmt.Errorf("cannot create instance %s, floating ip %s is already assigned, see instance %s", prjPair.Live.Instances[iNickname].HostName, prjPair.Live.Instances[iNickname].ExternalIpAddress, attachedToInstanceId))
		}
	}

	foundInstanceIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-instances",
		"--filter", "Name=tag:name,Values=" + prjPair.Live.Instances[iNickname].HostName},
		".Instances[0].InstanceId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if prjPair.Live.Instances[iNickname].Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundInstanceIdByName != "" {
			lb.Add(fmt.Sprintf("instance %s(%s) already there, updating project", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName))
			prjPair.SetInstanceId(iNickname, foundInstanceIdByName)
		}
	} else {
		if foundInstanceIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested instance id %s not present, consider removing this id from the project file", prjPair.Live.Instances[iNickname].Id))
		} else if prjPair.Live.Instances[iNickname].Id != foundInstanceIdByName {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested instance id %s not matching existing instance id %s", prjPair.Live.Instances[iNickname].Id, foundInstanceIdByName))
		}
	}

	if prjPair.Live.Instances[iNickname].Id != "" {
		lb.Add(fmt.Sprintf("instance %s(%s) already there, no need to create", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName))
		return lb.Complete(nil)
	}

	// NOTE: AWS doesn't allow to specify hostname on creation
	newId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "run-instances",
		"--image-id", imageId,
		"--count", "1",
		"--instance-type", flavorId,
		"--key-name", prjPair.Live.Instances[iNickname].RootKeyName,
		"--security-group-ids", prjPair.Live.SecurityGroups[prjPair.Live.Instances[iNickname].SecurityGroupNickname].Id,
		"--subnet-id", prjPair.Live.Network.Subnet.Id,
		"--private-ip-address", prjPair.Live.Instances[iNickname].IpAddress,
		"--tag-specification", fmt.Sprintf("ResourceType=instance,Tags=[{Key=name,Value=%s}]", prjPair.Live.Instances[iNickname].HostName)},
		".Instances[0].InstanceId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if newId == "" {
		return lb.Complete(fmt.Errorf("aws returned empty instance id"))
	}

	lb.Add(fmt.Sprintf("creating instance %s(%s)...", prjPair.Live.Instances[iNickname].HostName, newId))
	prjPair.SetInstanceId(iNickname, newId)

	return lb.Complete(nil)
}

func waitForAwsInstanceToBeCreated(prj *Project, instanceId string, timeoutSeconds int, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("waitForAwsInstanceToBeCreated:"+instanceId, isVerbose)
	startWaitTs := time.Now()

	for {
		status, er := ExecLocalAndGetJsonString(prj, "aws", []string{"ec2", "describe-instances",
			"--instance-ids", instanceId},
			".Reservations[0].Instances[0].State.Name", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
		if status == "" {
			return lb.Complete(fmt.Errorf("aws returned empty status for %s", instanceId))
		}
		if status == "running" {
			break
		}
		if status != "pending" {
			return lb.Complete(fmt.Errorf("%s was built, but the status is unknown: %s", instanceId, status))
		}
		if time.Since(startWaitTs).Seconds() > float64(prj.Timeouts.OpenstackInstanceCreation) {
			return lb.Complete(fmt.Errorf("giving up after waiting for %s to be created", instanceId))
		}
		time.Sleep(10 * time.Second)
	}
	return lb.Complete(nil)
}

func assignAwsFloatingIp(prj *Project, instanceId string, floatingIp string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("assignAwsFloatingIp:"+floatingIp, isVerbose)
	associationId, er := ExecLocalAndGetJsonString(prj, "aws", []string{"ec2", "associate-address",
		"--instance-id", instanceId,
		"--public-ip", floatingIp},
		".AssociationId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("floating ip %s was assigned to instance %s, association id %s", floatingIp, instanceId, associationId))
	return lb.Complete(nil)
}

func (*AwsDeployProvider) CreateInstanceAndWaitForCompletion(prjPair *ProjectPair, iNickname string, flavorId string, imageId string, availabilityZone string, isVerbose bool) (LogMsg, error) {
	sb := strings.Builder{}

	logMsg, err := createAwsInstance(prjPair, iNickname, flavorId, imageId, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = waitForAwsInstanceToBeCreated(&prjPair.Live, prjPair.Live.Instances[iNickname].Id, prjPair.Live.Timeouts.OpenstackInstanceCreation, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	if prjPair.Live.Instances[iNickname].ExternalIpAddress != "" {
		logMsg, err = assignAwsFloatingIp(&prjPair.Live, prjPair.Live.Instances[iNickname].Id, prjPair.Live.Instances[iNickname].ExternalIpAddress, isVerbose)
		AddLogMsg(&sb, logMsg)
		if err != nil {
			return LogMsg(sb.String()), err
		}
	}

	return LogMsg(sb.String()), nil
}
func (*AwsDeployProvider) DeleteInstance(prjPair *ProjectPair, iNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.DeleteInstance", isVerbose)
	if prjPair.Live.Instances[iNickname].Id == "" {
		return lb.Complete(fmt.Errorf("instance %s id cannot be empty", iNickname))
	}

	_, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "terminate-instances",
		"--instance-ids", prjPair.Live.Instances[iNickname].Id},
		".TerminatingInstances[0].InstanceId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	prjPair.CleanInstance(iNickname)

	return lb.Complete(nil)
}
