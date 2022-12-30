package deploy

import (
	"fmt"
	"time"
)

func GetFlavorIds(prjPair *ProjectPair, flavorMap map[string]string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("GetFlavors", isVerbose)

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"flavor", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	for flavorName, _ := range flavorMap {
		foundFlavorIdByName := FindOpenstackColumnValue(rows, "ID", "Name", flavorName)
		if foundFlavorIdByName == "" {
			return lb.Complete(fmt.Errorf("cannot find flavor %s", flavorName))
		}
		flavorMap[flavorName] = foundFlavorIdByName
	}

	return lb.Complete(nil)
}

func GetImageIds(prjPair *ProjectPair, imageMap map[string]string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("GetImages", isVerbose)

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"image", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	for name, _ := range imageMap {
		foundIdByName := FindOpenstackColumnValue(rows, "ID", "Name", name)
		if foundIdByName == "" {
			return lb.Complete(fmt.Errorf("cannot find image %s", name))
		}
		imageMap[name] = foundIdByName
	}

	return lb.Complete(nil)
}

func CreateInstance(prjPair *ProjectPair, iNickname string, flavorId string, imageId string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("CreateInstance:"+prjPair.Live.Instances[iNickname].HostName, isVerbose)
	if prjPair.Live.Instances[iNickname].HostName == "" ||
		prjPair.Live.Instances[iNickname].IpAddress == "" {
		return lb.Complete(fmt.Errorf("instance hostname, ip address cannot be empty"))
	}

	// If floating ip is requested and it's already assigned, fail
	if prjPair.Live.Instances[iNickname].FloatingIpAddress != "" {
		rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"floating", "ip", "list"})
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		// | ID         | Floating IP Address | Fixed IP Address | Port | Floating Network | Project                          |
		// | 8e6b5d7... | 205.234.86.102      | None             | None | e098d02f-... | 56ac58a4... |
		foundFoatingIpPort := FindOpenstackColumnValue(rows, "Port", "Floating IP Address", prjPair.Live.Instances[iNickname].FloatingIpAddress)
		if foundFoatingIpPort == "" {
			return lb.Complete(fmt.Errorf("cannot create instance %s, floating ip %s is not available, make sure it was reserved", prjPair.Live.Instances[iNickname].HostName, prjPair.Live.Instances[iNickname].FloatingIpAddress))
		}
		if foundFoatingIpPort != "None" {
			return lb.Complete(fmt.Errorf("cannot create instance %s, floating ip %s is already assigned, see port %s", prjPair.Live.Instances[iNickname].HostName, prjPair.Live.Instances[iNickname].FloatingIpAddress, foundFoatingIpPort))
		}
	}

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"server", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}
	// | ID                                   | Name                       |
	// | 8aa8a5e8-2aad-4006-8911-af7de31b08fb | sample_deployment_name_bastion |
	foundInstanceIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Instances[iNickname].HostName)
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

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{
		"server", "create", "--flavor", flavorId, "--image", imageId,
		"--key-name", prjPair.Live.RootKeyName,
		"--availability-zone", prjPair.Live.AvailabilityZone,
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
	newId := FindOpenstackFieldValue(rows, "id")
	if newId == "" {
		return lb.Complete(fmt.Errorf("openstack returned empty instance id"))
	}

	lb.Add(fmt.Sprintf("creating instance %s(%s)...", prjPair.Live.Instances[iNickname].HostName, newId))
	prjPair.SetInstanceId(iNickname, newId)

	var status string
	startInstanceWaitTs := time.Now()
	for time.Since(startInstanceWaitTs).Seconds() < float64(prjPair.Live.Timeouts.OpenstackInstanceCreation) {
		status = FindOpenstackFieldValue(rows, "status")
		if status == "" {
			return lb.Complete(fmt.Errorf("openstack returned empty instance status"))
		}
		if status != "BUILD" {
			break
		}
		lb.Add(fmt.Sprintf("instance %s(%s) is being created", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName))
		time.Sleep(5 * time.Second)
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"server", "show", prjPair.Live.Instances[iNickname].HostName})
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
	}
	if status == "BUILD" {
		return lb.Complete(fmt.Errorf("building instance %s(%s) took too long", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName))
	}
	if status != "ACTIVE" {
		return lb.Complete(fmt.Errorf("instance %s(%s) was built, but the status is %s", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName, status))
	}

	if prjPair.Live.Instances[iNickname].FloatingIpAddress != "" {
		_, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"server", "add", "floating", "ip", newId, prjPair.Live.Instances[iNickname].FloatingIpAddress})
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
		lb.Add(fmt.Sprintf("floating ip %s was assigned to instance %s(%s)", prjPair.Live.Instances[iNickname].FloatingIpAddress, prjPair.Live.Instances[iNickname].HostName, newId))
	}

	return lb.Complete(nil)
}

func DeleteInstance(prjPair *ProjectPair, iNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("DeleteInstance:"+prjPair.Live.Instances[iNickname].HostName, isVerbose)
	if prjPair.Live.Instances[iNickname].HostName == "" {
		return lb.Complete(fmt.Errorf("instance hostname cannot be empty"))
	}

	_, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"server", "delete", prjPair.Live.Instances[iNickname].HostName})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	prjPair.CleanInstance(iNickname)

	return lb.Complete(nil)
}
