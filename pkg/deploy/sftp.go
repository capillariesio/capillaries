package deploy

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

type FileUploadSpec struct {
	IpAddress string
	Src       string
	Dst       string
	//Permissions int
}

type AfterFileUploadSpec struct {
	IpAddress string
	Env       map[string]string
	Cmd       []string
}

func FileGroupUpDefsToSpecs(prj *Project, fileGroupsToUpload map[string]*FileGroupUpDef) ([]*FileUploadSpec, []*AfterFileUploadSpec, error) {
	fileUploadSpecs := make([]*FileUploadSpec, 0)
	afterFileUploadSpecMap := map[string]map[string]struct{}{} // inst_ip->group_name->struct{}
	afterFileUploadSpecs := make([]*AfterFileUploadSpec, 0)
	for _, iDef := range prj.Instances {
		ipAddress := iDef.BestIpAddress()
		for _, fgName := range iDef.ApplicableFileGroups {
			if fgDef, ok := fileGroupsToUpload[fgName]; ok {
				fi, err := os.Stat(fgDef.Src)
				if err != nil {
					return nil, nil, fmt.Errorf("cannot analyze path %s in %s: %s", fgDef.Src, fgName, err.Error())
				}
				if fi.IsDir() {
					if err := filepath.WalkDir(fgDef.Src, func(path string, d fs.DirEntry, err error) error {
						if !d.IsDir() {
							fileSubpath := strings.ReplaceAll(path, fgDef.Src, "")
							fileUploadSpecs = append(fileUploadSpecs, &FileUploadSpec{
								IpAddress: ipAddress,
								Src:       path,
								Dst:       filepath.Join(fgDef.Dst, fileSubpath),
								//Permissions: fgDef.Permissions
							})
						}
						return nil
					}); err != nil {
						return nil, nil, fmt.Errorf("bad file group up %s in %s", fgName, iDef.HostName)
					}
				} else {
					fileUploadSpecs = append(fileUploadSpecs, &FileUploadSpec{
						IpAddress: ipAddress,
						Src:       fgDef.Src,
						Dst:       filepath.Join(fgDef.Dst, filepath.Base(fgDef.Src)),
						//Permissions: fgDef.Permissions
					})
				}
				if _, ok := afterFileUploadSpecMap[ipAddress]; !ok {
					afterFileUploadSpecMap[ipAddress] = map[string]struct{}{}
				}
				if _, ok := afterFileUploadSpecMap[ipAddress][fgName]; !ok {
					afterFileUploadSpecMap[ipAddress][fgName] = struct{}{}
					afterFileUploadSpecs = append(afterFileUploadSpecs, &AfterFileUploadSpec{
						IpAddress: ipAddress,
						Env:       fgDef.After.Env,
						Cmd:       fgDef.After.Cmd})
				}
			}
		}
	}

	return fileUploadSpecs, afterFileUploadSpecs, nil
}

type FileDownloadSpec struct {
	IpAddress string
	Src       string
	Dst       string
}

func InstanceFileGroupDownDefsToSpecs(prj *Project, ipAddress string, fgDef *FileGroupDownDef) ([]*FileDownloadSpec, error) {
	tsc, err := NewTunneledSshClient(prj.SshConfig, ipAddress)
	if err != nil {
		return nil, err
	}
	defer tsc.Close()

	sftp, err := sftp.NewClient(tsc.SshClient)
	if err != nil {
		return nil, fmt.Errorf("cannot create sftp client: %s", err.Error())
	}
	defer sftp.Close()

	fileDownloadSpecs := make([]*FileDownloadSpec, 0)
	w := sftp.Walk(fgDef.Src)
	for w.Step() {
		if w.Err() != nil {
			return nil, fmt.Errorf("sftp walker error in %s, %s: %s", fgDef.Src, w.Path(), w.Err().Error())
		}
		if !w.Stat().IsDir() {
			fileSubpath := strings.ReplaceAll(w.Path(), fgDef.Src, "")
			fileDownloadSpecs = append(fileDownloadSpecs, &FileDownloadSpec{
				IpAddress: ipAddress,
				Src:       w.Path(),
				Dst:       filepath.Join(fgDef.Dst, fileSubpath)})
		}
	}

	return fileDownloadSpecs, nil
}

