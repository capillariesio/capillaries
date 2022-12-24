package deploy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const InitVolumeAttachmentFunc string = `
init_volume_attachment()
{ 
  local deviceName=$1
  local volumeMountPath=$2
  local permissions=$3

  # Check if file system is already there
  local deviceBlockId=$(blkid -s UUID -o value $deviceName)
  if [ "$deviceBlockId" = "" ]; then
    # Make file system
    sudo mkfs.ext4 $deviceName
    if [ "$?" -ne "0" ]; then
      echo Cannot make file system
      return $?
    fi
  fi

  deviceBlockId=$(sudo blkid -s UUID -o value $deviceName)

  local alreadyMounted=$(cat /etc/fstab | grep $volumeMountPath)

  if [ "$alreadyMounted" = "" ]; then
    # Create mount point
    sudo mkdir -p $volumeMountPath
    if [ "$?" -ne "0" ]; then
      echo Cannot create mount dir
      return $?
    fi

    # Set permissions
    sudo chmod $permissions $volumeMountPath
    if [ "$?" -ne "0" ]; then
        echo Cannot change permissions to $permissions
        return $?
    fi

    # Adds a line to /etc/fstab
    echo "UUID=$deviceBlockId   $volumeMountPath   ext4   defaults   0   2 " | sudo tee -a /etc/fstab
  fi

  # Report UUID
  echo $deviceBlockId
}
`

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
			prjPair.SetVolumeId(volNickname, foundVolIdByName)
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

	sb.WriteString(fmt.Sprintf("created volume %s: %s(%s)", volNickname, prjPair.Live.Volumes[volNickname].Name, newId))
	prjPair.SetVolumeId(volNickname, newId)

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
		prjPair.SetVolumeId(volNickname, "")
		return nil
	}

	_, er = ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"volume", "delete", prjPair.Live.Volumes[volNickname].Name})
	if er.Error != nil {
		return er.Error
	}

	sb.WriteString(fmt.Sprintf("deleted volume %s, updating project file", prjPair.Live.Volumes[volNickname].Name))
	prjPair.SetVolumeId(volNickname, "")

	return nil
}

func ShowVolumeAttachment(prj *Project, sb *strings.Builder, volNickname string, iNickname string) (string, error) {
	rows, er := ExecLocalAndParseOpenstackOutput(prj, sb, "openstack", []string{"volume", "show", prj.Volumes[volNickname].Id})
	if er.Error != nil {
		return "", er.Error
	}
	// | Field        | Value
	// | attachments  | [{'server_id': '...', 'attachment_id': '...', ...}] |
	foundAttachmentsJson := FindOpenstackFieldValue(rows, "attachments")
	if foundAttachmentsJson == "" {
		return "", fmt.Errorf("cannot find attachments for volume %s, expected to see the newly created one", prj.Volumes[volNickname].Id)
	}
	var foundAttachmentsArray []map[string]string
	if err := json.Unmarshal([]byte(strings.ReplaceAll(strings.ReplaceAll(foundAttachmentsJson, "'", "\""), "None", "null")), &foundAttachmentsArray); err != nil {
		return "", err
	}

	// Walk through all reported attachments and find the one that belongs to this instance
	for _, attachment := range foundAttachmentsArray {
		serverId, ok1 := attachment["server_id"]
		attachmentId, ok2 := attachment["attachment_id"]
		if ok1 && ok2 && serverId == prj.Instances[iNickname].Id {
			return attachmentId, nil
		}
	}

	return "", nil
}

