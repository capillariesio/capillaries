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

type SecurityGroupRuleDef struct {
	Desc      string `json:"desc"`      // human-readable
	Id        string `json:"id"`        // guid
	Protocol  string `json:"protocol"`  // tcp
	Ethertype string `json:"ethertype"` // IPv4
	RemoteIp  string `json:"remote_ip"` // 0.0.0.0/0
	Port      int    `json:"port"`      // 22
	Direction string `json:"direction"` // ingress
}

type SecurityGroupDef struct {
	Name  string                  `json:"name"`
	Id    string                  `json:"id"`
	Rules []*SecurityGroupRuleDef `json:"rules"`
}

func (sg *SecurityGroupDef) Clean() {
	sg.Id = ""
	for _, r := range sg.Rules {
		r.Id = ""
	}
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
	Owner       string `json:"owner"`
	Id          string `json:"id"`
}

type AttachedVolumeDef struct {
	AttachmentId  string `json:"attachment_id"`
	Device        string `json:"device"`
	BlockDeviceId string `json:"block_device_id"`
}

type ServiceCommandsDef struct {
	Setup []string `json:"setup"`
	Start []string `json:"start"`
	Stop  []string `json:"stop"`
}
type ServiceDef struct {
	Env map[string]string  `json:"env"`
	Cmd ServiceCommandsDef `json:"cmd"`
}

type UserDef struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}
type PrivateKeyDef struct {
	Name       string `json:"name"`
	PrivateKey string `json:"private_key"`
}
type InstanceDef struct {
	HostName              string                        `json:"host_name"`
	SecurityGroupNickname string                        `json:"security_group"`
	IpAddress             string                        `json:"ip_address"`
	FloatingIpAddress     string                        `json:"floating_ip_address,omitempty"`
	FlavorName            string                        `json:"flavor"`
	ImageName             string                        `json:"image"`
	AttachedVolumes       map[string]*AttachedVolumeDef `json:"attached_volumes,omitempty"`
	Id                    string                        `json:"id"`
	Users                 []UserDef                     `json:"users,omitempty"`
	PrivateKeys           []PrivateKeyDef               `json:"private_keys,omitempty"`
	Service               ServiceDef                    `json:"service"`
	ApplicableFileGroups  []string                      `json:"applicable_file_groups,omitempty"`
}

func (iDef *InstanceDef) BestIpAddress() string {
	if iDef.FloatingIpAddress != "" {
		return iDef.FloatingIpAddress
	}
	return iDef.IpAddress
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
	BastionIpAddress   string `json:"bastion_ip_address"`
	Port               int    `json:"port"`
	User               string `json:"user"`
	PrivateKeyPath     string `json:"private_key_path"`
	PrivateKeyPassword string `json:"private_key_password"`
}

type FileGroupUpAfter struct {
	Env map[string]string `json:"env,omitempty"`
	Cmd []string          `json:"cmd,omitempty"`
}

type FileGroupUpDef struct {
	Src             string           `json:"src"`
	Dst             string           `json:"dst"`
	DirPermissions  int              `json:"dir_permissions"`
	FilePermissions int              `json:"file_permissions"`
	Owner           string           `json:"owner,omitempty"`
	After           FileGroupUpAfter `json:"after,omitempty"`
}

type FileGroupDownDef struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}
type Project struct {
	DeploymentName        string                       `json:"deployment_name"`
	SshConfig             *SshConfigDef                `json:"ssh_config"`
	RootKeyName           string                       `json:"root_key_name"`
	AvailabilityZone      string                       `json:"availability_zone"`
	Timeouts              ExecTimeouts                 `json:"timeouts"`
	OpenstackEnvVariables map[string]string            `json:"openstack_environment_variables"`
	SecurityGroups        map[string]*SecurityGroupDef `json:"security_groups"`
	Network               NetworkDef                   `json:"network"`
	Volumes               map[string]*VolumeDef        `json:"volumes"`
	FileGroupsUp          map[string]*FileGroupUpDef   `json:"file_groups_up"`
	FileGroupsDown        map[string]*FileGroupDownDef `json:"file_groups_down"`
	Instances             map[string]*InstanceDef      `json:"instances"`
}

type ProjectPair struct {
	Template           Project
	Live               Project
	ProjectFileDirPath string
}

func (prjPair *ProjectPair) SetSecurityGroupId(sgNickname string, newId string) {
	prjPair.Template.SecurityGroups[sgNickname].Id = newId
	prjPair.Live.SecurityGroups[sgNickname].Id = newId
}

