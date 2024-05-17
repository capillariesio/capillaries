package py_calc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/sethvargo/go-envconfig"
	"github.com/shopspring/decimal"
)

const ProcessorPyCalcName string = "py_calc"

// Separate struct to hold values from EnvConfig
type PyCalcEnvSettings struct {
	// Windows: `python` or `C:\Users\%USERNAME%\AppData\Local\Programs\Python\Python310\python.exe`
	// WSL: `python` or `/mnt/c/Users/myusername/AppData/Local/Programs/Python/Python310/python.exe`
	// Linux: `python`
	InterpreterPath string `json:"python_interpreter_path" env:"CAPI_PYCALC_INTERPRETER_PATH, overwrite"`
	// Usually: ["-u", "-"]. -u is essential: without it, we will not see stdout/stderr in the timeout scenario
	InterpreterParams []string `json:"python_interpreter_params" env:"CAPI_PYCALC_INTERPRETER_PARAMS, overwrite"`
	ExecutionTimeout  int      `json:"execution_timeout" env:"CAPI_PYCALC_EXECUTION_TIMEOUT, overwrite"` // Default 5000 milliseconds
}

// All processor settings, root values coming from node
type PyCalcProcessorDef struct {
	PythonUrls                    []string                          `json:"python_code_urls"`
	CalculatedFields              map[string]*sc.WriteTableFieldDef `json:"calculated_fields"`
	UsedInTargetExpressionsFields sc.FieldRefs
	PythonCode                    string
	CalculationOrder              []string
	EnvSettings                   PyCalcEnvSettings
}

func (procDef *PyCalcProcessorDef) GetFieldRefs() *sc.FieldRefs {
	fieldRefs := make(sc.FieldRefs, len(procDef.CalculatedFields))
	i := 0
	for fieldName, fieldDef := range procDef.CalculatedFields {
		fieldRefs[i] = sc.FieldRef{
			TableName: sc.CustomProcessorAlias,
			FieldName: fieldName,
			FieldType: fieldDef.Type}
		i++
	}
	return &fieldRefs
}

func harvestCallExp(callExp *ast.CallExpr, sigMap map[string]struct{}) error {
	// This expression is a python fuction call.
	// Build a func_name(arg,arg,...) signature for it to check our Python code later
	funIdentExp, ok := callExp.Fun.(*ast.Ident)
	if !ok {
		return fmt.Errorf("cannot cast to ident in harvestCallExp")
	}
	funcSig := fmt.Sprintf("%s(%s)", funIdentExp.Name, strings.Trim(strings.Repeat("arg,", len(callExp.Args)), ","))
	sigMap[funcSig] = struct{}{}

	for _, argExp := range callExp.Args {
		callExp, ok := argExp.(*ast.CallExpr)
		if ok {
			if err := harvestCallExp(callExp, sigMap); err != nil {
				return err
			}
		}
	}

	return nil
}