func AttachVolume(prjPair *ProjectPair, logChan chan string, iNickname string, volNickname string) error {
	if prjPair.Live.Volumes[volNickname].Id == "" || prjPair.Live.Instances[iNickname].Id == "" {
		return fmt.Errorf("cannot attach volume %s(%s) to %s(%s), no empty ids allowed", volNickname, prjPair.Live.Volumes[volNickname].Id, iNickname, prjPair.Live.Instances[iNickname].Id)
	}

	sb := strings.Builder{}
	defer func() {
		logChan <- CmdChainExecToString(fmt.Sprintf("AttachVolume: %s to %s", volNickname, iNickname), sb.String())
	}()

	rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"server", "volume", "list", prjPair.Live.Instances[iNickname].Id})
	if er.Error != nil {
		return er.Error
	}

	// | ID                                   | Device   | Server ID                            | Volume ID                            |
	// | 8b9b3491-f083-4485-8374-258372f3db35 | /dev/vdb | 216f9481-4c9d-4530-b865-51cedfa4b8e7 | 8b9b3491-f083-4485-8374-258372f3db35 |
	foundDevice := FindOpenstackColumnValue(rows, "Device", "Volume ID", prjPair.Live.Volumes[volNickname].Id)
	// Do not compare/complain, just overwrite: the number of attachment does not help catch unaccounted cloud resources anyways
	if prjPair.Live.Instances[iNickname].AttachedVolumes[volNickname].Device != "" {
		if foundDevice != "" {
			sb.WriteString(fmt.Sprintf("volume %s already attached to %s, device %s, updating project\n", volNickname, iNickname, foundDevice))
		} else {
			sb.WriteString(fmt.Sprintf("volume %s was not attached to %s, cleaning attachment info, updating project\n", volNickname, iNickname))
		}
		prjPair.SetAttachedVolumeDevice(iNickname, volNickname, foundDevice)
	} else {
		if foundDevice != "" && foundDevice != prjPair.Live.Instances[iNickname].AttachedVolumes[volNickname].Device {
			sb.WriteString(fmt.Sprintf("volume %s already to %s, but with a different device(%s->%s), updating project\n", volNickname, iNickname, prjPair.Live.Instances[iNickname].AttachedVolumes[volNickname].Device, foundDevice))
			prjPair.SetAttachedVolumeDevice(iNickname, volNickname, foundDevice)
		}
	}

	if prjPair.Live.Instances[iNickname].AttachedVolumes[volNickname].Device == "" {
		rows, er := ExecLocalAndParseOpenstackOutput(&prjPair.Live, &sb, "openstack", []string{"server", "add", "volume", prjPair.Live.Instances[iNickname].Id, prjPair.Live.Volumes[volNickname].Id})
		if er.Error != nil {
			return er.Error
		}

		// | Field     | Value                                |
		// | ID        | 8aa8a5e8-2aad-4006-8911-af7de31b08fb |
		// | Server ID | 1466c448-86c8-404b-b3f1-d95051276f38 |
		// | Volume ID | 8aa8a5e8-2aad-4006-8911-af7de31b08fb |
		// | Device    | /dev/vdb                             |
		newId := FindOpenstackFieldValue(rows, "ID") // This is not the attachment id, it's a dupr of volume id
		device := FindOpenstackFieldValue(rows, "Device")
		if newId == "" || device == "" {
			return fmt.Errorf("got empty id/device (%s/%s) when attaching volume %s to %s", newId, device, volNickname, iNickname)
		}
		sb.WriteString(fmt.Sprintf("attached volume %s to %s, attachment id %s, updating project\n", volNickname, iNickname, newId))
		prjPair.SetAttachedVolumeDevice(iNickname, volNickname, device)
	}

	// At this point, we are sure we have good device
	// We may need to wait a few sec until the device is ready

	startAttachWaitTs := time.Now()
	for time.Since(startAttachWaitTs).Seconds() < float64(prjPair.Live.Timeouts.AttachVolume) {
		attachmentId, err := ShowVolumeAttachment(&prjPair.Live, &sb, volNickname, iNickname)
		if err != nil {
			return err
		}
		if attachmentId != "" {
			prjPair.SetVolumeAttachmentId(iNickname, volNickname, attachmentId)
			break
		}
		time.Sleep(5 * time.Second)
	}

	if prjPair.Live.Instances[iNickname].AttachedVolumes[volNickname].AttachmentId == "" {
		return fmt.Errorf("cannot find newly created attachment, volume %s, instance %s", volNickname, iNickname)
	}

	blockDeviceId, er := ExecSshAndReturnLastLine(&prjPair.Live, &sb, prjPair.Live.Instances[iNickname].BestIpAddress(), fmt.Sprintf("%s\ninit_volume_attachment %s %s %d",
		InitVolumeAttachmentFunc,
		prjPair.Live.Instances[iNickname].AttachedVolumes[volNickname].Device,
		prjPair.Live.Volumes[volNickname].MountPoint,
		prjPair.Live.Volumes[volNickname].Permissions))
	if er.Error != nil {
		return er.Error
	}

	sb.WriteString(fmt.Sprintf("initialized volume %s on %s, uuid %s\n", volNickname, iNickname, blockDeviceId))
	prjPair.SetVolumeBlockDeviceId(iNickname, volNickname, blockDeviceId)

	return nil
}