func (prjPair *ProjectPair) SetSecurityGroupRuleId(sgNickname string, ruleIdx int, newId string) {
	prjPair.Template.SecurityGroups[sgNickname].Rules[ruleIdx].Id = newId
	prjPair.Live.SecurityGroups[sgNickname].Rules[ruleIdx].Id = newId
}

func (prjPair *ProjectPair) CleanSecurityGroup(sgNickname string) {
	prjPair.Template.SecurityGroups[sgNickname].Clean()
	prjPair.Live.SecurityGroups[sgNickname].Clean()
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

func (prj *Project) validate() error {
	// Check instance presence and uniqueness: hostnames, ip addresses, security groups
	hostnameMap := map[string]struct{}{}
	internalIpMap := map[string]struct{}{}
	floatingIpMap := map[string]struct{}{}
	referencedUpFileGroups := map[string]struct{}{}
	referencedDownFileGroups := map[string]struct{}{}
	for iNickname, iDef := range prj.Instances {
		if iDef.HostName == "" {
			return fmt.Errorf("instance %s has empty hostname", iNickname)
		}
		if _, ok := hostnameMap[iDef.HostName]; ok {
			return fmt.Errorf("instances share hostname %s", iDef.HostName)
		}
		hostnameMap[iDef.HostName] = struct{}{}

		if iDef.IpAddress == "" {
			return fmt.Errorf("instance %s has empty ip address", iNickname)
		}
		if _, ok := internalIpMap[iDef.IpAddress]; ok {
			return fmt.Errorf("instances share internal ip %s", iDef.IpAddress)
		}
		internalIpMap[iDef.IpAddress] = struct{}{}

		if iDef.FloatingIpAddress != "" {
			if _, ok := floatingIpMap[iDef.FloatingIpAddress]; ok {
				return fmt.Errorf("instances share floating ip %s", iDef.FloatingIpAddress)
			}
			floatingIpMap[iDef.FloatingIpAddress] = struct{}{}
		}

		// Security groups
		if iDef.SecurityGroupNickname == "" {
			return fmt.Errorf("instance %s has empty security group", iNickname)
		}
		if _, ok := prj.SecurityGroups[iDef.SecurityGroupNickname]; !ok {
			return fmt.Errorf("instance %s has invalid security group %s", iNickname, iDef.SecurityGroupNickname)
		}

		// File groups in instances
		for _, fgName := range iDef.ApplicableFileGroups {
			_, okUp := prj.FileGroupsUp[fgName]
			_, okDown := prj.FileGroupsDown[fgName]
			if okUp && okDown {
				return fmt.Errorf("instance %s has file group %s referenced as up and down, pick one: either up or down", iNickname, fgName)
			} else if okUp {
				referencedUpFileGroups[fgName] = struct{}{}
			} else if okDown {
				referencedDownFileGroups[fgName] = struct{}{}
			} else {
				return fmt.Errorf("instance %s has invalid file group %s", iNickname, fgName)
			}
		}
	}

	// Need at least one floating ip address
	if len(floatingIpMap) == 0 {
		return fmt.Errorf("none of the instances has floating ip address, at least one must have it")
	}

	// All file groups shpuld be referenced, otherwise useless
	for fgName, _ := range prj.FileGroupsUp {
		if _, ok := referencedUpFileGroups[fgName]; !ok {
			return fmt.Errorf("up file group %s not reference by any instance, consider removing it", fgName)
		}
	}
	for fgName, _ := range prj.FileGroupsDown {
		if _, ok := referencedDownFileGroups[fgName]; !ok {
			return fmt.Errorf("down file group %s not reference by any instance, consider removing it", fgName)
		}
	}

	return nil
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

	prjPair := ProjectPair{ProjectFileDirPath: filepath.Dir(prjFullPath)}

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
		// Revert \n unescaping in parameter values - we want to preserve "\n"
		if strings.Contains(v, "\n") {
			v = strings.ReplaceAll(v, "\n", "\\n")
		}
		prjString = strings.ReplaceAll(prjString, fmt.Sprintf("{%s}", k), v)
	}

	// Re-deserialize, now with replaced params

	if err := json.Unmarshal([]byte(prjString), &prjPair.Live); err != nil {
		return nil, "", fmt.Errorf("cannot parse project file with replaced vars %s: %s", prjFullPath, err.Error())
	}

	if err := prjPair.Live.validate(); err != nil {
		return nil, "", fmt.Errorf("cannot load project file %s: %s", prjFullPath, err.Error())
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
