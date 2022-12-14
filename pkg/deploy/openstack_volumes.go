package deploy

import (
	"fmt"
	"strings"
)

func CreateVolume(prjPair *ProjectPair, logChan chan string, volNickname string) error {
	if prjPair.Live.Volumes[volNickname].Name == "" ||
		prjPair.Live.Volumes[volNickname].MountPoint == "" ||
		prjPair.Live.Volumes[volNickname].Size == 0 {
		return fmt.Errorf("volume name, mount point, size cannot be empty")
	}

	sb := strings.Builder{}
	defer func() {
		logChan <- CmdChainExecToString("CreateVolume:"+prjPair.Live.Volumes[volNickname].Name, sb.String())
	}()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"volume", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                       | Status    | Size | Attached to |
	// | 8aa8a5e8-2aad-4006-8911-af7de31b08fb | sample_deployment_name_cfg | available |    1 |             |
	foundVolIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Volumes[volNickname].Name)
	if prjPair.Live.Volumes[volNickname].Id == "" {
		// If it was already created, save it for future use, but do not create
		if foundVolIdByName != "" {
			sb.WriteString(fmt.Sprintf("volume %s(%s) already there, updating project\n", prjPair.Live.Volumes[volNickname].Name, foundVolIdByName))
			prjPair.Template.Volumes[volNickname].Id = foundVolIdByName
			prjPair.Live.Volumes[volNickname].Id = foundVolIdByName
		}
	} else {
		if foundVolIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return fmt.Errorf("requested volume id %s not present, consider removing this id from the project file", prjPair.Live.Volumes[volNickname].Id)
		} else if prjPair.Live.Volumes[volNickname].Id != foundVolIdByName {
			// It is already there, but has different id, complain
			return fmt.Errorf("requested volume id %s not matching existing volume id %s", prjPair.Live.Volumes[volNickname].Id, foundVolIdByName)
		}
	}

	if prjPair.Live.Volumes[volNickname].Id != "" {
		sb.WriteString(fmt.Sprintf("volume %s(%s) already there, no need to create\n", prjPair.Live.Volumes[volNickname].Name, foundVolIdByName))
		return nil
	}

	rows, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"volume", "create", "--size", fmt.Sprintf("%d", prjPair.Live.Volumes[volNickname].Size), "--availability-zone", prjPair.Live.AvailabilityZone, prjPair.Live.Volumes[volNickname].Name})
	if er.Error != nil {
		return er.Error
	}

	// | id  | 8aa8a5e8-2aad-4006-8911-af7de31b08fb
	newId := FindOpenstackFieldValue(rows, "id")
	if newId == "" {
		return fmt.Errorf("openstack returned empty volume id")
	}

	sb.WriteString(fmt.Sprintf("created volume %s(%s)", prjPair.Live.SecurityGroup.Name, newId))
	prjPair.Template.Volumes[volNickname].Id = newId
	prjPair.Live.Volumes[volNickname].Id = newId

	return nil
}

func DeleteVolume(prjPair *ProjectPair, logChan chan string, volNickname string) error {
	if prjPair.Live.Volumes[volNickname].Name == "" {
		return fmt.Errorf("volume name, mount point, size cannot be empty")
	}

	sb := strings.Builder{}
	defer func() { logChan <- CmdChainExecToString("DeleteVolume:"+volNickname, sb.String()) }()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"volume", "list"})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Name                       | Status    | Size | Attached to |
	// | 8aa8a5e8-2aad-4006-8911-af7de31b08fb | sample_deployment_name_cfg | available |    1 |             |
	foundVolIdByName := FindOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Volumes[volNickname].Name)
	if foundVolIdByName == "" {
		sb.WriteString(fmt.Sprintf("volume %s not found, nothing to delete", prjPair.Live.Volumes[volNickname].Name))
		prjPair.Template.Volumes[volNickname].Id = ""
		prjPair.Live.Volumes[volNickname].Id = ""
		return nil
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"volume", "delete", prjPair.Live.Volumes[volNickname].Name})
	if er.Error != nil {
		return er.Error
	}

	sb.WriteString(fmt.Sprintf("deleted volume %s, updating project file", prjPair.Live.Volumes[volNickname].Name))
	prjPair.Template.Volumes[volNickname].Id = ""
	prjPair.Live.Volumes[volNickname].Id = ""

	return nil
}
