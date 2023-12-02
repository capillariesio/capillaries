package deploy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type ExecResult struct {
	Cmd     string
	Stdout  string
	Stderr  string
	Elapsed float64
	Error   error
}

func (er *ExecResult) ToString() string {
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

func CmdChainExecToString(title string, logContent string, err error, isVerbose bool) string {
	if err != nil {
		title = fmt.Sprintf("%s: %s%s%s", title, LogColorRed, err, LogColorReset)
	} else {
		title = fmt.Sprintf("%s: %sOK%s", title, LogColorGreen, LogColorReset)
	}

	if isVerbose {
		return fmt.Sprintf(
			`
=========================================
%s
=========================================
%s
=========================================
`, title, logContent)
	}

	return title
}

func ExecLocal(prj *Project, cmdPath string, params []string, envVars map[string]string, dir string) ExecResult {
	// Protect it with a timeout
	cmdCtx, cancel := context.WithTimeout(context.Background(), time.Duration(prj.Timeouts.OpenstackCmd*int(time.Second)))
	defer cancel()

	p := exec.CommandContext(cmdCtx, cmdPath, params...)

	if dir != "" {
		p.Dir = dir
	}

	for k, v := range envVars {
		p.Env = append(p.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Inherit $HOME
	if _, ok := envVars["HOME"]; !ok {
		p.Env = append(p.Env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))
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
	rawOutput := stdout.String()
	rawErrors := stderr.String()
	if err != nil {
		// Cmd not found, nonzero exit status etc
		return ExecResult{rawInput, rawOutput, rawErrors, elapsed, err}
	} else if cmdCtx.Err() == context.DeadlineExceeded {
		// Timeout occurred, err.Error() is probably: 'signal: killed'
		return ExecResult{rawInput, rawOutput, rawErrors, elapsed, fmt.Errorf("cmd execution timeout exceeded")}
	}

	return ExecResult{rawInput, rawOutput, rawErrors, elapsed, nil}
}

func BuildArtifacts(prjPair *ProjectPair, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("BuildArtifacts", isVerbose)
	curDir, err := os.Getwd()
	if err != nil {
		return lb.Complete(err)
	}
	for _, cmd := range prjPair.Live.Artifacts.Cmd {
		fullCmdPath, err := filepath.Abs(path.Join(curDir, cmd))
		if err != nil {
			return lb.Complete(err)
		}
		cmdDir, cmdFileName := filepath.Split(fullCmdPath)
		er := ExecLocal(&prjPair.Live, "./"+cmdFileName, []string{}, prjPair.Live.Artifacts.Env, cmdDir)
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
	}
	return lb.Complete(nil)
}