func (procDef *PyCalcProcessorDef) Deserialize(raw json.RawMessage, customProcSettings json.RawMessage, caPath string, privateKeys map[string]string) error {
	var err error
	if err = json.Unmarshal(raw, procDef); err != nil {
		return fmt.Errorf("cannot unmarshal py_calc processor def: %s", err.Error())
	}

	if err = json.Unmarshal(customProcSettings, &procDef.EnvSettings); err != nil {
		return fmt.Errorf("cannot unmarshal py_calc processor env settings: %s", err.Error())
	}

	if err := envconfig.Process(context.TODO(), &procDef.EnvSettings); err != nil {
		return fmt.Errorf("cannot process pycalc env variables: %s", err.Error())
	}

	if len(procDef.EnvSettings.InterpreterPath) == 0 {
		return fmt.Errorf("py_calc interpreter path cannot be empty")
	}

	if procDef.EnvSettings.ExecutionTimeout == 0 {
		procDef.EnvSettings.ExecutionTimeout = 5000
	}

	errors := make([]string, 0)
	usedPythonFunctionSignatures := map[string]struct{}{}

	// Calculated fields
	for _, fieldDef := range procDef.CalculatedFields {

		// Use relaxed Go parser for Python - we are lucky that Go designers liked Python, so we do not have to implement a separate Python partser (for now)
		if fieldDef.ParsedExpression, err = sc.ParseRawRelaxedGolangExpressionStringAndHarvestFieldRefs(fieldDef.RawExpression, &fieldDef.UsedFields, sc.FieldRefAllowUnknownIdents); err != nil {
			errors = append(errors, fmt.Sprintf("cannot parse field expression [%s]: [%s]", fieldDef.RawExpression, err.Error()))
		} else if !sc.IsValidFieldType(fieldDef.Type) {
			errors = append(errors, fmt.Sprintf("invalid field type [%s]", fieldDef.Type))
		}

		// Each calculated field expression must be a valid Python expression and either:
		// 1. some_python_func(<reader fields and other calculated fields>)
		// 2. some reader field, like r.order_id (will be checked by checkFieldUsageInCustomProcessor)
		// 3. some field calculated by this processor, like p.calculatedMargin  (will be checked by checkFieldUsageInCustomProcessor)

		// Check top-level expression
		switch typedExp := fieldDef.ParsedExpression.(type) {
		case *ast.CallExpr:
			if err := harvestCallExp(typedExp, usedPythonFunctionSignatures); err != nil {
				errors = append(errors, fmt.Sprintf("cannot harvest Python call expressions in %s: %s", fieldDef.RawExpression, err.Error()))
			}
		case *ast.SelectorExpr:
			// Assume it's a reader or calculated field. Do not check it here, checkFieldUsageInCustomProcessor() will do that
		default:
			errors = append(errors, fmt.Sprintf("invalid calculated field expression '%s', expected either 'some_function_from_your_python_code(...)' or some reader field, like 'r.order_id', or some other calculated (by this processor) field, like 'p.calculatedMargin'", fieldDef.RawExpression))
		}
	}

	procDef.UsedInTargetExpressionsFields = sc.GetFieldRefsUsedInAllTargetExpressions(procDef.CalculatedFields)

	// Python files
	var b strings.Builder
	procDef.PythonCode = ""
	for _, url := range procDef.PythonUrls {
		bytes, err := xfer.GetFileBytes(url, caPath, privateKeys)
		if err != nil {
			errors = append(errors, err.Error())
		}
		b.WriteString(string(bytes))
		b.WriteString("\n")
	}

	procDef.PythonCode = b.String()

	if errCheckDefs := checkPythonFuncDefAvailability(usedPythonFunctionSignatures, procDef.PythonCode); errCheckDefs != nil {
		errors = append(errors, errCheckDefs.Error())
	}

	// Build a set of "r.inputFieldX" and "p.calculatedFieldY" to perform Python code checks
	srcVarsSet := map[string]struct{}{}
	for _, fieldRef := range *procDef.GetFieldRefs() {
		srcVarsSet[fieldRef.GetAliasHash()] = struct{}{}
	}

	// Define calculation sequence
	// Populate DAG
	dag := map[string][]string{}
	for tgtFieldName, tgtFieldDef := range procDef.CalculatedFields {
		tgtFieldAlias := fmt.Sprintf("p.%s", tgtFieldName)
		dag[tgtFieldAlias] = make([]string, len(tgtFieldDef.UsedFields))
		for i, usedFieldRef := range tgtFieldDef.UsedFields {
			dag[tgtFieldAlias][i] = usedFieldRef.GetAliasHash()
		}
	}

	// Check DAG and return calc fields in the order they should be calculated
	if procDef.CalculationOrder, err = kahn(dag); err != nil {
		errors = append(errors, fmt.Sprintf("%s. Calc dependency map:\n%v", err, dag))
	}

	// TODO: deserialize other stuff from raw here if needed

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

func (procDef *PyCalcProcessorDef) GetUsedInTargetExpressionsFields() *sc.FieldRefs {
	return &procDef.UsedInTargetExpressionsFields
}

// Python supports microseconds in datetime. Unfortunately, Cassandra supports only milliseconds. Millis are our lingua franca.
// So, use only three digits after decimal point
// Python 8601 requires ":" in the timezone
const PythonDatetimeFormat string = "2006-01-02T15:04:05.000-07:00"

func valueToPythonExpr(val any) string {
	switch typedVal := val.(type) {
	case int64:
		return fmt.Sprintf("%d", typedVal)
	case float64:
		return fmt.Sprintf("%f", typedVal)
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(typedVal, "'", `\'`)) // Use single commas in Python - this may go to logs
	case bool:
		if typedVal {
			return "TRUE"
		} else {
			return "FALSE"
		}
	case decimal.Decimal:
		return typedVal.String()
	case time.Time:
		return typedVal.Format(fmt.Sprintf("\"%s\"", PythonDatetimeFormat))
	default:
		return fmt.Sprintf("cannot convert '%v(%T)' to Python expression", typedVal, typedVal)
	}
}

