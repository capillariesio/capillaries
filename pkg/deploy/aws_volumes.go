package deploy

import (
	"fmt"
	"strings"
	"time"
)

func (*AwsDeployProvider) CreateVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder(fmt.Sprintf("aws.CreateVolume: %s", volNickname), isVerbose)
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

	foundVolIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-volumes",
		"--filter", "Name=tag:name,Values=" + prjPair.Live.Instances[iNickname].Volumes[volNickname].Name},
		".Volumes[0].VolumeId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

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

	newId, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "create-volume",
		"--availability-zone", prjPair.Live.Instances[iNickname].Volumes[volNickname].AvailabilityZone,
		"--size", fmt.Sprintf("%d", prjPair.Live.Instances[iNickname].Volumes[volNickname].Size),
		"--volume-type", prjPair.Live.Instances[iNickname].Volumes[volNickname].Type,
		"--tag-specification", fmt.Sprintf("ResourceType=volume,Tags=[{Key=name,Value=%s}]", prjPair.Live.Instances[iNickname].Volumes[volNickname].Name)},
		".VolumeId", false)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("created volume %s: %s(%s)", volNickname, prjPair.Live.Instances[iNickname].Volumes[volNickname].Name, newId))
	prjPair.SetVolumeId(iNickname, volNickname, newId)

	return lb.Complete(nil)
}

func (*AwsDeployProvider) AttachVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.AttachVolume", isVerbose)
	if prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId == "" || prjPair.Live.Instances[iNickname].Id == "" {
		return lb.Complete(fmt.Errorf("cannot attach volume %s(%s) to %s(%s), no empty ids allowed", volNickname, prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId, iNickname, prjPair.Live.Instances[iNickname].Id))
	}

	foundDevice, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-volumes",
		"--volume-ids", prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId},
		".Volumes[0].Attachments[0].Device", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

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
		newDevice, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "attach-volume",
			"--volume-id", prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId,
			"--intance-id", prjPair.Live.Instances[iNickname].Id},
			".Device", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		lb.Add(fmt.Sprintf("attached volume %s to %s, device %s, updating project", volNickname, iNickname, newDevice))
		prjPair.SetAttachedVolumeDevice(iNickname, volNickname, newDevice)
	}

	// At this point, we are sure we have good device
	// We may need to wait a few sec until the device is ready

	startAttachWaitTs := time.Now()
	for time.Since(startAttachWaitTs).Seconds() < float64(prjPair.Live.Timeouts.AttachVolume) {

		state, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-volumes",
			"--volume-ids", prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId},
			".Volumes[0].Attachments[0].State", false)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}

		if state == "attached" {
			break
		}
		if state != "attaching" {
			return lb.Complete(fmt.Errorf("cannot attach volume %s, unknown state %s", volNickname, state))
		}
		time.Sleep(5 * time.Second)
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

func (*AwsDeployProvider) DeleteVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.DeleteVolume", isVerbose)
	if prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId == "" {
		return lb.Complete(fmt.Errorf("volume id for %s.%s cannot be empty", iNickname, volNickname))
	}

	foundVolIdByName, er := ExecLocalAndGetJsonString(&prjPair.Live, "aws", []string{"ec2", "describe-volumes",
		"--filter", "Name=tag:name,Values=" + prjPair.Live.Instances[iNickname].Volumes[volNickname].Name},
		".Volumes[0].VolumeId", true)
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	if foundVolIdByName == "" {
		lb.Add(fmt.Sprintf("volume %s not found, nothing to delete", prjPair.Live.Instances[iNickname].Volumes[volNickname].Name))
		prjPair.SetVolumeId(iNickname, volNickname, "")
		return lb.Complete(nil)
	}

	if foundVolIdByName != prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId {
		lb.Add(fmt.Sprintf("volume %s found, it has id %s, does not match known id %s", prjPair.Live.Instances[iNickname].Volumes[volNickname].Name, foundVolIdByName, prjPair.Live.Instances[iNickname].Volumes[volNickname].VolumeId))
		return lb.Complete(nil)
	}

	er = ExecLocal(&prjPair.Live, "aws", []string{"ec2", "delete-volume", "--volume-id", foundVolIdByName}, prjPair.Live.CliEnvVars, "")
	lb.Add(er.ToString())
	if er.Error != nil {
		return lb.Complete(er.Error)
	}

	lb.Add(fmt.Sprintf("deleted volume %s_%s, updating project file", iNickname, volNickname))
	prjPair.SetVolumeId(iNickname, volNickname, "")

	return lb.Complete(nil)
}
