package py_calc

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/proc"
)

func (procDef *PyCalcProcessorDef) Run(logger *l.Logger, pCtx *ctx.MessageProcessingContext, rsIn *proc.Rowset, flushVarsArray func(varsArray []*eval.VarValuesMap, varsArrayCount int) error) error {
	logger.PushF("custom.PyCalcProcessorDef.Run")
	defer logger.PopF()

	//err := procDef.executeCalculations(logger, pCtx, rsIn, rsOut, time.Duration(procDef.EnvSettings.ExecutionTimeout*int(time.Millisecond)))

	timeout := time.Duration(procDef.EnvSettings.ExecutionTimeout * int(time.Millisecond))

	codeBase, err := procDef.buildPythonCodebaseFromRowset(rsIn)
	if err != nil {
		return fmt.Errorf("cannot build Python codebase from rowset: %s", err)
	}

	// Protect it with a timeout
	cmdCtx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	p := exec.CommandContext(cmdCtx, procDef.EnvSettings.InterpreterPath, procDef.EnvSettings.InterpreterParams...)

	// Supply our calculation code to Python as stdin
	p.Stdin = strings.NewReader(codeBase)

	// Do not use pipes, work with raw data, otherwise stdout/stderr
	// will not be easily available in the timeout scenario
	var stdout, stderr bytes.Buffer
	p.Stdout = &stdout
	p.Stderr = &stderr

	//return fmt.Errorf(codeBase.String())

	// Run
	pythonStartTime := time.Now()
	err = p.Run()
	pythonDur := time.Since(pythonStartTime)
	logger.InfoCtx(pCtx, "PythonInterpreter: %d items in %v (%.0f items/s)", rsIn.RowCount, pythonDur, float64(rsIn.RowCount)/pythonDur.Seconds())

	rawOutput := stdout.String()
	rawErrors := stderr.String()

	// Really verbose, use for troubleshooting only
	// fmt.Println(codeBase, rawOutput)

	//fmt.Println(fmt.Sprintf("err.Error():'%s', cmdCtx.Err():'%v'", err.Error(), cmdCtx.Err()))

	if err != nil {
		fullErrorInfo, err := procDef.analyseExecError(codeBase, rawOutput, rawErrors, err)
		if len(fullErrorInfo) > 0 {
			logger.ErrorCtx(pCtx, fullErrorInfo)
		}
		return fmt.Errorf("Python interpreter returned an error: %s", err)
	} else {
		if cmdCtx.Err() == context.DeadlineExceeded {
			// Timeout occurred, err.Error() is probably: 'signal: killed'
			return fmt.Errorf("Python calculation timeout %d s expired;", timeout)
		} else {
			return procDef.analyseExecSuccess(codeBase, rawOutput, rawErrors, procDef.GetFieldRefs(), rsIn, flushVarsArray)
		}
	}
}