func pythonResultToRowsetValue(fieldRef *sc.FieldRef, fieldValue any) (any, error) {
	switch fieldRef.FieldType {
	case sc.FieldTypeString:
		finalVal, ok := fieldValue.(string)
		if !ok {
			return nil, fmt.Errorf("string %s, unexpected type %T(%v)", fieldRef.FieldName, fieldValue, fieldValue)
		}
		return finalVal, nil
	case sc.FieldTypeBool:
		finalVal, ok := fieldValue.(bool)
		if !ok {
			return nil, fmt.Errorf("bool %s, unexpected type %T(%v)", fieldRef.FieldName, fieldValue, fieldValue)
		}
		return finalVal, nil
	case sc.FieldTypeInt:
		finalVal, ok := fieldValue.(float64)
		if !ok {
			return nil, fmt.Errorf("int %s, unexpected type %T(%v)", fieldRef.FieldName, fieldValue, fieldValue)
		}
		finalIntVal := int64(finalVal)
		return finalIntVal, nil
	case sc.FieldTypeFloat:
		finalVal, ok := fieldValue.(float64)
		if !ok {
			return nil, fmt.Errorf("float %s, unexpected type %T(%v)", fieldRef.FieldName, fieldValue, fieldValue)
		}
		return finalVal, nil
	case sc.FieldTypeDecimal2:
		finalVal, ok := fieldValue.(float64)
		if !ok {
			return nil, fmt.Errorf("decimal %s, unexpected type %T(%v)", fieldRef.FieldName, fieldValue, fieldValue)
		}
		finalDecVal := decimal.NewFromFloat(finalVal).Round(2)
		return finalDecVal, nil
	case sc.FieldTypeDateTime:
		finalVal, ok := fieldValue.(string)
		if !ok {
			return nil, fmt.Errorf("time %s, unexpected type %T(%v)", fieldRef.FieldName, fieldValue, fieldValue)
		}
		timeVal, err := time.Parse(PythonDatetimeFormat, finalVal)
		if err != nil {
			return nil, fmt.Errorf("bad time result %s, unexpected format %s", fieldRef.FieldName, finalVal)
		}
		return timeVal, nil
	default:
		return nil, fmt.Errorf("unexpected field type %s, %s, %T(%v)", fieldRef.FieldType, fieldRef.FieldName, fieldValue, fieldValue)
	}
}

