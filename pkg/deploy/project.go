package deploy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type ExecTimeouts struct {
	OpenstackCmd              int `json:"openstack_cmd"`
	OpenstackInstanceCreation int `json:"openstack_instance_creation"`
	AttachVolume              int `json:"attach_volume"`
}

type SecurityGroupDef struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type SubnetDef struct {
	Name string `json:"name"`
	Id   string `json:"id"`
	Cidr string `json:"cidr"`
}

type RouterDef struct {
	Name                       string `json:"name"`
	Id                         string `json:"id"`
	ExternalGatewayNetworkName string `json:"external_gateway_network_name"`
}

type NetworkDef struct {
	Name   string    `json:"name"`
	Id     string    `json:"id"`
	Subnet SubnetDef `json:"subnet"`
	Router RouterDef `json:"router"`
}

type VolumeDef struct {
	Name        string `json:"name"`
	MountPoint  string `json:"mount_point"`
	Size        int    `json:"size"`
	Permissions int    `json:"permissions"`
	Id          string `json:"id"`
}

type AttachedVolumeDef struct {
	AttachmentId  string `json:"attachment_id"`
	Device        string `json:"device"`
	BlockDeviceId string `json:"block_device_id"`
}

type ServiceDef struct {
	Env      map[string]string `json:"env"`
	Priority int               `json:"priority"`
	Cmd      map[string]string `json:"cmd"`
}

type InstanceDef struct {
	HostName             string                        `json:"host_name"`
	IpAddress            string                        `json:"ip_address"`
	FloatingIpAddress    string                        `json:"floating_ip_address"`
	FlavorName           string                        `json:"flavor"`
	ImageName            string                        `json:"image"`
	AttachedVolumes      map[string]*AttachedVolumeDef `json:"attached_volumes"`
	Id                   string                        `json:"id"`
	Service              ServiceDef                    `json:"service"`
	ApplicableFileGroups []string                      `json:"applicable_file_groups"`
}

func (iDef *InstanceDef) Clean() {
	iDef.Id = ""
	for _, volAttachDef := range iDef.AttachedVolumes {
		volAttachDef.AttachmentId = ""
		volAttachDef.Device = ""
		volAttachDef.BlockDeviceId = ""
	}
}

type SshConfigDef struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	User               string `json:"user"`
	PrivateKeyPath     string `json:"private_key_path"`
	PrivateKeyPassword string `json:"private_key_password"`
}

type FileGroupUpDef struct {
	Src         string `json:"src"`
	Dst         string `json:"dst"`
	Permissions int    `json:"permissions"`
}
type Project struct {
	DeploymentName        string                     `json:"deployment_name"`
	SshConfig             *SshConfigDef              `json:"ssh_config"`
	RootKeyName           string                     `json:"root_key_name"`
	AvailabilityZone      string                     `json:"availability_zone"`
	Timeouts              ExecTimeouts               `json:"timeouts"`
	OpenstackEnvVariables map[string]string          `json:"openstack_environment_variables"`
	SecurityGroup         SecurityGroupDef           `json:"security_group"`
	Network               NetworkDef                 `json:"network"`
	Volumes               map[string]*VolumeDef      `json:"volumes"`
	FileGroupsUp          map[string]*FileGroupUpDef `json:"file_groups_up"`
	Instances             map[string]*InstanceDef    `json:"instances"`
}

type ProjectPair struct {
	Template Project
	Live     Project
}

func (prjPair *ProjectPair) SetSecurityGroupId(newId string) {
	prjPair.Template.SecurityGroup.Id = newId
	prjPair.Live.SecurityGroup.Id = newId
}

func (prjPair *ProjectPair) SetNetworkId(newId string) {
	prjPair.Template.Network.Subnet.Id = newId
	prjPair.Live.Network.Subnet.Id = newId
}

func (prjPair *ProjectPair) SetRouterId(newId string) {
	prjPair.Template.Network.Router.Id = newId
	prjPair.Live.Network.Router.Id = newId
}

func (prjPair *ProjectPair) SetSubnetId(newId string) {
	prjPair.Template.Network.Subnet.Id = newId
	prjPair.Live.Network.Subnet.Id = newId
}

func (prjPair *ProjectPair) SetVolumeId(volNickname string, newId string) {
	prjPair.Template.Volumes[volNickname].Id = newId
	prjPair.Live.Volumes[volNickname].Id = newId
}

func (prjPair *ProjectPair) SetAttachedVolumeDevice(iNickname string, volNickname string, device string) {
	prjPair.Template.Instances[iNickname].AttachedVolumes[volNickname].Device = device
	prjPair.Live.Instances[iNickname].AttachedVolumes[volNickname].Device = device
}

