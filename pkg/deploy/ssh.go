package deploy

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SshClient struct {
	Config *ssh.ClientConfig
	Server string
}

func signerFromPem(pemBytes []byte, password []byte) (ssh.Signer, error) {

	// read pem block
	err := errors.New("Pem decode failed, no key found")
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		return nil, err
	}

	// handle encrypted key
	if x509.IsEncryptedPEMBlock(pemBlock) {
		// decrypt PEM
		pemBlock.Bytes, err = x509.DecryptPEMBlock(pemBlock, []byte(password))
		if err != nil {
			return nil, fmt.Errorf("Decrypting PEM block failed %v", err)
		}

		// get RSA, EC or DSA key
		key, err := parsePemBlock(pemBlock)
		if err != nil {
			return nil, err
		}

		// generate signer instance from key
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			return nil, fmt.Errorf("Creating signer from encrypted key failed %v", err)
		}

		return signer, nil
	} else {
		// generate signer instance from plain key
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing plain private key failed %v", err)
		}

		return signer, nil
	}
}

func parsePemBlock(block *pem.Block) (interface{}, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing PKCS private key failed %v", err)
		} else {
			return key, nil
		}
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing EC private key failed %v", err)
		} else {
			return key, nil
		}
	case "DSA PRIVATE KEY":
		key, err := ssh.ParseDSAPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing DSA private key failed %v", err)
		} else {
			return key, nil
		}
	default:
		return nil, fmt.Errorf("Parsing private key failed, unsupported key type %q", block.Type)
	}
}

// func NewSshClient(user string, host string, port int, privateKeyPath string, privateKeyPassword string) (*SshClient, error) {
// 	// read private key file
// 	pemBytes, err := ioutil.ReadFile(privateKeyPath)
// 	if err != nil {
// 		return nil, fmt.Errorf("Reading private key file failed %v", err)
// 	}
// 	// create signer
// 	signer, err := signerFromPem(pemBytes, []byte(privateKeyPassword))
// 	if err != nil {
// 		return nil, err
// 	}
// 	// build SSH client config
// 	config := &ssh.ClientConfig{
// 		User: user,
// 		Auth: []ssh.AuthMethod{
// 			ssh.PublicKeys(signer),
// 		},
// 		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
// 			// use OpenSSH's known_hosts file if you care about host validation
// 			return nil
// 		},
// 	}

// 	client := &SshClient{
// 		Config: config,
// 		Server: fmt.Sprintf("%v:%v", host, port),
// 	}

// 	return client, nil
// }

func NewSshClientConfig(user string, host string, port int, privateKeyPath string, privateKeyPassword string) (*ssh.ClientConfig, error) {
	pemBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Reading private key file failed %v", err)
	}

	signer, err := signerFromPem(pemBytes, []byte(privateKeyPassword))
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// use OpenSSH's known_hosts file if you care about host validation
			return nil
		},
	}, nil
}