/*
CheckDefsAvailability - makes sure all expressions mentioned in calc expressions have correspondent Python functions defined.
Pays attention to function name and number of arguments. Expressions may contain:
- calculated field references like p.orderMargin
- number/string/bool constants
- calls to other Python functions
Does NOT perform deep checks for Python function call hierarchy.
*/
func checkPythonFuncDefAvailability(usedPythonFunctionSignatures map[string]struct{}, codeFormulaDefs string) error {
	var errors strings.Builder

	// Walk trhough the whole code and collect Python function defs
	availableDefSigs := map[string]struct{}{}
	reDefSig := regexp.MustCompile(`(?m)^def[ ]+([a-zA-Z0-9_]+)[ ]*([ \(\)a-zA-Z0-9_,\.\"\'\t]+)[ ]*:`)
	reArgCommas := regexp.MustCompile("[^)(,]+")

	// Find "def someFunc123(someParam, 'some literal'):"
	defSigMatches := reDefSig.FindAllStringSubmatch(codeFormulaDefs, -1)
	for _, sigMatch := range defSigMatches {
		if strings.ReplaceAll(sigMatch[2], " ", "") == "()" {
			// This Python function definition does not accept arguments
			availableDefSigs[fmt.Sprintf("%s()", sigMatch[1])] = struct{}{}
		} else {
			// Strip all characters from the arg list, except commas and parenthesis,
			// this will give us the canonical signature with the number of arguments presented by commas: "sum(arg,arg,arg)""
			canonicalSig := fmt.Sprintf("%s%s", sigMatch[1], reArgCommas.ReplaceAllString(sigMatch[2], "arg"))
			availableDefSigs[canonicalSig] = struct{}{}
		}
	}

	// A curated list of Python functions allowed in script expressions. No exec() please.
	pythoBuiltinFunctions := map[string]struct{}{
		"str(arg)":   {},
		"int(arg)":   {},
		"float(arg)": {},
		"round(arg)": {},
		"len(arg)":   {},
		"bool(arg)":  {},
		"abs(arg)":   {},
	}

	// Walk through all Python signatures in calc expressions (we harvested them in Deserialize)
	// and make sure correspondent python function defs are available
	for usedSig := range usedPythonFunctionSignatures {
		if _, ok := availableDefSigs[usedSig]; !ok {
			if _, ok := pythoBuiltinFunctions[usedSig]; !ok {
				errors.WriteString(fmt.Sprintf("function def '%s' not found in Python file, and it's not in the list of allowed Python built-in functions; ", usedSig))
			}
		}
	}

	if errors.Len() > 0 {
		var defs strings.Builder
		for defSig := range availableDefSigs {
			defs.WriteString(fmt.Sprintf("%s; ", defSig))
		}
		return fmt.Errorf("Python function defs availability check failed, the following functions are not defined: [%s]. Full list of available Python function definitions: %s", errors.String(), defs.String())
	}

	return nil
}