func FileGroupDownDefsToSpecs(prj *Project, fileGroupsToDownload map[string]*FileGroupDownDef) ([]*FileDownloadSpec, error) {
	fileDownloadSpecs := make([]*FileDownloadSpec, 0)
	groupCountMap := map[string]int{}
	sbDuplicates := strings.Builder{}
	for _, iDef := range prj.Instances {
		for _, fgName := range iDef.ApplicableFileGroups {
			if fgDef, ok := fileGroupsToDownload[fgName]; ok {
				instanceGroupSpecs, err := InstanceFileGroupDownDefsToSpecs(prj, iDef.BestIpAddress(), fgDef)
				if err != nil {
					return nil, err
				}
				fileDownloadSpecs = append(fileDownloadSpecs, instanceGroupSpecs...)

				// Additional check: do not download same group from more than one instance
				if _, ok := groupCountMap[fgName]; !ok {
					groupCountMap[fgName] = 0
				}
				groupCountMap[fgName] = groupCountMap[fgName] + 1
				if groupCountMap[fgName] == 2 {
					sbDuplicates.WriteString(fmt.Sprintf("%s;", fgName))
				}
			}
		}
	}

	if sbDuplicates.Len() > 0 {
		return nil, fmt.Errorf("cannot download, the following file groups are assoiated with more than one instance: %s", sbDuplicates.String())
	}

	return fileDownloadSpecs, nil
}

func UploadFileSftp(prj *Project, logChan chan string, ipAddress string, srcPath string, dstPath string) error {
	tsc, err := NewTunneledSshClient(prj.SshConfig, ipAddress)
	if err != nil {
		return err
	}
	defer tsc.Close()

	sftp, err := sftp.NewClient(tsc.SshClient)
	if err != nil {
		return fmt.Errorf("cannot create sftp client to %s: %s", ipAddress, err.Error())
	}
	defer sftp.Close()

	if err := sftp.MkdirAll(filepath.Dir(dstPath)); err != nil {
		return fmt.Errorf("cannot create target dir for %s %s: %s", ipAddress, dstPath, err.Error())
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("cannot open for upload %s: %s", srcPath, err.Error())
	}
	defer srcFile.Close()

	dstFile, err := sftp.Create(dstPath)
	if err != nil {
		return fmt.Errorf("cannot create on upload %s %s: %s", ipAddress, dstPath, err.Error())
	}

	bytesRead, err := dstFile.ReadFrom(srcFile)
	if err != nil {
		return fmt.Errorf("cannot read for upload %s: %s", srcPath, err.Error())
	}
	// truePermissions, err := strconv.ParseInt(fmt.Sprintf("%d", permissions), 8, 0)
	// if err != nil {
	// 	return fmt.Errorf("cannot read oct convert permission %s %d: %s", ipAddress, permissions, err.Error())
	// }

	// if err := sftp.Chmod(dstPath, os.FileMode(truePermissions)); err != nil {
	// 	return fmt.Errorf("cannot chmod %s %s: %s", ipAddress, dstPath, err.Error())
	// }

	logChan <- fmt.Sprintf("Uploaded %s to %s:%s, %d bytes", srcPath, ipAddress, dstPath, bytesRead)
	return nil
}

func DownloadFileSftp(prj *Project, logChan chan string, ipAddress string, srcPath string, dstPath string) error {
	tsc, err := NewTunneledSshClient(prj.SshConfig, ipAddress)
	if err != nil {
		return err
	}
	defer tsc.Close()

	sftp, err := sftp.NewClient(tsc.SshClient)
	if err != nil {
		return fmt.Errorf("cannot create sftp client: %s", err.Error())
	}
	defer sftp.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0666); err != nil {
		return fmt.Errorf("cannot create target dir for %s: %s", dstPath, err.Error())
	}

	srcFile, err := sftp.Open(srcPath)
	if err != nil {
		return fmt.Errorf("cannot open for download %s: %s", srcPath, err.Error())
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("cannot create on download %s: %s", dstPath, err.Error())
	}

	bytesRead, err := dstFile.ReadFrom(srcFile)
	if err != nil {
		return fmt.Errorf("cannot read for download %s: %s", srcPath, err.Error())
	}

	logChan <- fmt.Sprintf("Downloaded %s:%s to %s, %d bytes", ipAddress, srcPath, dstPath, bytesRead)
	return nil
}
