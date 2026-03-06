package eval

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// IMPORTANT: please keep this eval core component TableFieldType- and custom function-agnostic.
// It should not be aware of things like decimal2 or some math.iif() functions.

func DetectRootAggFunc(exp ast.Expr) (AggEnabledType, AggFuncType, []ast.Expr) {
	if callExp, ok := exp.(*ast.CallExpr); ok {
		funExp := callExp.Fun
		if funIdentExp, ok := funExp.(*ast.Ident); ok {
			aggFuncType := StringToAggFunc(funIdentExp.Name)
			if aggFuncType != AggUnknown {
				return AggFuncEnabled, aggFuncType, callExp.Args
			}
		}
	}
	return AggFuncDisabled, AggUnknown, nil
}

type AggEnabledType int

const (
	AggFuncDisabled AggEnabledType = iota
	AggFuncEnabled
)

// Custom functions
type EvalFunction func(args []any) (any, error)

// Identifiers used in the calculation. Examples:
// - ""."var1": plain variable var1
// - ""."field1": plain field name field1 (table name given somewhere else implicitly)
// - "table1"."field1": fully qualified field name
// - "token"."field1": some custom data used in custom functions, this example can be used by the implementation of Cassandra's token(field1)
// Capillaries always use fully qualified field names
type VarValuesMap map[string]map[string]any

func (vars *VarValuesMap) Tables() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for table := range *vars {
		sb.WriteString(fmt.Sprintf("%s ", table))
	}
	sb.WriteString("]")
	return sb.String()
}

func (vars *VarValuesMap) Names() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for table, fldMap := range *vars {
		for fld := range fldMap {
			sb.WriteString(fmt.Sprintf("%s.%s ", table, fld))
		}
	}
	sb.WriteString("]")
	return sb.String()
}

type EvalCtx struct {
	aggFunc            AggFuncType
	aggType            AggDataType
	aggCallExp         *ast.CallExpr
	count              int64
	stringAggCollector StringAggCollector
	sumCollector       SumCollector
	avgCollector       AvgCollector
	minCollector       MinCollector
	maxCollector       MaxCollector
	value              any
	aggEnabled         AggEnabledType
	// If >=0, round all intermediate dec calculations to this number of decimal digits.
	// This approach does not work for: avg(decimal2*decimal4) because it is not clear how to round decimal2*decimal4
	// Users have to stick to one precision, unfortunately.
	roundDec int32
	// Provided by caller
	evalFunctions map[string]EvalFunction
	evalConstants map[string]any
	evalVars      VarValuesMap
}

func (ectx *EvalCtx) IsAggFuncEnabled() bool {
	return ectx.aggEnabled == AggFuncEnabled
}

func (ectx *EvalCtx) SetVars(vars VarValuesMap) {
	ectx.evalVars = vars
}

func (ectx *EvalCtx) SetRoundDec(roundDec int32) {
	ectx.roundDec = roundDec
}

func (ectx *EvalCtx) GetValue() any {
	if ectx.aggEnabled == AggFuncEnabled && (ectx.aggFunc == AggCount || ectx.aggFunc == AggCountIf || ectx.aggFunc == AggSum || ectx.aggFunc == AggSumIf || ectx.aggFunc == AggAvg || ectx.aggFunc == AggAvgIf) && ectx.value == nil {
		return int64(0)
	}
	return ectx.value
}

func (ectx *EvalCtx) GetSafeValue(defaultValue any) any {
	if ectx.aggEnabled == AggFuncEnabled && (ectx.aggFunc == AggCount || ectx.aggFunc == AggCountIf || ectx.aggFunc == AggSum || ectx.aggFunc == AggSumIf || ectx.aggFunc == AggAvg || ectx.aggFunc == AggAvgIf) {
		return ectx.GetValue()
	}
	if ectx.value == nil {
		return defaultValue
	}

	return ectx.value
}

// Not ready to make these limits/defaults public