func kahn(depMap map[string][]string) ([]string, error) {
	inDegreeMap := make(map[string]int)
	for node := range depMap {
		if depMap[node] != nil {
			for _, v := range depMap[node] {
				inDegreeMap[v]++
			}
		}
	}

	var queue []string
	for node := range depMap {
		if _, ok := inDegreeMap[node]; !ok {
			queue = append(queue, node)
		}
	}

	var execOrder []string
	for len(queue) > 0 {
		node := queue[len(queue)-1]
		queue = queue[:(len(queue) - 1)]
		// Prepend "node" to execOrder only if it's the key in depMap
		// (this is not one of the algorithm requirements,
		// it's just our caller needs only those elements)
		if _, ok := depMap[node]; ok {
			execOrder = append(execOrder, "")
			copy(execOrder[1:], execOrder)
			execOrder[0] = node
		}
		for _, v := range depMap[node] {
			inDegreeMap[v]--
			if inDegreeMap[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	for _, inDegree := range inDegreeMap {
		if inDegree > 0 {
			return []string{}, fmt.Errorf("Formula expressions have cycle(s)")
		}
	}

	return execOrder, nil
}

func (procDef *PyCalcProcessorDef) buildPythonCodebaseFromRowset(rsIn *proc.Rowset) (string, error) {
	// Build a massive Python source: function defs, dp init, calculation, result json
	// This is the hardcoded Python code structure we rely on. Do not change it.
	var codeBase strings.Builder

	codeBase.WriteString(fmt.Sprintf(`
import traceback
import json
print("\n%s") # Provide function defs
%s
`,
		FORMULA_MARKER_FUNCTION_DEFINITIONS,
		procDef.PythonCode))

	for rowIdx := 0; rowIdx < rsIn.RowCount; rowIdx++ {
		itemCalculationCodebase, err := procDef.printItemCalculationCode(rowIdx, rsIn)
		if err != nil {
			return "", err
		}
		codeBase.WriteString(fmt.Sprintf("%s\n", itemCalculationCodebase))
	}

	return codeBase.String(), nil
}

func (procDef *PyCalcProcessorDef) analyseExecError(codeBase string, rawOutput string, rawErrors string, err error) (string, error) {
	// Linux: err.Error():'exec: "python333": executable file not found in $PATH'
	// Windows: err.Error():'exec: "C:\\Program Files\\Python3badpath\\python.exe": file does not exist'
	// MacOS/WSL:  err.Error():'fork/exec /mnt/c/Users/myusername/AppData/Local/Programs/Python/Python310/python.exe: no such file or directory'
	if strings.Contains(err.Error(), "file not found") ||
		strings.Contains(err.Error(), "file does not exist") ||
		strings.Contains(err.Error(), "no such file or directory") {
		return "", fmt.Errorf("interpreter binary not found: %s", procDef.EnvSettings.InterpreterPath)
	} else if strings.Contains(err.Error(), "exit status") {
		// err.Error():'exit status 1', cmdCtx.Err():'%!s(<nil>)'
		// Python interpreter reported an error: there was a syntax error in the codebase and no results were returned
		fullErrorInfo := fmt.Sprintf("interpreter returned an error (probably syntax):\n%s %v\n%s\n%s\n%s",
			procDef.EnvSettings.InterpreterPath, procDef.EnvSettings.InterpreterParams,
			rawOutput,
			rawErrors,
			getErrorLineNumberInfo(&codeBase, rawErrors))

		// fmt.Println(fullErrorInfo)
		return fullErrorInfo, fmt.Errorf("interpreter returned an error (probably syntax), see log for details: %s", rawErrors)
	}
	return "", fmt.Errorf("unexpected calculation errors: %s", rawErrors)
}

const pyCalcFlushBufferSize int = 1000

func getRowResultJson(rowIdx int, codeBase *string, rawOutput *string, sectionEndPos int) (string, string, error) {
	startMarker := fmt.Sprintf("%s:%d", FORMULA_MARKER_DATA_POINTS_INITIALIZATION, rowIdx)
	endMarker := fmt.Sprintf("%s:%d", FORMULA_MARKER_END, rowIdx)
	relSectionStartPos := strings.Index((*rawOutput)[sectionEndPos:], startMarker)
	relSectionEndPos := strings.Index((*rawOutput)[sectionEndPos:], endMarker)
	sectionStartPos := sectionEndPos + relSectionStartPos
	if sectionStartPos == -1 {
		return "", "", fmt.Errorf("%d: unexpected, cannot find calculation start marker %s;", rowIdx, startMarker)
	}
	sectionEndPos = sectionEndPos + relSectionEndPos
	if sectionEndPos == -1 {
		return "", "", fmt.Errorf("%d: unexpected, cannot find calculation end marker %s;", rowIdx, endMarker)
	}
	if sectionStartPos > sectionEndPos {
		return "", "", fmt.Errorf("%d: unexpected, end marker %s(%d) is earlier than start marker %s(%d);", rowIdx, endMarker, sectionStartPos, endMarker, sectionEndPos)
	}

	rawSectionOutput := (*rawOutput)[sectionStartPos:sectionEndPos]
	successMarker := fmt.Sprintf("%s:%d", FORMULA_MARKER_SUCCESS, rowIdx)
	sectionSuccessPos := strings.Index(rawSectionOutput, successMarker)
	if sectionSuccessPos == -1 {
		// There was an error calculating fields for this item
		// Assume the last line of the out is the error
		errorLines := strings.Split(rawSectionOutput, "\n")
		errorText := ""
		for i := len(errorLines) - 1; i >= 0; i-- {
			errorText = strings.Trim(errorLines[i], "\r \t")
			if len(errorText) > 0 {
				break
			}
		}
		errorText = fmt.Sprintf("%d:cannot calculate data points;%s; ", rowIdx, errorText)
		// errors.WriteString(errorText)
		return "", fmt.Sprintf("%s\n%s", errorText, getErrorLineNumberInfo(codeBase, rawSectionOutput)), nil
	}

	// SUCCESS code snippet is there, will try to parse the result JSON
	return rawSectionOutput[sectionSuccessPos+len(successMarker):], "", nil
}

func (procDef *PyCalcProcessorDef) analyseExecSuccess(codeBase string, rawOutput string, _ string, outFieldRefs *sc.FieldRefs, rsIn *proc.Rowset, flushVarsArray func(varsArray []*eval.VarValuesMap, varsArrayCount int) error) error {
	// No Python interpreter errors, but there may be runtime errors and good results.
	// Timeout error may be there too.

	var errors strings.Builder
	varsArray := make([]*eval.VarValuesMap, pyCalcFlushBufferSize)
	varsArrayCount := 0

	sectionEndPos := 0
	for rowIdx := 0; rowIdx < rsIn.RowCount; rowIdx++ {
		jsonString, rowError, fatalError := getRowResultJson(rowIdx, &codeBase, &rawOutput, sectionEndPos)
		if fatalError != nil {
			return fatalError
		}
		if len(rowError) > 0 {
			errors.WriteString(rowError)
		} else {
			// SUCCESS code snippet jsonString is there, try to get the result JSON
			var itemResults map[string]any
			err := json.Unmarshal([]byte(jsonString), &itemResults)
			if err != nil {
				// Bad JSON
				errorText := fmt.Sprintf("%d:unexpected error, cannot deserialize results, %s, '%s'", rowIdx, err, jsonString)
				errors.WriteString(errorText)
				// logText.WriteString(errorText)
			} else {
				// Success

				// We need to include reader fieldsin the result, writermay use any of them
				vars := eval.VarValuesMap{}
				if err := rsIn.ExportToVars(rowIdx, &vars); err != nil {
					return err
				}

				vars[sc.CustomProcessorAlias] = map[string]any{}

				for _, outFieldRef := range *outFieldRefs {
					pythonFieldValue, ok := itemResults[outFieldRef.FieldName]
					if !ok {
						errors.WriteString(fmt.Sprintf("cannot find result for row %d, field %s;", rowIdx, outFieldRef.FieldName))
					} else {
						valVolatile, err := pythonResultToRowsetValue(&outFieldRef, pythonFieldValue)
						if err != nil {
							errors.WriteString(fmt.Sprintf("cannot deserialize result for row %d: %s;", rowIdx, err.Error()))
						} else {
							vars[sc.CustomProcessorAlias][outFieldRef.FieldName] = valVolatile
						}
					}
				}

				if errors.Len() == 0 {
					varsArray[varsArrayCount] = &vars
					varsArrayCount++
					if varsArrayCount == len(varsArray) {
						if err = flushVarsArray(varsArray, varsArrayCount); err != nil {
							return fmt.Errorf("error flushing vars array of size %d: %s", varsArrayCount, err.Error())
						}
						varsArray = make([]*eval.VarValuesMap, pyCalcFlushBufferSize)
						varsArrayCount = 0
					}
				}
			}
		}
	}

	if errors.Len() > 0 {
		// fmt.Println(fmt.Sprintf("%s\nRaw output below:\n%s\nFull codebase below (may be big):\n%s", logText.String(), rawOutput, codeBase.String()))
		return fmt.Errorf(errors.String())
	}

	// fmt.Println(fmt.Sprintf("%s\nRaw output below:\n%s", logText.String(), rawOutput))
	if varsArrayCount > 0 {
		if err := flushVarsArray(varsArray, varsArrayCount); err != nil {
			return fmt.Errorf("error flushing leftovers vars array of size %d: %s", varsArrayCount, err.Error())
		}
	}
	return nil
}

/*
getErrorLineNumbers - shows error lines +/- 5 if error info found in the output
*/
func getErrorLineNumberInfo(codeBase *string, rawErrors string) string {
	var errorLineNumberInfo strings.Builder

	reErrLine := regexp.MustCompile(`File "<stdin>", line ([\d]+)`)
	groupMatches := reErrLine.FindAllStringSubmatch(rawErrors, -1)
	if len(groupMatches) > 0 {
		for matchIdx := 0; matchIdx < len(groupMatches); matchIdx++ {
			errLineNum, errAtoi := strconv.Atoi(groupMatches[matchIdx][1])
			if errAtoi != nil {
				errorLineNumberInfo.WriteString(fmt.Sprintf("Unexpected error, cannot parse error line number (%s): %s", groupMatches[matchIdx][1], errAtoi))
			} else {
				errorLineNumberInfo.WriteString(fmt.Sprintf("Source code lines close to the error location (line %d):\n", errLineNum))
				scanner := bufio.NewScanner(strings.NewReader(*codeBase))
				lineNum := 1
				for scanner.Scan() {
					if lineNum+15 >= errLineNum && lineNum-15 <= errLineNum {
						errorLineNumberInfo.WriteString(fmt.Sprintf("%06d    %s\n", lineNum, scanner.Text()))
					}
					lineNum++
				}
			}
		}
	} else {
		errorLineNumberInfo.WriteString(fmt.Sprintf("Unexpected error, cannot find error line number in raw error output %s", rawErrors))
	}

	return errorLineNumberInfo.String()
}

const FORMULA_MARKER_FUNCTION_DEFINITIONS = "--FMDEF"
const FORMULA_MARKER_DATA_POINTS_INITIALIZATION = "--FMINIT"
const FORMULA_MARKER_CALCULATIONS = "--FMCALC"
const FORMULA_MARKER_SUCCESS = "--FMOK"
const FORMULA_MARKER_END = "--FMEND"

const ReaderPrefix string = "r_"
const ProcPrefix string = "p_"

func (procDef *PyCalcProcessorDef) printItemCalculationCode(rowIdx int, rsIn *proc.Rowset) (string, error) {
	// Initialize input variables in no particular order
	vars := eval.VarValuesMap{}
	err := rsIn.ExportToVars(rowIdx, &vars)
	if err != nil {
		return "", err
	}
	var bIn strings.Builder
	for fieldName, fieldVal := range vars[sc.ReaderAlias] {
		bIn.WriteString(fmt.Sprintf("  %s%s = %s\n", ReaderPrefix, fieldName, valueToPythonExpr(fieldVal)))
	}

	// Calculation expression order matters (we got it from DAG analysis), so follow it
	// for calc data points. Also follow it for results JSON (although this is not important)
	var bCalc strings.Builder
	var bRes strings.Builder
	prefixRemover := strings.NewReplacer(fmt.Sprintf("%s.", sc.CustomProcessorAlias), "")
	prefixReplacer := strings.NewReplacer(fmt.Sprintf("%s.", sc.ReaderAlias), ReaderPrefix, fmt.Sprintf("%s.", sc.CustomProcessorAlias), ProcPrefix)
	for fieldIdx, procFieldWithAlias := range procDef.CalculationOrder {
		procField := prefixRemover.Replace(procFieldWithAlias)
		bCalc.WriteString(fmt.Sprintf("  %s%s = %s\n", ProcPrefix, procField, prefixReplacer.Replace(procDef.CalculatedFields[procField].RawExpression)))
		bRes.WriteString(fmt.Sprintf("  \"%s\":%s%s", procField, ProcPrefix, procField))
		if fieldIdx < len(procDef.CalculationOrder)-1 {
			bRes.WriteString(",")
		}
	}

	const codeBaseSkeleton = `
print('')
print('%s:%d')
try:
%s
  print('%s:%d')
%s
  print('%s:%d')
  print(json.dumps({%s}))
except:
  print(traceback.format_exc()) 
print('%s:%d')
`
	return fmt.Sprintf(codeBaseSkeleton,
		FORMULA_MARKER_DATA_POINTS_INITIALIZATION, rowIdx,
		bIn.String(),
		FORMULA_MARKER_CALCULATIONS, rowIdx,
		bCalc.String(),
		FORMULA_MARKER_SUCCESS, rowIdx,
		bRes.String(),
		FORMULA_MARKER_END, rowIdx), nil
}
