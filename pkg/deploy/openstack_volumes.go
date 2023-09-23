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
  local owner=$4

  # Check if file system is already there
  local deviceBlockId=$(blkid -s UUID -o value $deviceName)
  if [ "$deviceBlockId" = "" ]; then
    # Make file system
    sudo mkfs.ext4 $deviceName
    if [ "$?" -ne "0" ]; then
      echo Error $?, cannot make file system
      return $?
    fi
  fi

  deviceBlockId=$(sudo blkid -s UUID -o value $deviceName)

  local alreadyMounted=$(cat /etc/fstab | grep $volumeMountPath)

  if [ "$alreadyMounted" = "" ]; then
    # Create mount point
    sudo mkdir -p $volumeMountPath
    if [ "$?" -ne "0" ]; then
      echo Error $?, cannot create mount dir $volumeMountPath
      return $?
    fi

    # Set permissions
    sudo chmod $permissions $volumeMountPath
    if [ "$?" -ne "0" ]; then
		echo Error $?, cannot change $volumeMountPath permissions to $permissions
        return $?
    fi

	if [ -n "$owner" ]; then
	    sudo chown $owner $volumeMountPath
		if [ "$?" -ne "0" ]; then
			echo Error $?, cannot change $volumeMountPath owner to $owner
		    return $?
		fi
	fi

    # Adds a line to /etc/fstab
    echo "UUID=$deviceBlockId   $volumeMountPath   ext4   defaults   0   2 " | sudo tee -a /etc/fstab
  fi

  # Report UUID
  echo $deviceBlockId
  return 0
}
`

func (*OpenstackDeployProvider) CreateVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder(fmt.Sprintf("openstack.CreateVolume: %s", volNickname), isVerbose)
	if prjPair.Live.Instances[iNickname].Volumes[volNickname].MountPoint == "" ||
		prjPair.Live.Instances[iNickname].Volumes[volNickname].AvailabilityZone == "" ||
		prjPair.Live.Instances[iNickname].Volumes[volNickname].Name == "" ||
		prjPair.Live.Instances[iNickname].Volumes[volNickname].Permissions == 0 ||
		prjPair.Live.Instances[iNickname].Volumes[volNickname].Owner == "" ||
		prjPair.Live.Instances[iNickname].Volumes[volNickname].Size == 0 {
		return lb.Complete(fmt.Errorf("volume name(%s), mount point(%s), availability zone(%s), size(%d), permissions(%d), owner(%s) cannot be empty",
			prjPair.Live.Instances[iNickname].Volumes[volNickname].Name,
			prjPair.Live.Instances[iNickname].Volumes[volNickname].MountPoint,
			prjPair.Live.Instances[iNickname].Volumes[volNickname].AvailabilityZone,
			prjPair.Live.Instances[iNickname].Volumes[volNickname].Size,
			prjPair.Live.Instances[iNickname].Volumes[volNickname].Permissions,
			prjPair.Live.Instances[iNickname].Volumes[volNickname].Owner))
	}

	rows, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"volume", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | ID                                   | Name                       | Status    | Size | Attached to |
	// | 8aa8a5e8-2aad-4006-8911-af7de31b08fb | sample_deployment_name_cfg | available |    1 |             |
	foundVolIdByName := findOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Instances[iNickname].Volumes[volNickname].Name)
	if prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId == "" {
		// If it was already created, save it for future use, but do not create
		if foundVolIdByName != "" {
			lb.Add(fmt.Sprintf("volume %s(%s) already there, updating project", prjPair.Live.Instances[iNickname].Volumes[volNickname].Name, foundVolIdByName))
			//fmt.Printf("Setting existing %s-%s %s\n", iNickname, volNickname, foundVolIdByName)
			prjPair.SetVolumeId(iNickname, volNickname, foundVolIdByName)
		}
	} else {
		if foundVolIdByName == "" {
			// It was supposed to be there, but it's not present, complain
			return lb.Complete(fmt.Errorf("requested volume id %s not present, consider removing this id from the project file", prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId))
		} else if prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId != foundVolIdByName {
			// It is already there, but has different id, complain
			return lb.Complete(fmt.Errorf("requested volume id %s not matching existing volume id %s", prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId, foundVolIdByName))
		}
	}

	if prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId != "" {
		lb.Add(fmt.Sprintf("volume %s(%s) already there, no need to create", prjPair.Live.Instances[iNickname].Volumes[volNickname].Name, foundVolIdByName))
		return lb.Complete(nil)
	}

	volCreateParams := []string{"volume", "create",
		"--size", fmt.Sprintf("%d", prjPair.Live.Instances[iNickname].Volumes[volNickname].Size),
		"--availability-zone", prjPair.Live.Instances[iNickname].Volumes[volNickname].AvailabilityZone,
		prjPair.Live.Instances[iNickname].Volumes[volNickname].Name}
	if prjPair.Live.Instances[iNickname].Volumes[volNickname].Type != "" {
		volCreateParams = append(volCreateParams, "--type")
		volCreateParams = append(volCreateParams, prjPair.Live.Instances[iNickname].Volumes[volNickname].Type)
	}

	rows, er = execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", volCreateParams)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | id  | 8aa8a5e8-2aad-4006-8911-af7de31b08fb
	newId := findOpenstackFieldValue(rows, "id")
	if newId == "" {
		return lb.Complete(fmt.Errorf("openstack returned empty volume id"))
	}

	lb.Add(fmt.Sprintf("created volume %s: %s(%s)", volNickname, prjPair.Live.Instances[iNickname].Volumes[volNickname].Name, newId))
	prjPair.SetVolumeId(iNickname, volNickname, newId)
	//fmt.Printf("Setting id %s-%s %s\n", iNickname, volNickname, newId)

	return lb.Complete(nil)
}

func (*OpenstackDeployProvider) DeleteVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder(fmt.Sprintf("openstack.DeleteVolume: %s", volNickname), isVerbose)
	if prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId == "" {
		return lb.Complete(fmt.Errorf("volume id for %s.%s cannot be empty", iNickname, volNickname))
	}

	rows, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"volume", "list"})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | ID                                   | Name                       | Status    | Size | Attached to |
	// | 8aa8a5e8-2aad-4006-8911-af7de31b08fb | sample_deployment_name_cfg | available |    1 |             |
	foundVolIdByName := findOpenstackColumnValue(rows, "ID", "Name", prjPair.Live.Instances[iNickname].Volumes[volNickname].Name)
	if foundVolIdByName == "" {
		lb.Add(fmt.Sprintf("volume %s not found, nothing to delete", prjPair.Live.Instances[iNickname].Volumes[volNickname].Name))
		prjPair.SetVolumeId(iNickname, volNickname, "")
		return lb.Complete(nil)
	}

	_, er = execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"volume", "delete", prjPair.Live.Instances[iNickname].Volumes[volNickname].Name})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted volume %s_%s, updating project file", iNickname, volNickname))
	prjPair.SetVolumeId(iNickname, volNickname, "")

	return lb.Complete(nil)
}

func showOpenstackVolumeAttachment(prj *Project, volNickname string, iNickname string, isVerbose bool) (string, LogMsg, error) {
	rows, er := execLocalAndParseOpenstackOutput(prj, "openstack", []string{"volume", "show", prj.Instances[iNickname].Volumes[volNickname].VolumeId})
	lb := NewLogBuilder(fmt.Sprintf("%s on %s", er.Cmd, iNickname), isVerbose)
	lb.Add(er.ToString())
	if er.Error != nil {
		logMsg, err := lb.Complete(er.Error)
		return "", logMsg, err
	}
	// | Field        | Value
	// | attachments  | [{'server_id': '...', 'attachment_id': '...', ...}] |
	foundAttachmentsJson := findOpenstackFieldValue(rows, "attachments")
	if foundAttachmentsJson == "" {
		logMsg, err := lb.Complete(fmt.Errorf("cannot find attachments for volume %s, expected to see the newly created one", prj.Instances[iNickname].Volumes[volNickname].VolumeId))
		return "", logMsg, err
	}
	var foundAttachmentsArray []map[string]string
	if err := json.Unmarshal([]byte(strings.ReplaceAll(strings.ReplaceAll(foundAttachmentsJson, "'", "\""), "None", "null")), &foundAttachmentsArray); err != nil {
		logMsg, err := lb.Complete(err)
		return "", logMsg, err
	}

	// Walk through all reported attachments and find the one that belongs to this instance
	for _, attachment := range foundAttachmentsArray {
		serverId, ok1 := attachment["server_id"]
		attachmentId, ok2 := attachment["attachment_id"]
		if ok1 && ok2 && serverId == prj.Instances[iNickname].Id {
			logMsg, _ := lb.Complete(nil)
			return attachmentId, logMsg, nil
		}
	}

	logMsg, _ := lb.Complete(nil)
	return "", logMsg, nil
}

func (*OpenstackDeployProvider) AttachVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder(fmt.Sprintf("openstack.AttachVolume: %s to %s", volNickname, iNickname), isVerbose)
	if prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId == "" || prjPair.Live.Instances[iNickname].Id == "" {
		return lb.Complete(fmt.Errorf("cannot attach volume %s(%s) to %s(%s), no empty ids allowed", volNickname, prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId, iNickname, prjPair.Live.Instances[iNickname].Id))
	}

	rows, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"server", "volume", "list", prjPair.Live.Instances[iNickname].Id})
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	// | ID                                   | Device   | Server ID                            | Volume ID                            |
	// | 8b9b3491-f083-4485-8374-258372f3db35 | /dev/vdb | 216f9481-4c9d-4530-b865-51cedfa4b8e7 | 8b9b3491-f083-4485-8374-258372f3db35 |
	foundDevice := findOpenstackColumnValue(rows, "Device", "Volume ID", prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId)
	// Do not compare/complain, just overwrite: the number of attachment does not help catch unaccounted cloud resources anyways
	if prjPair.Live.Instances[iNickname].Volumes[volNickname].Device != "" {
		if foundDevice != "" {
			lb.Add(fmt.Sprintf("volume %s already attached to %s, device %s, updating project", volNickname, iNickname, foundDevice))
		} else {
			lb.Add(fmt.Sprintf("volume %s was not attached to %s, cleaning attachment info, updating project", volNickname, iNickname))
		}
		prjPair.SetAttachedVolumeDevice(iNickname, volNickname, foundDevice)
	} else {
		if foundDevice != "" && foundDevice != prjPair.Live.Instances[iNickname].Volumes[volNickname].Device {
			lb.Add(fmt.Sprintf("volume %s already to %s, but with a different device(%s->%s), updating project", volNickname, iNickname, prjPair.Live.Instances[iNickname].Volumes[volNickname].Device, foundDevice))
			prjPair.SetAttachedVolumeDevice(iNickname, volNickname, foundDevice)
		}
	}

	if prjPair.Live.Instances[iNickname].Volumes[volNickname].Device == "" {
		rows, er := execLocalAndParseOpenstackOutput(&prjPair.Live, "openstack", []string{"server", "add", "volume", prjPair.Live.Instances[iNickname].Id, prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId})
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		// | Field     | Value                                |
		// | ID        | 8aa8a5e8-2aad-4006-8911-af7de31b08fb |
		// | Server ID | 1466c448-86c8-404b-b3f1-d95051276f38 |
		// | Volume ID | 8aa8a5e8-2aad-4006-8911-af7de31b08fb |
		// | Device    | /dev/vdb                             |
		newId := findOpenstackFieldValue(rows, "ID") // This is not the attachment id, it's a dupr of volume id
		device := findOpenstackFieldValue(rows, "Device")
		if newId == "" || device == "" {
			return lb.Complete(fmt.Errorf("got empty id/device (%s/%s) when attaching volume %s to %s", newId, device, volNickname, iNickname))
		}
		lb.Add(fmt.Sprintf("attached volume %s to %s, attachment id %s, updating project", volNickname, iNickname, newId))
		prjPair.SetAttachedVolumeDevice(iNickname, volNickname, device)
	}

	// At this point, we are sure we have good device
	// We may need to wait a few sec until the device is ready

	startAttachWaitTs := time.Now()
	for time.Since(startAttachWaitTs).Seconds() < float64(prjPair.Live.Timeouts.AttachVolume) {
		attachmentId, logMsg, err := showOpenstackVolumeAttachment(&prjPair.Live, volNickname, iNickname, isVerbose)
		lb.Add(string(logMsg))
		if err != nil {
			return lb.Complete(err)
		}
		if attachmentId != "" {
			prjPair.SetVolumeAttachmentId(iNickname, volNickname, attachmentId)
			break
		}
		time.Sleep(5 * time.Second)
	}

	if prjPair.Live.Instances[iNickname].Volumes[volNickname].AttachmentId == "" {
		return lb.Complete(fmt.Errorf("cannot find newly created attachment, volume %s, instance %s", volNickname, iNickname))
	}

	deviceBlockId, er := ExecSshAndReturnLastLine(prjPair.Live.SshConfig, prjPair.Live.Instances[iNickname].BestIpAddress(), fmt.Sprintf("%s\ninit_volume_attachment %s %s %d '%s'",
		InitVolumeAttachmentFunc,
		prjPair.Live.Instances[iNickname].Volumes[volNickname].Device,
		prjPair.Live.Instances[iNickname].Volumes[volNickname].MountPoint,
		prjPair.Live.Instances[iNickname].Volumes[volNickname].Permissions,
		prjPair.Live.Instances[iNickname].Volumes[volNickname].Owner))
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(fmt.Errorf("cannot attach volume %s, instance %s: %s", volNickname, iNickname, er.Error.Error()))
	}

	if deviceBlockId == "" || strings.HasPrefix(deviceBlockId, "Error") {
		return lb.Complete(fmt.Errorf("cannot attach volume %s, instance %s, returned blockDeviceId is: %s", volNickname, iNickname, deviceBlockId))
	}

	lb.Add(fmt.Sprintf("initialized volume %s on %s, uuid %s", volNickname, iNickname, deviceBlockId))
	prjPair.SetVolumeBlockDeviceId(iNickname, volNickname, deviceBlockId)

	return lb.Complete(nil)
}
