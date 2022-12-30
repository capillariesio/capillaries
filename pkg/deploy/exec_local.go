package deploy

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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
		title = fmt.Sprintf("%s: %s", title, err)
	} else {
		title = fmt.Sprintf("%s: OK", title)
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
	} else {
		return title
	}
}

func ExecLocal(prj *Project, cmdPath string, params []string) ExecResult {
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
		return ExecResult{rawInput, rawOutput, rawErrors, elapsed, err}
	} else if cmdCtx.Err() == context.DeadlineExceeded {
		// Timeout occurred, err.Error() is probably: 'signal: killed'
		return ExecResult{rawInput, rawOutput, rawErrors, elapsed, fmt.Errorf("cmd execution timeout exceeded")}
	} else {
		return ExecResult{rawInput, rawOutput, rawErrors, elapsed, nil}
	}
}
