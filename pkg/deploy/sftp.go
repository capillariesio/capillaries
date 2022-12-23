package deploy

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type FileUploadSpec struct {
	InstanceNickname string
	Src              string
	Dst              string
	Permissions      int
}

func FileGroupUpDefsToSpecs(prj *Project, fileGroupsToUpload map[string]*FileGroupUpDef) ([]*FileUploadSpec, error) {
	fileUploadSpecs := make([]*FileUploadSpec, 0)
	for iNickname, iDef := range prj.Instances {
		for _, fgName := range iDef.ApplicableFileGroups {
			if fgDef, ok := fileGroupsToUpload[fgName]; ok {
				if err := filepath.WalkDir(fgDef.Src, func(path string, d fs.DirEntry, err error) error {
					if !d.IsDir() {
						fileSubpath := strings.ReplaceAll(path, fgDef.Src, "")
						fileUploadSpecs = append(fileUploadSpecs, &FileUploadSpec{
							InstanceNickname: iNickname,
							Src:              path,
							Dst:              filepath.Join(fgDef.Dst, fileSubpath),
							Permissions:      fgDef.Permissions})
					}
					return nil
				}); err != nil {
					return nil, fmt.Errorf("bad file group up %s in %s", fgName, iNickname)
				}
			}
		}
	}

	return fileUploadSpecs, nil
}

func UploadFileSftp(prj *Project, logChan chan string, iNickName string, srcPath string, dstPath string, permissions int) error {
	if prj.Instances[iNickName].FloatingIpAddress == "" {
		// TODO: tweak ssh cfg to connect to this instance via jumphost
		return fmt.Errorf("sftp jumphost not supported yet")
	}

	sshClient, err := NewSshClient(
		prj.SshConfig.User,
		prj.SshConfig.Host,
		prj.SshConfig.Port,
		prj.SshConfig.PrivateKeyPath,
		prj.SshConfig.PrivateKeyPassword)
	if err != nil {
		return err
	}
	conn, err := ssh.Dial("tcp", sshClient.Server, sshClient.Config)
	if err != nil {
		return fmt.Errorf("dial to %v failed %v", sshClient.Server, err)
	}
	defer conn.Close()

	sftp, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("cannot create sftp client: %s", err.Error())
	}
	defer sftp.Close()

	if err := sftp.MkdirAll(filepath.Dir(dstPath)); err != nil {
		return fmt.Errorf("cannot create target dir for %s: %s", dstPath, err.Error())
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("cannot open for upload %s: %s", srcPath, err.Error())
	}
	defer srcFile.Close()

	dstFile, err := sftp.Create(dstPath)
	if err != nil {
		return fmt.Errorf("cannot create onupload %s: %s", dstPath, err.Error())
	}

	bytesRead, err := dstFile.ReadFrom(srcFile)
	if err != nil {
		return fmt.Errorf("cannot read for upload %s: %s", srcPath, err.Error())
	}
	truePermissions, err := strconv.ParseInt(fmt.Sprintf("%d", permissions), 8, 0)
	if err != nil {
		return fmt.Errorf("cannot read oct convert permission %d: %s", permissions, err.Error())
	}

	if err := sftp.Chmod(dstPath, os.FileMode(truePermissions)); err != nil {
		return fmt.Errorf("cannot chmod %s: %s", srcPath, err.Error())
	}

	logChan <- fmt.Sprintf("Uploaded %s to %s, %d bytes", srcPath, dstPath, bytesRead)
	return nil
}
