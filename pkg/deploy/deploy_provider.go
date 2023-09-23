package deploy

import "fmt"

type DeployProvider interface {
	CreateFloatingIp(prjPair *ProjectPair, isVerbose bool) (LogMsg, error)
	DeleteFloatingIp(prjPair *ProjectPair, isVerbose bool) (LogMsg, error)
	CreateSecurityGroups(prjPair *ProjectPair, isVerbose bool) (LogMsg, error)
	DeleteSecurityGroups(prjPair *ProjectPair, isVerbose bool) (LogMsg, error)
	CreateNetworking(prjPair *ProjectPair, isVerbose bool) (LogMsg, error)
	DeleteNetworking(prjPair *ProjectPair, isVerbose bool) (LogMsg, error)
	GetFlavorIds(prjPair *ProjectPair, flavorMap map[string]string, isVerbose bool) (LogMsg, error)
	GetImageIds(prjPair *ProjectPair, imageMap map[string]string, isVerbose bool) (LogMsg, error)
	GetKeypairs(prjPair *ProjectPair, keypairMap map[string]struct{}, isVerbose bool) (LogMsg, error)
	CreateInstanceAndWaitForCompletion(prjPair *ProjectPair, iNickname string, flavorId string, imageId string, availabilityZone string, isVerbose bool) (LogMsg, error)
	DeleteInstance(prjPair *ProjectPair, iNickname string, isVerbose bool) (LogMsg, error)
	CreateVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error)
	AttachVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error)
	DeleteVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error)
}

type OpenstackDeployProvider struct{}

type AwsDeployProvider struct{}

func DeployProviderFactory(deployProviderName string) (DeployProvider, error) {
	if deployProviderName == DeployProviderOpenstack {
		return &OpenstackDeployProvider{}, nil
	} else if deployProviderName == DeployProviderAws {
		return &AwsDeployProvider{}, nil
	} else {
		return nil, fmt.Errorf("unsupported deploy provider %s", deployProviderName)
	}
}

func reportPublicIp(prj *Project) {
	fmt.Printf(`
Public IP reserved, now you can use it for SSH jumphost in your ~/.ssh/config:

Host %s
  User %s
  StrictHostKeyChecking=no
  UserKnownHostsFile=/dev/null
  IdentityFile %s

Also, you may find it convenient to use in your commands:

export BASTION_IP=%s

`,
		prj.SshConfig.ExternalIpAddress,
		prj.SshConfig.User,
		prj.SshConfig.PrivateKeyPath,
		prj.SshConfig.ExternalIpAddress)
}
