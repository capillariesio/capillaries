package xfer

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

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
		return nil, fmt.Errorf("username %s in sftp uri %s ot found in enviroment configuration", userName, uri)
	}

	port := int64(22)
	if u.Port() != "" {
		port, err = strconv.ParseInt(u.Port(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot parse port in sftp uri %s", uri)
		}
	}

	return &ParsedSftpUri{userName, u.Host, int(port), privateKeyPath, u.Path}, nil
}

func DownloadSftpFile(uri string, privateKeys map[string]string, dstFile *os.File) error {
	parsedUri, err := parseSftpUri(uri, privateKeys)

	// Assume empty key password ""
	sshClientConfig, err := NewSshClientConfig(parsedUri.User, parsedUri.Host, parsedUri.Port, parsedUri.PrivateKeyPath, "")
	if err != nil {
		return err
	}

	sshUrl := fmt.Sprintf("%s:%d", parsedUri.Host, parsedUri.Port)

	sshClient, err := ssh.Dial("tcp", sshUrl, sshClientConfig)
	if err != nil {
		return fmt.Errorf("dial to %s failed: %s", uri, err.Error())
	}
	defer sshClient.Close()

	sftp, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("cannot create sftp client to %s: %s", uri, err.Error())
	}
	defer sftp.Close()

	srcFile, err := sftp.Open(parsedUri.RemotePath)
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