const (
	maxSupportedInt   int64   = int64(math.MaxInt64)
	minSupportedInt   int64   = int64(math.MinInt64)
	maxSupportedFloat float64 = math.MaxFloat64
	minSupportedFloat float64 = -math.MaxFloat32
)

func maxSupportedDecimal() decimal.Decimal {
	return decimal.NewFromFloat32(math.MaxFloat32)
}
func minSupportedDecimal() decimal.Decimal {
	return decimal.NewFromFloat32(-math.MaxFloat32 + 1)
}

func getAggStringSeparator(aggFuncType AggFuncType, aggFuncArgs []ast.Expr) (string, error) {
	if aggFuncType == AggStringAgg && len(aggFuncArgs) != 2 {
		return "", fmt.Errorf("%s must have two parameters", aggFuncType)
	} else if aggFuncType == AggStringAggIf && len(aggFuncArgs) != 3 {
		return "", fmt.Errorf("%s must have three parameters", aggFuncType)
	}
	switch separatorExpTyped := aggFuncArgs[1].(type) {
	case *ast.BasicLit:
		switch separatorExpTyped.Kind {
		case token.STRING:
			return strings.Trim(separatorExpTyped.Value, "\""), nil
		default:
			return "", errors.New("string_agg/if second parameter must be a constant string")
		}
	default:
		return "", errors.New("string_agg/if second parameter must be a basic literal")
	}
}

func defaultDecimal() decimal.Decimal {
	// Explicit zero, otherwise its decimal NIL
	return decimal.NewFromInt(0)
}

func defaultBigint() *big.Int {
	return big.NewInt(0)
}

func newPlainEvalCtx(aggEnabled AggEnabledType) *EvalCtx {
	return &EvalCtx{
		aggFunc:            AggUnknown,
		aggType:            AggTypeUnknown,
		aggEnabled:         aggEnabled,
		stringAggCollector: StringAggCollector{Separator: "", Sb: strings.Builder{}},
		sumCollector:       SumCollector{Dec: defaultDecimal()},
		avgCollector:       AvgCollector{Dec: defaultDecimal(), Int: defaultBigint()},
		minCollector:       MinCollector{Int: maxSupportedInt, Float: maxSupportedFloat, Dec: maxSupportedDecimal(), Str: ""},
		maxCollector:       MaxCollector{Int: minSupportedInt, Float: minSupportedFloat, Dec: minSupportedDecimal(), Str: ""},
		roundDec:           -1,
	}
}

func NewPlainEvalCtx(functions map[string]EvalFunction, constants map[string]any, vars VarValuesMap) *EvalCtx {
	eCtx := newPlainEvalCtx(AggFuncDisabled)
	eCtx.evalFunctions = functions
	eCtx.evalConstants = constants
	eCtx.evalVars = vars
	return eCtx
}

func NewAggEvalCtx(aggFuncType AggFuncType, aggFuncArgs []ast.Expr, functions map[string]EvalFunction, constants map[string]any, vars VarValuesMap) (*EvalCtx, error) {
	eCtx := newPlainEvalCtx(AggFuncEnabled)
	eCtx.aggFunc = aggFuncType
	eCtx.evalFunctions = functions
	eCtx.evalConstants = constants
	eCtx.evalVars = vars

	// Special case: we need to provide eCtx.StringAgg with a separator and
	// explicitly set its type to AggTypeString from the very beginning (instead of detecting it later, as we do for other agg functions)
	if aggFuncType == AggStringAgg || aggFuncType == AggStringAggIf {
		var aggStringErr error
		eCtx.stringAggCollector.Separator, aggStringErr = getAggStringSeparator(aggFuncType, aggFuncArgs)
		if aggStringErr != nil {
			return nil, aggStringErr
		}
		eCtx.aggType = AggTypeString
	}
	return eCtx, nil
}

