package deploy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/xfer"
	"golang.org/x/crypto/ssh"
)

// func (sshClient *xfer.SshClient) RunCommand(cmd string) ExecResult {
// 	// open connection
// 	conn, err := ssh.Dial("tcp", sshClient.Server, sshClient.Config)
// 	if err != nil {
// 		return ExecResult{cmd, "", "", 0, fmt.Errorf("Dial to %v failed %v", sshClient.Server, err)}
// 	}
// 	defer conn.Close()

// 	// open session
// 	session, err := conn.NewSession()
// 	if err != nil {
// 		return ExecResult{cmd, "", "", 0, fmt.Errorf("Create session for %v failed %v", sshClient.Server, err)}
// 	}
// 	defer session.Close()

// 	var stdout, stderr bytes.Buffer
// 	session.Stdout = &stdout
// 	session.Stderr = &stderr

// 	runStartTime := time.Now()
// 	err = session.Run(cmd)
// 	elapsed := time.Since(runStartTime).Seconds()

// 	return ExecResult{cmd, string(stdout.Bytes()), string(stderr.Bytes()), elapsed, err}
// }

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
	bastionSshClientConfig, err := xfer.NewSshClientConfig(
		sshConfig.User,
		sshConfig.PrivateKeyPath,
		sshConfig.PrivateKeyPassword)
	if err != nil {
		return nil, err
	}

	bastionUrl := fmt.Sprintf("%s:%d", sshConfig.ExternalIpAddress, sshConfig.Port)

	tsc := TunneledSshClient{}

	if ipAddress == sshConfig.ExternalIpAddress {
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

		tunneledSshClientConfig, err := xfer.NewSshClientConfig(
			sshConfig.User,
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

func ExecSsh(sshConfig *SshConfigDef, ipAddress string, cmd string) ExecResult {
	tsc, err := NewTunneledSshClient(sshConfig, ipAddress)
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

	er := ExecResult{cmd, stdout.String(), stderr.String(), elapsed, err}
	return er
}

func ExecSshForClient(sshClient *ssh.Client, cmd string) (string, string, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("cannot create session for %s: %s", sshClient.RemoteAddr(), err.Error())
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run(cmd); err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("cannot execute '%s' at %s: %s (stderr: %s)", sshClient.RemoteAddr(), cmd, err.Error(), stderr.String())
	}
	return stdout.String(), stderr.String(), nil
}

func ExecSshAndReturnLastLine(sshConfig *SshConfigDef, ipAddress string, cmd string) (string, ExecResult) {
	er := ExecSsh(sshConfig, ipAddress, cmd)
	if er.Error != nil {
		return "", er
	}

	lines := strings.Split(strings.Trim(er.Stdout, "\n "), "\n")
	return strings.TrimSpace(lines[len(lines)-1]), er
}

type LogBuilder struct {
	Sb        *strings.Builder
	IsVerbose bool
	Header    string
	StartTs   time.Time
}

type LogMsg string

const LogColorReset string = "\033[0m"
const LogColorRed string = "\033[31m"
const LogColorGreen string = "\033[32m"

func NewLogBuilder(header string, isVerbose bool) *LogBuilder {
	lb := LogBuilder{Sb: &strings.Builder{}, IsVerbose: isVerbose, Header: header, StartTs: time.Now()}
	if lb.IsVerbose {
		lb.Sb.WriteString("\n===============================================\n")
		lb.Sb.WriteString(fmt.Sprintf("%s : started\n", lb.Header))
	} else {
		lb.Sb.WriteString(fmt.Sprintf("%s : ", lb.Header))
	}
	return &lb
}

func AddLogMsg(sb *strings.Builder, logMsg LogMsg) {
	sb.WriteString(string(logMsg))
}

func (lb *LogBuilder) Add(content string) {
	if !lb.IsVerbose {
		return
	}
	lb.Sb.WriteString(fmt.Sprintf("%s\n", content))
}

func (lb *LogBuilder) Complete(err error) (LogMsg, error) {
	if lb.IsVerbose {
		lb.Sb.WriteString(fmt.Sprintf("%s : ", lb.Header))
	}
	lb.Sb.WriteString(fmt.Sprintf("elapsed %.3fs, ", time.Since(lb.StartTs).Seconds()))
	if err == nil {
		lb.Sb.WriteString(LogColorGreen + "OK" + LogColorReset)
	} else {
		lb.Sb.WriteString(LogColorRed + err.Error() + LogColorReset)
	}
	lb.Sb.WriteString("\n")
	return LogMsg(lb.Sb.String()), err
}

func readScriptFile(fullScriptPath string) (string, error) {
	f, err := os.Open(fullScriptPath)
	if err != nil {
		return "", fmt.Errorf("cannot open shell script %s: %s", fullScriptPath, err.Error())
	}
	defer f.Close()

	shellScriptBytes, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("cannot read shell script %s: %s", fullScriptPath, err.Error())
	}
	return string(shellScriptBytes), nil
}

func ExecScriptsOnInstance(sshConfig *SshConfigDef, ipAddress string, env map[string]string, prjFileDirPath string, shellScriptFiles []string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder(fmt.Sprintf("ExecScriptsOnInstance: %s on %s", shellScriptFiles, ipAddress), isVerbose)

	if len(shellScriptFiles) == 0 {
		lb.Add(fmt.Sprintf("no commands to execute on %s", ipAddress))
		return lb.Complete(nil)
	}

	for _, shellScriptFile := range shellScriptFiles {
		cmdBuilder := strings.Builder{}
		for k, v := range env {
			cmdBuilder.WriteString(fmt.Sprintf("%s=%s\n", k, v))
		}

		shellScriptString, err := readScriptFile(filepath.Join(prjFileDirPath, shellScriptFile))
		if err != nil {
			return lb.Complete(err)
		}

		cmdBuilder.WriteString(shellScriptString)

		er := ExecSsh(sshConfig, ipAddress, cmdBuilder.String())
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
	}
	return lb.Complete(nil)
}

func ExecCommandsOnInstance(sshConfig *SshConfigDef, ipAddress string, cmds []string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder(fmt.Sprintf("ExecCommandsOnInstance: %d on %s", len(cmds), ipAddress), isVerbose)
	if len(cmds) == 0 {
		lb.Add(fmt.Sprintf("no commands to execute on %s", ipAddress))
		return lb.Complete(nil)
	}

	for _, cmd := range cmds {
		er := ExecSsh(sshConfig, ipAddress, cmd)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
	}
	return lb.Complete(nil)
}
