package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const CreateInstanceUserFunc string = `
create_instance_user()
{ 
  local sftpUser=$1
  local sftpUserPublicKey=$2
  
  sudo useradd -m -d /home/$sftpUser -s /bin/bash $sftpUser
  
  sudo chmod 777 /home/$sftpUser
  mkdir /home/$sftpUser/.ssh
  sudo chmod 777 /home/$sftpUser/.ssh
  sudo echo $sftpUserPublicKey > /home/$sftpUser/.ssh/authorized_keys
  
  sudo chown $sftpUser /home/$sftpUser/.ssh/authorized_keys
  sudo chmod 600 /home/$sftpUser/.ssh/authorized_keys
  
  sudo chown $sftpUser /home/$sftpUser/.ssh
  sudo chmod 700 /home/$sftpUser/.ssh
  
  sudo chown $sftpUser /home/$sftpUser
  sudo chmod 700 /home/$sftpUser
  }
`

func NewCreateInstanceUsersCommands(iDef *InstanceDef) ([]string, error) {
	cmds := make([]string, len(iDef.Users))
	for uIdx, uDef := range iDef.Users {
		if len(uDef.Name) > 0 {
			keyPath := uDef.PublicKeyPath
			if strings.HasPrefix(keyPath, "~/") {
				homeDir, _ := os.UserHomeDir()
				keyPath = filepath.Join(homeDir, keyPath[2:])
			}
			keyBytes, err := os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("cannot read public key '%s' for user %s on %s: %s", uDef.PublicKeyPath, uDef.Name, iDef.HostName, err.Error())
			}
			key := string(keyBytes)
			if !strings.HasPrefix(key, "ssh-") && !strings.HasPrefix(key, "ecsda-") {
				return nil, fmt.Errorf("cannot copy private key '%s' on %s: public key should start with ssh or ecdsa-", uDef.PublicKeyPath, iDef.HostName)
			}

			cmds[uIdx] = fmt.Sprintf("%s\ncreate_instance_user '%s' '%s'", CreateInstanceUserFunc, uDef.Name, key)
		} else {
			return nil, fmt.Errorf("cannot create instance %s user '%s': name cannot be null, public key should start with ssh or ecdsa-", iDef.HostName, uDef.Name)
		}
	}
	return cmds, nil
}

const CopyPrivateKeyFunc string = `
copy_private_key()
{ 
  local sftpUser=$1
  local sftpUserPrivateKey=$2

  sudo echo $sftpUserPrivateKey | sed 's~\\n~\n~g' > ~/.ssh/$sftpUser
  sudo chmod 600 ~/.ssh/$sftpUser
}
`

func NewCopyPrivateKeysCommands(iDef *InstanceDef) ([]string, error) {
	cmds := make([]string, len(iDef.PrivateKeys))
	for uIdx, uDef := range iDef.PrivateKeys {
		if len(uDef.Name) > 0 {
			keyPath := uDef.PrivateKeyPath
			if strings.HasPrefix(keyPath, "~/") {
				homeDir, _ := os.UserHomeDir()
				keyPath = filepath.Join(homeDir, keyPath[2:])
			}
			keyBytes, err := os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("cannot read private key '%s' for user %s on %s: %s", keyPath, uDef.Name, iDef.HostName, err.Error())
			}
			key := string(keyBytes)
			if !strings.HasPrefix(key, "-----BEGIN OPENSSH PRIVATE KEY-----") {
				return nil, fmt.Errorf("cannot copy private key '%s' on %s: private key should start with -----BEGIN OPENSSH PRIVATE KEY-----", uDef.PrivateKeyPath, iDef.HostName)
			}

			// Make sure escaped \n remains escaped (this is how we store private keys in our json config files) with actual EOLs
			cmds[uIdx] = fmt.Sprintf("%s\ncopy_private_key '%s' '%s'", CopyPrivateKeyFunc, uDef.Name, strings.ReplaceAll(string(key), "\n", "\\n"))
		} else {
			return nil, fmt.Errorf("cannot copy private key '%s' on %s: name cannot be null", uDef.Name, iDef.HostName)
		}
	}
	return cmds, nil
}