func CheckArgs(funcName string, requiredArgCount int, actualArgCount int) error {
	if actualArgCount != requiredArgCount {
		return fmt.Errorf("cannot evaluate %s(), requires %d args, %d supplied", funcName, requiredArgCount, actualArgCount)
	}
	return nil
}

func (eCtx *EvalCtx) EvalBinaryInt(valLeftVolatile any, op token.Token, valRightVolatile any) (result int64, finalErr error) {

	result = math.MaxInt
	valLeft, ok := valLeftVolatile.(int64)
	if !ok {
		return 0, fmt.Errorf("cannot evaluate binary int64 expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(int64)
	if !ok {
		return 0, fmt.Errorf("cannot evaluate binary int64 expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	defer func() {
		if r := recover(); r != nil {
			finalErr = fmt.Errorf("%v", r)
		}
	}()

	switch op {
	case token.ADD:
		return valLeft + valRight, nil
	case token.SUB:
		return valLeft - valRight, nil
	case token.MUL:
		return valLeft * valRight, nil
	case token.QUO:
		return valLeft / valRight, nil
	case token.REM:
		return valLeft % valRight, nil
	default:
		return 0, fmt.Errorf("cannot perform int op %v against int %d and int %d", op, valLeft, valRight)
	}
}

func isCompareOp(op token.Token) bool {
	return op == token.GTR || op == token.LSS || op == token.GEQ || op == token.LEQ || op == token.EQL || op == token.NEQ
}

func (eCtx *EvalCtx) EvalBinaryIntToBool(valLeftVolatile any, op token.Token, valRightVolatile any) (bool, error) {

	valLeft, ok := valLeftVolatile.(int64)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary int64 expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(int64)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary int64 expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if !isCompareOp(op) {
		return false, fmt.Errorf("cannot perform bool op %v against int %d and int %d", op, valLeft, valRight)
	}

	if op == token.GTR && valLeft > valRight ||
		op == token.LSS && valLeft < valRight ||
		op == token.GEQ && valLeft >= valRight ||
		op == token.LEQ && valLeft <= valRight ||
		op == token.EQL && valLeft == valRight ||
		op == token.NEQ && valLeft != valRight {
		return true, nil
	}
	return false, nil
}

func (eCtx *EvalCtx) EvalBinaryFloat64ToBool(valLeftVolatile any, op token.Token, valRightVolatile any) (bool, error) {

	valLeft, ok := valLeftVolatile.(float64)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary foat64 expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(float64)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary float64 expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if !isCompareOp(op) {
		return false, fmt.Errorf("cannot perform bool op %v against float %f and float %f", op, valLeft, valRight)
	}

	if op == token.GTR && valLeft > valRight ||
		op == token.LSS && valLeft < valRight ||
		op == token.GEQ && valLeft >= valRight ||
		op == token.LEQ && valLeft <= valRight ||
		op == token.EQL && valLeft == valRight ||
		op == token.NEQ && valLeft != valRight {
		return true, nil
	}
	return false, nil
}

func (eCtx *EvalCtx) EvalBinaryDecimal2ToBool(valLeftVolatile any, op token.Token, valRightVolatile any) (bool, error) {

	valLeft, ok := valLeftVolatile.(decimal.Decimal)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary decimal expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(decimal.Decimal)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary decimal expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if !isCompareOp(op) {
		return false, fmt.Errorf("cannot perform bool op %v against decimal %v and decimal %v", op, valLeft, valRight)
	}
	if op == token.GTR && valLeft.Cmp(valRight) > 0 ||
		op == token.LSS && valLeft.Cmp(valRight) < 0 ||
		op == token.GEQ && valLeft.Cmp(valRight) >= 0 ||
		op == token.LEQ && valLeft.Cmp(valRight) <= 0 ||
		op == token.EQL && valLeft.Cmp(valRight) == 0 ||
		op == token.NEQ && valLeft.Cmp(valRight) != 0 {
		return true, nil
	}
	return false, nil
}

func (eCtx *EvalCtx) EvalBinaryTimeToBool(valLeftVolatile any, op token.Token, valRightVolatile any) (bool, error) {

	valLeft, ok := valLeftVolatile.(time.Time)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary time expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(time.Time)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary time expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if !isCompareOp(op) {
		return false, fmt.Errorf("cannot perform bool op %v against time %v and time %v", op, valLeft, valRight)
	}
	if op == token.GTR && valLeft.After(valRight) ||
		op == token.LSS && valLeft.Before(valRight) ||
		op == token.GEQ && (valLeft.After(valRight) || valLeft.Equal(valRight)) ||
		op == token.LEQ && (valLeft.Before(valRight) || valLeft.Equal(valRight)) ||
		op == token.EQL && valLeft.Equal(valRight) ||
		op == token.NEQ && !valLeft.Equal(valRight) {
		return true, nil
	}
	return false, nil
}

func (eCtx *EvalCtx) EvalBinaryFloat64(valLeftVolatile any, op token.Token, valRightVolatile any) (float64, error) {

	valLeft, ok := valLeftVolatile.(float64)
	if !ok {
		return 0.0, fmt.Errorf("cannot evaluate binary float64 expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(float64)
	if !ok {
		return 0.0, fmt.Errorf("cannot evaluate binary float expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	switch op {
	case token.ADD:
		return valLeft + valRight, nil
	case token.SUB:
		return valLeft - valRight, nil
	case token.MUL:
		return valLeft * valRight, nil
	case token.QUO:
		return valLeft / valRight, nil
	default:
		return 0, fmt.Errorf("cannot perform float64 op %v against float64 %f and float64 %f", op, valLeft, valRight)
	}
}

func (eCtx *EvalCtx) EvalBinaryDecimal(valLeftVolatile any, op token.Token, valRightVolatile any) (result decimal.Decimal, err error) {

	result = decimal.NewFromFloat(math.MaxFloat64)
	err = nil
	valLeft, ok := valLeftVolatile.(decimal.Decimal)
	if !ok {
		return decimal.NewFromInt(0), fmt.Errorf("cannot evaluate binary decimal expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(decimal.Decimal)
	if !ok {
		return decimal.NewFromInt(0), fmt.Errorf("cannot evaluate binary decimal expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	var val decimal.Decimal
	switch op {
	case token.ADD:
		val = valLeft.Add(valRight)
	case token.SUB:
		val = valLeft.Sub(valRight)
	case token.MUL:
		val = valLeft.Mul(valRight)
	case token.QUO:
		val = valLeft.Div(valRight)
	default:
		return decimal.NewFromInt(0), fmt.Errorf("cannot perform decimal op %v against decimal %v and float64 %v", op, valLeft, valRight)
	}

	// Round(2) when needed
	if eCtx.roundDec >= 0 {
		val = val.Round(eCtx.roundDec)
	}

	return val, nil
}

func (eCtx *EvalCtx) EvalBinaryBool(valLeftVolatile any, op token.Token, valRightVolatile any) (bool, error) {

	valLeft, ok := valLeftVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary bool expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary bool expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	switch op {
	case token.LAND:
		return valLeft && valRight, nil
	case token.LOR:
		return valLeft || valRight, nil
	default:
		return false, fmt.Errorf("cannot perform bool op %v against bool %v and bool %v", op, valLeft, valRight)
	}
}

func (eCtx *EvalCtx) EvalBinaryBoolToBool(valLeftVolatile any, op token.Token, valRightVolatile any) (bool, error) {

	valLeft, ok := valLeftVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary bool expression %v with %T on the left", op, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary bool expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if !(op == token.EQL || op == token.NEQ) {
		return false, fmt.Errorf("cannot evaluate binary bool expression, op %v not supported (and will never be)", op)
	}

	if op == token.EQL && valLeft == valRight ||
		op == token.NEQ && valLeft != valRight {
		return true, nil
	}
	return false, nil
}

func (eCtx *EvalCtx) EvalUnaryBoolNot(exp ast.Expr) (bool, error) {
	valVolatile, err := eCtx.Eval(exp)
	if err != nil {
		return false, err
	}

	val, ok := valVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot evaluate unary bool not expression with %T on the right", valVolatile)
	}

	return !val, nil
}

func (eCtx *EvalCtx) EvalUnaryMinus(exp ast.Expr) (any, error) {
	valVolatile, err := eCtx.Eval(exp)
	if err != nil {
		return false, err
	}

	switch typedVal := valVolatile.(type) {
	case int:
		return int64(-typedVal), nil
	case int16:
		return int64(-typedVal), nil
	case int32:
		return int64(-typedVal), nil
	case int64:
		return -typedVal, nil
	case float32:
		return float64(-typedVal), nil
	case float64:
		return -typedVal, nil
	case decimal.Decimal:
		return typedVal.Neg(), nil
	default:
		return false, fmt.Errorf("cannot evaluate unary minus expression '-%v(%T)', unsupported type", valVolatile, valVolatile)
	}
}

func (eCtx *EvalCtx) EvalBinaryString(valLeftVolatile any, op token.Token, valRightVolatile any) (string, error) {

	valLeft, ok := valLeftVolatile.(string)
	if !ok {
		return "", fmt.Errorf("cannot evaluate binary string expression %v with %T on the left", op, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(string)
	if !ok {
		return "", fmt.Errorf("cannot evaluate binary string expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	switch op {
	case token.ADD:
		return valLeft + valRight, nil
	default:
		return "", fmt.Errorf("cannot perform string op %v against string '%s' and string '%s', op not supported", op, valLeft, valRight)
	}
}

func (eCtx *EvalCtx) EvalBinaryStringToBool(valLeftVolatile any, op token.Token, valRightVolatile any) (bool, error) {

	valLeft, ok := valLeftVolatile.(string)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary string expression %v with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}
	valLeft = strings.ReplaceAll(strings.Trim(valLeft, "\""), `\"`, `\`)

	valRight, ok := valRightVolatile.(string)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary decimal expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}
	valRight = strings.ReplaceAll(strings.Trim(valRight, "\""), `\"`, `"`)

	if !isCompareOp(op) {
		return false, fmt.Errorf("cannot perform bool op %v against string %v and string %v", op, valLeft, valRight)
	}
	if op == token.GTR && valLeft > valRight ||
		op == token.LSS && valLeft < valRight ||
		op == token.GEQ && valLeft >= valRight ||
		op == token.LEQ && valLeft <= valRight ||
		op == token.EQL && valLeft == valRight ||
		op == token.NEQ && valLeft != valRight {
		return true, nil
	}
	return false, nil
}

func (eCtx *EvalCtx) EvalFunc(callExp *ast.CallExpr, funcName string, args []any) (any, error) {
	var err error
	switch funcName {
	// Aggregate functions, this is part of evalcore
	case "string_agg":
		eCtx.value, err = eCtx.CallAggStringAgg(callExp, args)
	case "sum":
		eCtx.value, err = eCtx.CallAggSum(callExp, args)
	case "count":
		eCtx.value, err = eCtx.CallAggCount(callExp, args)
	case "avg":
		eCtx.value, err = eCtx.CallAggAvg(callExp, args)
	case "min":
		eCtx.value, err = eCtx.CallAggMin(callExp, args)
	case "max":
		eCtx.value, err = eCtx.CallAggMax(callExp, args)
	case "string_agg_if":
		eCtx.value, err = eCtx.CallAggStringAggIf(callExp, args)
	case "sum_if":
		eCtx.value, err = eCtx.CallAggSumIf(callExp, args)
	case "count_if":
		eCtx.value, err = eCtx.CallAggCountIf(callExp, args)
	case "avg_if":
		eCtx.value, err = eCtx.CallAggAvgIf(callExp, args)
	case "min_if":
		eCtx.value, err = eCtx.CallAggMinIf(callExp, args)
	case "max_if":
		eCtx.value, err = eCtx.CallAggMaxIf(callExp, args)

	default:
		// Caller-provided functions
		if eCtx.evalFunctions != nil {
			if evalFunc, ok := eCtx.evalFunctions[funcName]; ok {
				return evalFunc(args)
			}
		}
		return nil, fmt.Errorf("cannot evaluate unsupported func '%s'", funcName)
	}
	return eCtx.value, err
}

func (eCtx *EvalCtx) evalBinaryArithmeticExp(valLeftVolatile any, exp *ast.BinaryExpr, valRightVolatile any) (any, error) {
	switch valLeftVolatile.(type) {
	case string:
		var err error
		eCtx.value, err = eCtx.EvalBinaryString(valLeftVolatile, exp.Op, valRightVolatile)
		return eCtx.value, err
	default:
		// Assume both args are numbers (int, float, dec)
		stdArgLeft, stdArgRight, err := castNumberPairToCommonType(valLeftVolatile, valRightVolatile)
		if err != nil {
			return nil, fmt.Errorf("cannot perform binary arithmetic op, incompatible arg types '%v(%T)' %v '%v(%T)' ", valLeftVolatile, valLeftVolatile, exp.Op, valRightVolatile, valRightVolatile)
		}
		switch stdArgLeft.(type) {
		case int64:
			eCtx.value, err = eCtx.EvalBinaryInt(stdArgLeft, exp.Op, stdArgRight)
			return eCtx.value, err
		case float64:
			eCtx.value, err = eCtx.EvalBinaryFloat64(stdArgLeft, exp.Op, stdArgRight)
			return eCtx.value, err
		case decimal.Decimal:
			eCtx.value, err = eCtx.EvalBinaryDecimal(stdArgLeft, exp.Op, stdArgRight)
			return eCtx.value, err
		default:
			return nil, fmt.Errorf("cannot perform binary arithmetic op, unexpected std type '%v(%T)' %v '%v(%T)' ", valLeftVolatile, valLeftVolatile, exp.Op, valRightVolatile, valRightVolatile)
		}
	}
}

func (eCtx *EvalCtx) evalBinaryBoolToBoolExp(valLeftVolatile any, exp *ast.BinaryExpr, valRightVolatile any) (any, error) {
	switch valLeftTyped := valLeftVolatile.(type) {
	case bool:
		var err error
		eCtx.value, err = eCtx.EvalBinaryBool(valLeftTyped, exp.Op, valRightVolatile)
		return eCtx.value, err
	default:
		return nil, fmt.Errorf("cannot perform binary op %v against %T left", exp.Op, valLeftVolatile)
	}
}
func (eCtx *EvalCtx) evalBinaryCompareExp(valLeftVolatile any, exp *ast.BinaryExpr, valRightVolatile any) (any, error) {
	if (valLeftVolatile == nil && valRightVolatile != nil) || (valLeftVolatile != nil && valRightVolatile == nil) {
		// Cannot be compared, NEQ returns true, all other ops return false
		eCtx.value = (exp.Op == token.NEQ)
		return eCtx.value, nil
	}
	if valLeftVolatile == nil && valRightVolatile == nil {
		// EQ returns true, all other ops return false
		eCtx.value = (exp.Op == token.EQL)
		return eCtx.value, nil
	}
	var err error
	switch valLeftVolatile.(type) {
	case time.Time:
		eCtx.value, err = eCtx.EvalBinaryTimeToBool(valLeftVolatile, exp.Op, valRightVolatile)
		return eCtx.value, err
	case string:
		eCtx.value, err = eCtx.EvalBinaryStringToBool(valLeftVolatile, exp.Op, valRightVolatile)
		return eCtx.value, err
	case bool:
		eCtx.value, err = eCtx.EvalBinaryBoolToBool(valLeftVolatile, exp.Op, valRightVolatile)
		return eCtx.value, err
	default:
		// Assume both args are numbers (int, float, dec)
		stdArgLeft, stdArgRight, err := castNumberPairToCommonType(valLeftVolatile, valRightVolatile)
		if err != nil {
			return nil, fmt.Errorf("cannot perform binary comp op, incompatible arg types '%v(%T)' %v '%v(%T)' ", valLeftVolatile, valLeftVolatile, exp.Op, valRightVolatile, valRightVolatile)
		}
		switch stdArgLeft.(type) {
		case int64:
			eCtx.value, err = eCtx.EvalBinaryIntToBool(stdArgLeft, exp.Op, stdArgRight)
			return eCtx.value, err
		case float64:
			eCtx.value, err = eCtx.EvalBinaryFloat64ToBool(stdArgLeft, exp.Op, stdArgRight)
			return eCtx.value, err
		case decimal.Decimal:
			eCtx.value, err = eCtx.EvalBinaryDecimal2ToBool(stdArgLeft, exp.Op, stdArgRight)
			return eCtx.value, err
		default:
			return nil, fmt.Errorf("cannot perform binary comp op, unexpected std type '%v(%T)' %v '%v(%T)' ", valLeftVolatile, valLeftVolatile, exp.Op, valRightVolatile, valRightVolatile)
		}
	}
}

func (eCtx *EvalCtx) evalBinaryExp(exp *ast.BinaryExpr) (any, error) {
	valLeftVolatile, err := eCtx.Eval(exp.X)
	if err != nil {
		return nil, err
	}
	valRightVolatile, err := eCtx.Eval(exp.Y)
	if err != nil {
		return 0, err
	}
	if exp.Op == token.ADD || exp.Op == token.SUB || exp.Op == token.MUL || exp.Op == token.QUO || exp.Op == token.REM {
		return eCtx.evalBinaryArithmeticExp(valLeftVolatile, exp, valRightVolatile)
	} else if exp.Op == token.LOR || exp.Op == token.LAND {
		return eCtx.evalBinaryBoolToBoolExp(valLeftVolatile, exp, valRightVolatile)
	} else if exp.Op == token.GTR || exp.Op == token.GEQ || exp.Op == token.LSS || exp.Op == token.LEQ || exp.Op == token.EQL || exp.Op == token.NEQ {
		return eCtx.evalBinaryCompareExp(valLeftVolatile, exp, valRightVolatile)
	}
	return nil, fmt.Errorf("cannot perform binary expression unknown op %v", exp.Op)
}

func (eCtx *EvalCtx) evalUnaryExp(exp *ast.UnaryExpr) (any, error) {
	switch exp.Op {
	case token.NOT:
		var err error
		eCtx.value, err = eCtx.EvalUnaryBoolNot(exp.X)
		return eCtx.value, err
	case token.SUB:
		var err error
		eCtx.value, err = eCtx.EvalUnaryMinus(exp.X)
		return eCtx.value, err
	default:
		return nil, fmt.Errorf("cannot evaluate unary op %v, unknown op", exp.Op)
	}
}

// When adding support for another expression here, make sure it's also handled in harvestFieldRefsFromParsedExpression()
func (eCtx *EvalCtx) Eval(exp ast.Expr) (any, error) {
	switch exp := exp.(type) {
	case *ast.BinaryExpr:
		return eCtx.evalBinaryExp(exp)

	case *ast.BasicLit:
		switch exp.Kind {
		case token.INT:
			i, _ := strconv.ParseInt(exp.Value, 10, 64)
			eCtx.value = i
			return i, nil
		case token.FLOAT:
			i, _ := strconv.ParseFloat(exp.Value, 64)
			eCtx.value = i
			return i, nil
		case token.IDENT:
			return nil, fmt.Errorf("cannot evaluate expression %s of type token.IDENT", exp.Value)
		case token.STRING:
			eCtx.value = exp.Value
			if exp.Value[0] == '"' {
				return strings.Trim(exp.Value, "\""), nil
			}
			return strings.Trim(exp.Value, "`"), nil

		default:
			return nil, fmt.Errorf("cannot evaluate expression %s of type %v", exp.Value, exp.Kind)
		}

	case *ast.UnaryExpr:
		return eCtx.evalUnaryExp(exp)

	case *ast.Ident:
		if eCtx.evalConstants != nil {
			golangConst, ok := eCtx.evalConstants[exp.Name]
			if ok {
				eCtx.value = golangConst
				return golangConst, nil
			}
		}

		if eCtx.evalVars == nil {
			return nil, fmt.Errorf("cannot evaluate ident expression '%s', no variables supplied to the context", exp.Name)
		}

		// Non-selector idents are stored under ""
		objectAttributes, ok := eCtx.evalVars[""]
		if !ok {
			return nil, fmt.Errorf("cannot evaluate ident expression '%s', no empty object", exp.Name)
		}

		val, ok := objectAttributes[exp.Name]
		if !ok {
			return nil, fmt.Errorf("cannot evaluate ident expression %s, variable not supplied", exp.Name)
		}
		eCtx.value = val
		return val, nil

	case *ast.CallExpr:
		args := make([]any, len(exp.Args))

		for i, v := range exp.Args {
			arg, err := eCtx.Eval(v)
			if err != nil {
				return nil, err
			}
			args[i] = arg
		}

		switch typedExp := exp.Fun.(type) {
		case *ast.Ident:
			var err error
			eCtx.value, err = eCtx.EvalFunc(exp, typedExp.Name, args)
			return eCtx.value, err

		case *ast.SelectorExpr:
			switch expIdent := typedExp.X.(type) {
			case *ast.Ident:
				var err error
				eCtx.value, err = eCtx.EvalFunc(exp, fmt.Sprintf("%s.%s", expIdent.Name, typedExp.Sel.Name), args)
				return eCtx.value, err
			default:
				return nil, fmt.Errorf("cannot evaluate fun expression %v, unknown type of X: %T", typedExp.X, typedExp.X)
			}

		default:
			return nil, fmt.Errorf("cannot evaluate func call expression %v, unknown type of X: %T", exp.Fun, exp.Fun)
		}

	case *ast.SelectorExpr:
		switch objectIdent := exp.X.(type) {
		case *ast.Ident:
			if eCtx.evalConstants != nil {
				golangConst, ok := eCtx.evalConstants[fmt.Sprintf("%s.%s", objectIdent.Name, exp.Sel.Name)]
				if ok {
					eCtx.value = golangConst
					return golangConst, nil
				}
			}

			if eCtx.evalVars == nil {
				return nil, fmt.Errorf("cannot evaluate selector ident expression '%s', no variables supplied to the context", objectIdent.Name)
			}

			objectAttributes, ok := eCtx.evalVars[objectIdent.Name]
			if !ok {
				return nil, fmt.Errorf("cannot evaluate selector ident expression '%s', variable not supplied, check table/alias name", objectIdent.Name)
			}

			val, ok := objectAttributes[exp.Sel.Name]
			if !ok {
				return nil, fmt.Errorf("cannot evaluate selector ident expression %s.%s, variable not supplied, check field name", objectIdent.Name, exp.Sel.Name)
			}
			eCtx.value = val
			return val, nil
		default:
			return nil, fmt.Errorf("cannot evaluate selector expression %v, unknown type of X: %T", exp.X, exp.X)
		}

	case *ast.ParenExpr:
		return eCtx.Eval(exp.X)

	default:
		return nil, fmt.Errorf("cannot evaluate expression %v of unsupported type %T", exp, exp)
	}
}
