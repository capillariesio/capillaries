package deploy

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/sftp"
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

type FileDownloadSpec struct {
	InstanceNickname string
	Src              string
	Dst              string
}

func InstanceFileGroupDownDefsToSpecs(prj *Project, iNickname string, fgDef *FileGroupDownDef) ([]*FileDownloadSpec, error) {
	tsc, err := NewTunneledSshClient(prj.SshConfig, prj.Instances[iNickname].BestIpAddress())
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
				InstanceNickname: iNickname,
				Src:              w.Path(),
				Dst:              filepath.Join(fgDef.Dst, fileSubpath)})
		}
	}

	return fileDownloadSpecs, nil
}

func FileGroupDownDefsToSpecs(prj *Project, fileGroupsToDownload map[string]*FileGroupDownDef) ([]*FileDownloadSpec, error) {
	fileDownloadSpecs := make([]*FileDownloadSpec, 0)
	groupCountMap := map[string]int{}
	sbDuplicates := strings.Builder{}
	for iNickname, iDef := range prj.Instances {
		for _, fgName := range iDef.ApplicableFileGroups {
			if fgDef, ok := fileGroupsToDownload[fgName]; ok {
				instanceGroupSpecs, err := InstanceFileGroupDownDefsToSpecs(prj, iNickname, fgDef)
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

func UploadFileSftp(prj *Project, logChan chan string, iNickname string, srcPath string, dstPath string, permissions int) error {
	tsc, err := NewTunneledSshClient(prj.SshConfig, prj.Instances[iNickname].BestIpAddress())
	if err != nil {
		return err
	}
	defer tsc.Close()

	sftp, err := sftp.NewClient(tsc.SshClient)
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
		return fmt.Errorf("cannot create on upload %s: %s", dstPath, err.Error())
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

	logChan <- fmt.Sprintf("Uploaded %s to %s:%s, %d bytes", srcPath, iNickname, dstPath, bytesRead)
	return nil
}

func DownloadFileSftp(prj *Project, logChan chan string, iNickname string, srcPath string, dstPath string) error {
	tsc, err := NewTunneledSshClient(prj.SshConfig, prj.Instances[iNickname].BestIpAddress())
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

	logChan <- fmt.Sprintf("Downloaded %s:%s to %s, %d bytes", iNickname, srcPath, dstPath, bytesRead)
	return nil
}
