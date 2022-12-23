package deploy

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type LocalExecResult struct {
	Cmd     string
	Stdout  string
	Stderr  string
	Elapsed float64
	Error   error
}

func (er *LocalExecResult) ToString() string {
	var errString string
	if er.Error != nil {
		errString = er.Error.Error()
	}
	return fmt.Sprintf(`
-----------------------
cmd:
%s
stdout:
%s
stderr:
%s
error:
%s
elapsed:%0.3f
-----------------------
`, er.Cmd, er.Stdout, er.Stderr, errString, er.Elapsed)
}

func CmdChainExecToString(name string, content string) string {
	return fmt.Sprintf(
		`
=========================================
%s
=========================================
%s
=========================================
`, name, content)
}

func ExecLocal(prj *Project, cmdPath string, params []string) LocalExecResult {
	// Protect it with a timeout
	cmdCtx, cancel := context.WithTimeout(context.Background(), time.Duration(prj.Timeouts.OpenstackCmd*int(time.Second)))
	defer cancel()

	p := exec.CommandContext(cmdCtx, cmdPath, params...)

	for k, v := range prj.OpenstackEnvVariables {
		p.Env = append(p.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Do not use pipes, work with raw data, otherwise stdout/stderr
	// will not be easily available in the timeout scenario
	var stdout, stderr bytes.Buffer
	p.Stdout = &stdout
	p.Stderr = &stderr

	// Run
	runStartTime := time.Now()
	err := p.Run()
	elapsed := time.Since(runStartTime).Seconds()

	rawInput := fmt.Sprintf("%s %s", cmdPath, strings.Join(params, " "))
	rawOutput := string(stdout.Bytes())
	rawErrors := string(stderr.Bytes())
	if err != nil {
		// Cmd not found, nonzero exit status etc
		return LocalExecResult{rawInput, rawOutput, rawErrors, elapsed, err}
	} else if cmdCtx.Err() == context.DeadlineExceeded {
		// Timeout occurred, err.Error() is probably: 'signal: killed'
		return LocalExecResult{rawInput, rawOutput, rawErrors, elapsed, fmt.Errorf("cmd execution timeout exceeded")}
	} else {
		return LocalExecResult{rawInput, rawOutput, rawErrors, elapsed, nil}
	}
}

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
func NewSshClient(user string, host string, port int, privateKeyPath string, privateKeyPassword string) (*SshClient, error) {
	// read private key file
	pemBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Reading private key file failed %v", err)
	}
	// create signer
	signer, err := signerFromPem(pemBytes, []byte(privateKeyPassword))
	if err != nil {
		return nil, err
	}
	// build SSH client config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// use OpenSSH's known_hosts file if you care about host validation
			return nil
		},
	}

	client := &SshClient{
		Config: config,
		Server: fmt.Sprintf("%v:%v", host, port),
	}

	return client, nil
}

func (sshClient *SshClient) RunCommand(cmd string) LocalExecResult {
	// open connection
	conn, err := ssh.Dial("tcp", sshClient.Server, sshClient.Config)
	if err != nil {
		return LocalExecResult{cmd, "", "", 0, fmt.Errorf("Dial to %v failed %v", sshClient.Server, err)}
	}
	defer conn.Close()

	// open session
	session, err := conn.NewSession()
	if err != nil {
		return LocalExecResult{cmd, "", "", 0, fmt.Errorf("Create session for %v failed %v", sshClient.Server, err)}
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	runStartTime := time.Now()
	err = session.Run(cmd)
	elapsed := time.Since(runStartTime).Seconds()

	return LocalExecResult{cmd, string(stdout.Bytes()), string(stderr.Bytes()), elapsed, err}
}

func ExecSsh(prj *Project, logBuilder *strings.Builder, cmd string) LocalExecResult {
	sshClient, err := NewSshClient(
		prj.SshConfig.User,
		prj.SshConfig.Host,
		prj.SshConfig.Port,
		prj.SshConfig.PrivateKeyPath,
		prj.SshConfig.PrivateKeyPassword)
	if err != nil {
		return LocalExecResult{cmd, "", "", 0, err}
	}
	er := sshClient.RunCommand(cmd)
	logBuilder.WriteString(er.ToString())
	return er
}

func ExecSshAndReturnLastLine(prj *Project, logBuilder *strings.Builder, cmd string) (string, LocalExecResult) {
	er := ExecSsh(prj, logBuilder, cmd)
	if er.Error != nil {
		return "", er
	}

	lines := strings.Split(strings.Trim(er.Stdout, "\n "), "\n")
	return strings.TrimSpace(lines[len(lines)-1]), er
}
