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

type ParsedSftpUri struct {
	User           string
	Host           string
	Port           int
	PrivateKeyPath string
	RemotePath     string
}

const DefaultSshPort int = 22

func parseSftpUri(uri string, privateKeys map[string]string) (*ParsedSftpUri, error) {
	// sftp://user[:pass]@host[:port]/path/to/file
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	if u.User == nil || u.User.Username() == "" {
		return nil, fmt.Errorf("empty username in sftp uri %s", uri)
	}

	userName := u.User.Username()

	privateKeyPath, ok := privateKeys[userName]
	if !ok {
		return nil, fmt.Errorf("username %s in sftp uri %s not found in environment configuration", userName, uri)
	}

	hostParts := strings.Split(u.Host, ":")
	if len(hostParts) < 2 {
		return &ParsedSftpUri{userName, hostParts[0], DefaultSshPort, privateKeyPath, u.Path}, nil
	}
	port, err := strconv.ParseInt(hostParts[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("cannot parse port in sftp uri %s", uri)
	}

	return &ParsedSftpUri{userName, hostParts[0], int(port), privateKeyPath, u.Path}, nil
}

func DownloadSftpFile(uri string, privateKeys map[string]string, dstFile *os.File) error {
	parsedUri, err := parseSftpUri(uri, privateKeys)
	if err != nil {
		return err
	}

	// Assume empty key password ""
	sshClientConfig, err := NewSshClientConfig(parsedUri.User, parsedUri.PrivateKeyPath, "")
	if err != nil {
		return err
	}

	sshUrl := fmt.Sprintf("%s:%d", parsedUri.Host, parsedUri.Port)

	sshClient, err := ssh.Dial("tcp", sshUrl, sshClientConfig)
	if err != nil {
		return fmt.Errorf("dial to %s failed: %s", uri, err.Error())
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("cannot create sftp client to %s: %s", uri, err.Error())
	}
	defer sftpClient.Close()

	srcFile, err := sftpClient.Open(parsedUri.RemotePath)
	if err != nil {
		return fmt.Errorf("cannot open target file for sftp download %s: %s", uri, err.Error())
	}
	defer srcFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return fmt.Errorf("cannot read for download %s: %s", uri, err.Error())
	}

	return nil
}

func UploadSftpFile(srcPath string, uri string, privateKeys map[string]string) error {
	parsedUri, err := parseSftpUri(uri, privateKeys)
	if err != nil {
		return err
	}

	// Assume empty key password ""
	sshClientConfig, err := NewSshClientConfig(parsedUri.User, parsedUri.PrivateKeyPath, "")
	if err != nil {
		return err
	}

	sshUrl := fmt.Sprintf("%s:%d", parsedUri.Host, parsedUri.Port)

	sshClient, err := ssh.Dial("tcp", sshUrl, sshClientConfig)
	if err != nil {
		return fmt.Errorf("dial to %s failed: %s", uri, err.Error())
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("cannot create sftp client to %s: %s", uri, err.Error())
	}
	defer sftpClient.Close()

	if err := sftpClient.MkdirAll(filepath.Dir(parsedUri.RemotePath)); err != nil {
		return fmt.Errorf("cannot create target dir for %s: %s", uri, err.Error())
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("cannot open for upload %s: %s", srcPath, err.Error())
	}
	defer srcFile.Close()

	dstFile, err := sftpClient.Create(parsedUri.RemotePath)
	if err != nil {
		return fmt.Errorf("cannot create on upload %s: %s", uri, err.Error())
	}

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return fmt.Errorf("cannot read for upload %s to %s : %s", srcPath, uri, err.Error())
	}

	err = dstFile.Sync()
	if err != nil {
		return fmt.Errorf("cannot flush for upload %s to %s : %s", srcPath, uri, err.Error())
	}

	return nil
}
