package deploy

import (
	"fmt"
	"strings"
)

func (*OpenstackDeployProvider) GetFlavorIds(prjPair *ProjectPair, flavorMap map[string]string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("openstack.GetFlavors", isVerbose)

	rows, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"flavor", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	for flavorName := range flavorMap {
		foundFlavorIdByName := findOpenstackColumnValue(rows, "ID", "Name", flavorName)
		if foundFlavorIdByName == "" {
			return lb.Complete(fmt.Errorf("cannot find flavor %s", flavorName))
		}
		flavorMap[flavorName] = foundFlavorIdByName
	}

	return lb.Complete(nil)
}

func (*OpenstackDeployProvider) GetImageIds(prjPair *ProjectPair, imageMap map[string]string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("openstack.GetImages", isVerbose)

	rows, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"image", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	for name := range imageMap {
		foundIdByName := findOpenstackColumnValue(rows, "ID", "Name", name)
		if foundIdByName == "" {
			return lb.Complete(fmt.Errorf("cannot find image %s", name))
		}
		imageMap[name] = foundIdByName
	}

	return lb.Complete(nil)
}

func (*OpenstackDeployProvider) GetKeypairs(prjPair *ProjectPair, keypairMap map[string]struct{}, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("openstack.GetKeypairs", isVerbose)

	rows, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"keypair", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	for keypairName := range keypairMap {
		foundName := findOpenstackColumnValue(rows, "Fingerprint", "Name", keypairName)
		if foundName == "" {
			return lb.Complete(fmt.Errorf("cannot find keypair %s, you have to create it before running this command", keypairName))
		}
	}

	return lb.Complete(nil)
}

func (*OpenstackDeployProvider) CreateInstanceAndWaitForCompletion(prjPair *ProjectPair, iNickname string, flavorId string, imageId string, availabilityZone string, isVerbose bool) (LogMsg, error) {
	sb := strings.Builder{}

	logMsg, err := createOpenstackInstance(prjPair, iNickname, flavorId, imageId, availabilityZone, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	logMsg, err = waitForOpenstackEntityToBeCreated(&prjPair.Live, "server", prjPair.Live.Instances[iNickname].HostName, prjPair.Live.Instances[iNickname].Id, prjPair.Live.Timeouts.OpenstackInstanceCreation, isVerbose)
	AddLogMsg(&sb, logMsg)
	if err != nil {
		return LogMsg(sb.String()), err
	}

	if prjPair.Live.Instances[iNickname].ExternalIpAddress != "" {
		logMsg, err = assignOpenstackFloatingIp(&prjPair.Live, prjPair.Live.Instances[iNickname].Id, prjPair.Live.Instances[iNickname].ExternalIpAddress, isVerbose)
		AddLogMsg(&sb, logMsg)
		if err != nil {
			return LogMsg(sb.String()), err
		}
	}

	return LogMsg(sb.String()), nil
}

func assignOpenstackFloatingIp(prj *Project, instanceId string, floatingIp string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("assignOpenstackFloatingIp:"+floatingIp, isVerbose)
	_, er := execLocalAndParseOpenstackOutput(prj, "openstack", []string{"server", "add", "floating", "ip", instanceId, floatingIp})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	lb.Add(fmt.Sprintf("floating ip %s was assigned to instance %s", floatingIp, instanceId))
	return lb.Complete(nil)
}

func createOpenstackInstance(prjPair *ProjectPair, iNickname string, flavorId string, imageId string, availabilityZone string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("createOpenstackInstance:"+prjPair.Live.Instances[iNickname].HostName, isVerbose)
	if prjPair.Live.Instances[iNickname].HostName == "" ||
		prjPair.Live.Instances[iNickname].IpAddress == "" {
		return lb.Complete(fmt.Errorf("instance hostname(%s), ip address(%s) cannot be empty", prjPair.Live.Instances[iNickname].HostName, prjPair.Live.Instances[iNickname].IpAddress))
	}

	// If floating ip is requested and it's already assigned, fail
	if prjPair.Live.Instances[iNickname].ExternalIpAddress != "" {
		rows, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"floating", "ip", "list"})
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		// | ID         | Floating IP Address | Fixed IP Address | Port | Floating Network | Project                          |
		// | 8e6b5d7... | 205.234.86.102      | None             | None | e098d02f-... | 56ac58a4... |
		foundFoatingIpPort := findOpenstackColumnValue(rows, "Port", "Floating IP Address", prjPair.Live.Instances[iNickname].ExternalIpAddress)
		if foundFoatingIpPort == "" {
			return lb.Complete(fmt.Errorf("cannot create instance %s, floating ip %s is not available, make sure it was reserved", prjPair.Live.Instances[iNickname].HostName, prjPair.Live.Instances[iNickname].ExternalIpAddress))
		}
		if foundFoatingIpPort != "None" {
			return lb.Complete(fmt.Errorf("cannot create instance %s, floating ip %s is already assigned, see port %s", prjPair.Live.Instances[iNickname].HostName, prjPair.Live.Instances[iNickname].ExternalIpAddress, foundFoatingIpPort))
		}
	}

	rows, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"server", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	// | ID                                   | Name                       |
	// | 8aa8a5e8-2aad-4006-8911-af7de31b08fb | sample_deployment_name_bastion |
	foundInstanceIdByName := findOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Instances[iNickname].HostName)
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

	rows, er = execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{
		"server", "create", "--flavor", flavorId, "--image", imageId,
		"--key-name", prjPair.Live.Instances[iNickname].RootKeyName,
		"--availability-zone", availabilityZone,
		"--security-group", prjPair.Live.SecurityGroups[prjPair.Live.Instances[iNickname].SecurityGroupNickname].Name,
		"--nic", fmt.Sprintf("net-id=%s,v4-fixed-ip=%s", prjPair.Live.Network.Id, prjPair.Live.Instances[iNickname].IpAddress),
		prjPair.Live.Instances[iNickname].HostName})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | Field                       | Value                                               |
	// | id                          | 4203f08f-089d-4e8e-ab41-cd3ce227d9d2                |
	// | status                      | BUILD                                               |
	newId := findOpenstackFieldValue(rows, "id")
	if newId == "" {
		return lb.Complete(fmt.Errorf("openstack returned empty instance id"))
	}

	lb.Add(fmt.Sprintf("creating instance %s(%s)...", prjPair.Live.Instances[iNickname].HostName, newId))
	prjPair.SetInstanceId(iNickname, newId)

	return lb.Complete(nil)
}

func (*OpenstackDeployProvider) DeleteInstance(prjPair *ProjectPair, iNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("openstack.DeleteInstance:"+prjPair.Live.Instances[iNickname].HostName, isVerbose)
	if prjPair.Live.Instances[iNickname].HostName == "" {
		return lb.Complete(fmt.Errorf("instance %s hostname cannot be empty", iNickname))
	}

	_, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"server", "delete", prjPair.Live.Instances[iNickname].HostName})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	prjPair.CleanInstance(iNickname)

	return lb.Complete(nil)
}
