package deploy

import (
	"fmt"
	"strings"
	"time"
)

func GetFlavorIds(prjPair *ProjectPair, logChan chan string, flavorMap map[string]string) error {
	sb := strings.Builder{}
	defer func() {
		logChan <- CmdChainExecToString("GetFlavors:", sb.String())
	}()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"flavor", "list"})
	if er.Error != nil {
		return er.Error
	}

	for flavorName, _ := range flavorMap {
		foundFlavorIdByName := FindOpenstackColumnValue(rows, "ID", "Name", flavorName)
		if foundFlavorIdByName == "" {
			return fmt.Errorf("cannot find flavor %s", flavorName)
		}
		flavorMap[flavorName] = foundFlavorIdByName
	}

	return nil
}

func GetImageIds(prjPair *ProjectPair, logChan chan string, imageMap map[string]string) error {
	sb := strings.Builder{}
	defer func() {
		logChan <- CmdChainExecToString("GetImages:", sb.String())
	}()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"image", "list"})
	if er.Error != nil {
		return er.Error
	}

	for name, _ := range imageMap {
		foundIdByName := FindOpenstackColumnValue(rows, "ID", "Name", name)
		if foundIdByName == "" {
			return fmt.Errorf("cannot find image %s", name)
		}
		imageMap[name] = foundIdByName
	}

	return nil
}
func CreateInstance(prjPair *ProjectPair, logChan chan string, iNickname string, flavorId string, imageId string) error {
	if prjPair.Live.Instances[iNickname].HostName == "" ||
		prjPair.Live.Instances[iNickname].IpAddress == "" {
		return fmt.Errorf("instance hostname, ip address cannot be empty")
	}

	sb := strings.Builder{}
	defer func() {
		logChan <- CmdChainExecToString("CreateInstance:"+prjPair.Live.Instances[iNickname].HostName, sb.String())
	}()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"server", "list"})
	if er.Error != nil {
		return er.Error
	}
	// | ID                                   | Name                       |
	// | 8aa8a5e8-2aad-4006-8911-af7de31b08fb | sample_deployment_name_bastion |
	foundInstanceIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Instances[iNickname].HostName)
	if prjPair.Live.Instances[iNickname].Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundInstanceIdByName != "" {
			sb.WriteString(fmt.Sprintf("instance %s(%s) already there, updating project\n", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName))
			prjPair.Template.Instances[iNickname].Id = foundInstanceIdByName
			prjPair.Live.Instances[iNickname].Id = foundInstanceIdByName
		}
	} else {
		if foundInstanceIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return fmt.Errorf("requested instance id %s not present, consider removing this id from the project file", prjPair.Live.Instances[iNickname].Id)
		} else if prjPair.Live.Instances[iNickname].Id != foundInstanceIdByName {
			// It is already there, but has different id, complain
			return fmt.Errorf("requested instance id %s not matching existing instance id %s", prjPair.Live.Instances[iNickname].Id, foundInstanceIdByName)
		}
	}

	if prjPair.Live.Instances[iNickname].Id != "" {
		sb.WriteString(fmt.Sprintf("instance %s(%s) already there, no need to create\n", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName))
		return nil
	}

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{
		"server", "create", "--flavor", flavorId, "--image", imageId,
		"--key-name", prjPair.Live.RootKeyName,
		"--availability-zone", prjPair.Live.AvailabilityZone,
		"--security-group", prjPair.Live.SecurityGroup.Name,
		"--nic", fmt.Sprintf("net-id=%s,v4-fixed-ip=%s", prjPair.Live.Network.Id, prjPair.Live.Instances[iNickname].IpAddress),
		prjPair.Live.Instances[iNickname].HostName})
	if er.Error != nil {
		return er.Error
	}

	// | Field                       | Value                                               |
	// | id                          | 4203f08f-089d-4e8e-ab41-cd3ce227d9d2                |
	// | status                      | BUILD                                               |
	newId := FindOpenstackFieldValue(rows, "id")
	if newId == "" {
		return fmt.Errorf("openstack returned empty instance id")
	}

	sb.WriteString(fmt.Sprintf("creating instance %s(%s)...\n", prjPair.Live.Instances[iNickname].HostName, newId))
	prjPair.Template.Instances[iNickname].Id = newId
	prjPair.Live.Instances[iNickname].Id = newId

	var status string
	for i := 0; i <= 60; i = i + 1 {
		status = FindOpenstackFieldValue(rows, "status")
		if status == "" {
			return fmt.Errorf("openstack returned empty instance status")
		}
		if status != "BUILD" {
			break
		}
		sb.WriteString(fmt.Sprintf("instance %s(%s) is being created\n", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName))
		time.Sleep(1 * time.Second)
		rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{
			"server", "show", prjPair.Live.Instances[iNickname].HostName})
		if er.Error != nil {
			return er.Error
		}
	}
	if status == "BUILD" {
		return fmt.Errorf("building instance %s(%s) took too long", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName)
	}
	if status != "ACTIVE" {
		return fmt.Errorf("instance %s(%s) was built, but the status is %s", prjPair.Live.Instances[iNickname].HostName, foundInstanceIdByName, status)
	}

	return nil
}