func (prjPair *ProjectPair) SetVolumeAttachmentId(iNickname string, volNickname string, newId string) {
	prjPair.Template.Instances[iNickname].AttachedVolumes[volNickname].AttachmentId = newId
	prjPair.Live.Instances[iNickname].AttachedVolumes[volNickname].AttachmentId = newId
}

func (prjPair *ProjectPair) SetVolumeBlockDeviceId(iNickname string, volNickname string, newId string) {
	prjPair.Template.Instances[iNickname].AttachedVolumes[volNickname].BlockDeviceId = newId
	prjPair.Live.Instances[iNickname].AttachedVolumes[volNickname].BlockDeviceId = newId
}

func (prjPair *ProjectPair) CleanInstance(iNickname string) {
	prjPair.Template.Instances[iNickname].Clean()
	prjPair.Live.Instances[iNickname].Clean()
}

func (prjPair *ProjectPair) SetInstanceId(iNickname string, newId string) {
	prjPair.Template.Instances[iNickname].Id = newId
	prjPair.Live.Instances[iNickname].Id = newId
}

func LoadProject(prjFile string, prjParamsFile string) (*ProjectPair, string, error) {
	exec, err := os.Executable()
	if err != nil {
		return nil, "", fmt.Errorf("cannot find current executable path: %s", err.Error())
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("cannot get current dir: [%s]", err.Error())
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("cannot get home dir: [%s]", err.Error())
	}
	prjFullPath := filepath.Join(filepath.Dir(exec), prjFile)
	if _, err := os.Stat(prjFullPath); err != nil {
		prjFullPath = filepath.Join(cwd, prjFile)
		if _, err := os.Stat(prjFullPath); err != nil {
			return nil, "", fmt.Errorf("cannot find project file [%s], neither at [%s] nor at current dir [%s]: [%s]", prjFile, filepath.Dir(exec), filepath.Join(cwd, prjFile), err.Error())
		}
	}
	prjParamsFullPath := filepath.Join(filepath.Dir(exec), prjParamsFile)
	if _, err := os.Stat(prjParamsFullPath); err != nil {
		prjParamsFullPath = filepath.Join(cwd, prjParamsFile)
		if _, err := os.Stat(prjParamsFullPath); err != nil {
			prjParamsFullPath = filepath.Join(homeDir, prjParamsFile)
			if _, err := os.Stat(prjParamsFullPath); err != nil {
				return nil, "", fmt.Errorf("cannot find project params file [%s]: neither at [%s], at current dir [%s], at home dir [%s]: [%s]", prjParamsFile, filepath.Dir(exec), filepath.Join(cwd, prjParamsFile), filepath.Join(homeDir, prjParamsFile), err.Error())
			}
		}
	}

	prjBytes, err := ioutil.ReadFile(prjFullPath)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read project file %s: %s", prjFullPath, err.Error())
	}

	prjParamsBytes, err := ioutil.ReadFile(prjParamsFullPath)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read project params file %s: %s", prjParamsFullPath, err.Error())
	}

	var prjPair ProjectPair

	// Read project

	err = json.Unmarshal(prjBytes, &prjPair.Template)
	if err != nil {
		return nil, "", fmt.Errorf("cannot parse project file %s: %s", prjFullPath, err.Error())
	}

	prjString := string(prjBytes)

	// Read project params

	var prjParams map[string]string
	if err := json.Unmarshal(prjParamsBytes, &prjParams); err != nil {
		return nil, "", fmt.Errorf("cannot parse project params file %s: %s", prjParamsFullPath, err.Error())
	}

	// Replace project params

	for k, v := range prjParams {
		prjString = strings.ReplaceAll(prjString, fmt.Sprintf("{%s}", k), v)
	}

	// Re-deserialize, now with replaced params

	if err := json.Unmarshal([]byte(prjString), &prjPair.Live); err != nil {
		return nil, "", fmt.Errorf("cannot parse project file with replaced vars %s: %s", prjFullPath, err.Error())
	}

	return &prjPair, prjFullPath, nil
}

func (prj *Project) SaveProject(fullPrjPath string) error {
	prjJsonBytes, err := json.MarshalIndent(prj, "", "    ")
	if err != nil {
		return err
	}

	fPrj, err := os.Create(fullPrjPath)
	defer fPrj.Close()
	if _, err := fPrj.WriteString(string(prjJsonBytes)); err != nil {
		return err
	}
	fPrj.Sync()
	return nil
}