func (sshClient *SshClient) RunCommand(cmd string) ExecResult {
	// open connection
	conn, err := ssh.Dial("tcp", sshClient.Server, sshClient.Config)
	if err != nil {
		return ExecResult{cmd, "", "", 0, fmt.Errorf("Dial to %v failed %v", sshClient.Server, err)}
	}
	defer conn.Close()

	// open session
	session, err := conn.NewSession()
	if err != nil {
		return ExecResult{cmd, "", "", 0, fmt.Errorf("Create session for %v failed %v", sshClient.Server, err)}
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	runStartTime := time.Now()
	err = session.Run(cmd)
	elapsed := time.Since(runStartTime).Seconds()

	return ExecResult{cmd, string(stdout.Bytes()), string(stderr.Bytes()), elapsed, err}
}

type TunneledSshClient struct {
	ProxySshClient  *ssh.Client
	TunneledTcpConn net.Conn
	TunneledSshConn ssh.Conn
	SshClient       *ssh.Client
}

func (tsc *TunneledSshClient) Close() {
	if tsc.SshClient != nil {
		tsc.SshClient.Close()
	}
	if tsc.TunneledSshConn != nil {
		tsc.TunneledSshConn.Close()
	}
	if tsc.TunneledTcpConn != nil {
		tsc.TunneledTcpConn.Close()
	}
	if tsc.ProxySshClient != nil {
		tsc.ProxySshClient.Close()
	}
}

func NewTunneledSshClient(sshConfig *SshConfigDef, ipAddress string) (*TunneledSshClient, error) {
	bastionSshClientConfig, err := NewSshClientConfig(
		sshConfig.User,
		sshConfig.BastionIpAddress,
		sshConfig.Port,
		sshConfig.PrivateKeyPath,
		sshConfig.PrivateKeyPassword)
	if err != nil {
		return nil, err
	}

	bastionUrl := fmt.Sprintf("%s:%d", sshConfig.BastionIpAddress, sshConfig.Port)

	tsc := TunneledSshClient{}

	if ipAddress == sshConfig.BastionIpAddress {
		// Go directly to bastion
		tsc.SshClient, err = ssh.Dial("tcp", bastionUrl, bastionSshClientConfig)
		if err != nil {
			return nil, fmt.Errorf("dial direct to bastion %s failed: %s", bastionUrl, err.Error())
		}
	} else {
		// Dial twice
		tsc.ProxySshClient, err = ssh.Dial("tcp", bastionUrl, bastionSshClientConfig)
		if err != nil {
			return nil, fmt.Errorf("dial to bastion proxy %s failed: %s", bastionUrl, err.Error())
		}

		internalUrl := fmt.Sprintf("%s:%d", ipAddress, sshConfig.Port)

		tsc.TunneledTcpConn, err = tsc.ProxySshClient.Dial("tcp", internalUrl)
		if err != nil {
			return nil, fmt.Errorf("dial to internal URL %s failed: %s", internalUrl, err.Error())
		}

		tunneledSshClientConfig, err := NewSshClientConfig(
			sshConfig.User,
			ipAddress,
			sshConfig.Port,
			sshConfig.PrivateKeyPath,
			sshConfig.PrivateKeyPassword)
		if err != nil {
			return nil, err
		}
		var chans <-chan ssh.NewChannel
		var reqs <-chan *ssh.Request
		tsc.TunneledSshConn, chans, reqs, err = ssh.NewClientConn(tsc.TunneledTcpConn, internalUrl, tunneledSshClientConfig)
		if err != nil {
			return nil, fmt.Errorf("cannot establish ssh connection via TCP tunnel to internal URL %s: %s", internalUrl, err.Error())
		}

		tsc.SshClient = ssh.NewClient(tsc.TunneledSshConn, chans, reqs)
	}

	return &tsc, nil
}

func ExecSsh(prj *Project, logBuilder *strings.Builder, ipAddress string, cmd string) ExecResult {
	tsc, err := NewTunneledSshClient(prj.SshConfig, ipAddress)
	if err != nil {
		return ExecResult{cmd, "", "", 0, err}
	}
	defer tsc.Close()

	session, err := tsc.SshClient.NewSession()
	if err != nil {
		return ExecResult{cmd, "", "", 0, fmt.Errorf("cannot create session for %s: %s", ipAddress, err.Error())}
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	runStartTime := time.Now()
	err = session.Run(cmd)
	elapsed := time.Since(runStartTime).Seconds()

	er := ExecResult{cmd, string(stdout.Bytes()), string(stderr.Bytes()), elapsed, err}
	logBuilder.WriteString(er.ToString())
	return er
}

func ExecSshAndReturnLastLine(prj *Project, logBuilder *strings.Builder, ipAddress string, cmd string) (string, ExecResult) {
	er := ExecSsh(prj, logBuilder, ipAddress, cmd)
	if er.Error != nil {
		return "", er
	}

	lines := strings.Split(strings.Trim(er.Stdout, "\n "), "\n")
	return strings.TrimSpace(lines[len(lines)-1]), er
}

func ExecScriptsOnInstance(prj *Project, logChan chan string, ipAddress string, env map[string]string, shellScriptFiles []string) error {
	sb := strings.Builder{}
	defer func() {
		logChan <- CmdChainExecToString(fmt.Sprintf("ExecScriptOnInstance: %s on %s", shellScriptFiles, ipAddress), sb.String())
	}()

	if len(shellScriptFiles) == 0 {
		sb.WriteString(fmt.Sprintf("no commands to execute on %s", ipAddress))
		return nil
	}

	for _, shellScriptFile := range shellScriptFiles {
		cmdBuilder := strings.Builder{}
		for k, v := range env {
			cmdBuilder.WriteString(fmt.Sprintf("%s=%s\n", k, v))
		}

		f, err := os.Open(shellScriptFile)
		if err != nil {
			return fmt.Errorf("cannot open shell script %s: %s", shellScriptFile, err.Error())
		}
		defer f.Close()

		shellScriptBytes, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("cannot read shell script %s: %s", shellScriptFile, err.Error())
		}

		cmdBuilder.WriteString(string(shellScriptBytes))

		er := ExecSsh(prj, &sb, ipAddress, cmdBuilder.String())
		if er.Error != nil {
			return er.Error
		}
	}
	return nil
}
