package deploy

import (
	"fmt"
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
		if len(uDef.Name) > 0 && (strings.HasPrefix(uDef.PublicKey, "ssh-") || strings.HasPrefix(uDef.PublicKey, "ecdsa-")) {
			cmds[uIdx] = fmt.Sprintf("%s\ncreate_instance_user '%s' '%s'",
				CreateInstanceUserFunc, uDef.Name, uDef.PublicKey)
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
		if len(uDef.Name) > 0 && strings.HasPrefix(uDef.PrivateKey, "-----BEGIN OPENSSH PRIVATE KEY-----") {
			// Make sure escaped \n remains escaped (this is how we store private keys in our json config files) with actual EOLs
			cmds[uIdx] = fmt.Sprintf("%s\ncopy_private_key '%s' '%s'",
				CopyPrivateKeyFunc, uDef.Name, strings.ReplaceAll(uDef.PrivateKey, "\n", "\\n"))
		} else {
			return nil, fmt.Errorf("cannot copy private key '%s' on %s: name cannot be null, private key should start with -----BEGIN OPENSSH PRIVATE KEY-----", uDef.Name, iDef.HostName)
		}
	}
	return cmds, nil
}
