package xfer

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type ParsedSftpUrl struct {
	User           string
	Host           string
	Port           int
	PrivateKeyPath string
	RemotePath     string
}

const DefaultSshPort int = 22

func parseSftpUrl(fileUrl string, privateKeys map[string]string) (*ParsedSftpUrl, error) {
	// sftp://user[:pass]@host[:port]/path/to/file
	u, err := url.Parse(fileUrl)
	if err != nil {
		return nil, err
	}

	if u.User == nil || u.User.Username() == "" {
		return nil, fmt.Errorf("empty username in sftp url %s", fileUrl)
	}

	userName := u.User.Username()

	privateKeyPath, ok := privateKeys[userName]
	if !ok {
		return nil, fmt.Errorf("username %s in sftp url %s not found in environment configuration", userName, fileUrl)
	}

	hostParts := strings.Split(u.Host, ":")
	if len(hostParts) < 2 {
		return &ParsedSftpUrl{userName, hostParts[0], DefaultSshPort, privateKeyPath, u.Path}, nil
	}
	port, err := strconv.ParseInt(hostParts[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("cannot parse port in sftp url %s", fileUrl)
	}

	return &ParsedSftpUrl{userName, hostParts[0], int(port), privateKeyPath, u.Path}, nil
}

func DownloadSftpFile(url string, privateKeys map[string]string, dstFile *os.File) error {
	parsedUrl, err := parseSftpUrl(url, privateKeys)
	if err != nil {
		return err
	}

	// Assume empty key password ""
	sshClientConfig, err := NewSshClientConfig(parsedUrl.User, parsedUrl.PrivateKeyPath, "")
	if err != nil {
		return err
	}

	sshUrl := fmt.Sprintf("%s:%d", parsedUrl.Host, parsedUrl.Port)

	sshClient, err := ssh.Dial("tcp", sshUrl, sshClientConfig)
	if err != nil {
		return fmt.Errorf("dial to %s failed: %s", url, err.Error())
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("cannot create sftp client to %s: %s", url, err.Error())
	}
	defer sftpClient.Close()

	srcFile, err := sftpClient.Open(parsedUrl.RemotePath)
	if err != nil {
		return fmt.Errorf("cannot open target file for sftp download %s: %s", url, err.Error())
	}
	defer srcFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return fmt.Errorf("cannot read for download %s: %s", url, err.Error())
	}

	return nil
}

func UploadSftpFile(srcPath string, url string, privateKeys map[string]string) error {
	parsedUrl, err := parseSftpUrl(url, privateKeys)
	if err != nil {
		return err
	}

	// Assume empty key password ""
	sshClientConfig, err := NewSshClientConfig(parsedUrl.User, parsedUrl.PrivateKeyPath, "")
	if err != nil {
		return err
	}

	sshUrl := fmt.Sprintf("%s:%d", parsedUrl.Host, parsedUrl.Port)

	sshClient, err := ssh.Dial("tcp", sshUrl, sshClientConfig)
	if err != nil {
		return fmt.Errorf("dial to %s failed: %s", url, err.Error())
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("cannot create sftp client to %s: %s", url, err.Error())
	}
	defer sftpClient.Close()

	if err := sftpClient.MkdirAll(filepath.Dir(parsedUrl.RemotePath)); err != nil {
		return fmt.Errorf("cannot create target dir for %s: %s", url, err.Error())
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("cannot open for upload %s: %s", srcPath, err.Error())
	}
	defer srcFile.Close()

	dstFile, err := sftpClient.Create(parsedUrl.RemotePath)
	if err != nil {
		return fmt.Errorf("cannot create on upload %s: %s", url, err.Error())
	}

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return fmt.Errorf("cannot read for upload %s to %s : %s", srcPath, url, err.Error())
	}

	err = dstFile.Sync()
	if err != nil {
		return fmt.Errorf("cannot flush for upload %s to %s : %s", srcPath, url, err.Error())
	}

	return nil
}
