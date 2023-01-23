package deploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

type FileUploadSpec struct {
	IpAddress       string
	Src             string
	Dst             string
	DirPermissions  int
	FilePermissions int
	Owner           string
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
								IpAddress:       ipAddress,
								Src:             path,
								Dst:             filepath.Join(fgDef.Dst, fileSubpath),
								DirPermissions:  fgDef.DirPermissions,
								FilePermissions: fgDef.FilePermissions,
								Owner:           fgDef.Owner,
							})
						}
						return nil
					}); err != nil {
						return nil, nil, fmt.Errorf("bad file group up %s in %s", fgName, iDef.HostName)
					}
				} else {
					fileUploadSpecs = append(fileUploadSpecs, &FileUploadSpec{
						IpAddress:       ipAddress,
						Src:             fgDef.Src,
						Dst:             filepath.Join(fgDef.Dst, filepath.Base(fgDef.Src)),
						DirPermissions:  fgDef.DirPermissions,
						FilePermissions: fgDef.FilePermissions,
						Owner:           fgDef.Owner,
					})
				}
				if _, ok := afterFileUploadSpecMap[ipAddress]; !ok {
					afterFileUploadSpecMap[ipAddress] = map[string]struct{}{}
				}
				if _, ok := afterFileUploadSpecMap[ipAddress][fgName]; !ok {
					afterFileUploadSpecMap[ipAddress][fgName] = struct{}{}
					// Do not create afterSpec for empty list of commands
					if len(fgDef.After.Cmd) > 0 {
						afterFileUploadSpecs = append(afterFileUploadSpecs, &AfterFileUploadSpec{
							IpAddress: ipAddress,
							Env:       fgDef.After.Env,
							Cmd:       fgDef.After.Cmd})
					}
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
		return nil, fmt.Errorf("cannot download, the following file groups are associated with more than one instance: %s", sbDuplicates.String())
	}

	return fileDownloadSpecs, nil
}

func UploadFileSftp(prj *Project, ipAddress string, srcPath string, dstPath string, dirPermissions int, filePermissions int, owner string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder(fmt.Sprintf("Uploading %s to %s:%s", srcPath, ipAddress, dstPath), isVerbose)
	tsc, err := NewTunneledSshClient(prj.SshConfig, ipAddress)
	if err != nil {
		return lb.Complete(err)
	}
	defer tsc.Close()

	sftp, err := sftp.NewClient(tsc.SshClient)
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot create sftp client to %s: %s", ipAddress, err.Error()))
	}
	defer sftp.Close()

	pathParts := strings.Split(dstPath, string(os.PathSeparator))
	curPath := string(os.PathSeparator)
	for partIdx := 0; partIdx < len(pathParts)-1; partIdx++ {
		if pathParts[partIdx] == "" {
			continue
		}
		curPath = filepath.Join(curPath, pathParts[partIdx])
		fi, err := sftp.Stat(curPath)
		if err == nil && fi.IsDir() {
			// Nothing to do, we do not change existing directories
			continue
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return lb.Complete(fmt.Errorf("cannot check target dir %s%s: %s", ipAddress, curPath, err.Error()))
		}

		// Do not use sftp.Mkdir(), it causes SSH_FX_FAILURE when >1 clients is used in parallel
		if _, _, err := ExecSshForClient(tsc.SshClient, fmt.Sprintf("mkdir %s", curPath)); err != nil {
			if !strings.Contains(err.Error(), "File exists") {
				return lb.Complete(err)
			}
		}

		// Do not use sftp.Chmod(), it throws permission denied when >1 clients is used in parallel
		if _, _, err := ExecSshForClient(tsc.SshClient, fmt.Sprintf("sudo chmod %d %s", dirPermissions, curPath)); err != nil {
			return lb.Complete(err)
		}

		if owner != "" {
			// Do not use sftp.Chown(), it does not work for sudo
			if _, _, err := ExecSshForClient(tsc.SshClient, fmt.Sprintf("sudo chown %s %s", owner, curPath)); err != nil {
				return lb.Complete(err)
			}
		}
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot open for upload %s: %s", srcPath, err.Error()))
	}
	defer srcFile.Close()

	fi, err := srcFile.Stat()
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot obtain file info on upload %s: %s", srcPath, err.Error()))
	}

	lb.Sb.WriteString(fmt.Sprintf("(size %d) ", fi.Size()))

	dstFile, err := sftp.Create(dstPath)
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot create on upload %s%s: %s", ipAddress, dstPath, err.Error()))
	}

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot read for upload %s to %s: %s", srcPath, dstPath, err.Error()))
	}

	// sftp.Chmod 666 on a file owned by other user throws permission denied", even if existing permissions are 666 already
	// Also, sftp.Chmod tends to cause SSH_FX_FAILURE when >1 clients is used in parallel. Use sudo chmod command instead.
	if _, _, err := ExecSshForClient(tsc.SshClient, fmt.Sprintf("sudo chmod %d %s", filePermissions, dstPath)); err != nil {
		return lb.Complete(err)
	}

	if owner != "" {
		// Do not use sftp.Chown(), it does not work for sudo
		if _, _, err := ExecSshForClient(tsc.SshClient, fmt.Sprintf("sudo chown %s %s", owner, dstPath)); err != nil {
			return lb.Complete(err)
		}
	}

	return lb.Complete(nil)
}

func DownloadFileSftp(prj *Project, ipAddress string, srcPath string, dstPath string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder(fmt.Sprintf("Downloading %s:%s to %s", ipAddress, srcPath, dstPath), isVerbose)
	tsc, err := NewTunneledSshClient(prj.SshConfig, ipAddress)
	if err != nil {
		return lb.Complete(err)
	}
	defer tsc.Close()

	sftp, err := sftp.NewClient(tsc.SshClient)
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot create sftp client: %s", err.Error()))
	}
	defer sftp.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0666); err != nil {
		return lb.Complete(fmt.Errorf("cannot create target dir for %s: %s", dstPath, err.Error()))
	}

	srcFile, err := sftp.Open(srcPath)
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot open for download %s: %s", srcPath, err.Error()))
	}
	defer srcFile.Close()

	fi, err := srcFile.Stat()
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot obtain file info on download %s: %s", srcPath, err.Error()))
	}

	lb.Sb.WriteString(fmt.Sprintf("(size %d) ", fi.Size()))

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot create on download %s: %s", dstPath, err.Error()))
	}

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return lb.Complete(fmt.Errorf("cannot read for download %s: %s", srcPath, err.Error()))
	}

	return lb.Complete(nil)
}
