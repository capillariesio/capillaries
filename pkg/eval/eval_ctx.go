package eval

import (
	"fmt"
	"go/ast"
	"go/token"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type EvalCtx struct {
	Vars       *VarValuesMap
	AggFunc    AggFuncType
	AggType    AggDataType
	AggCallExp *ast.CallExpr
	Count      int64
	Sum        SumCollector
	Avg        AvgCollector
	Min        MinCollector
	Max        MaxCollector
	Value      interface{}
	AggEnabled bool
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

func defaultDecimal() decimal.Decimal {
	// Explicit zero, otherwise its decimal NIL
	return decimal.NewFromInt(0)
}

func NewPlainEvalCtx(aggEnabled bool) EvalCtx {
	return EvalCtx{
		AggFunc:    AggUnknown,
		AggType:    AggTypeUnknown,
		AggEnabled: aggEnabled,
		Sum:        SumCollector{Dec: defaultDecimal()},
		Avg:        AvgCollector{Dec: defaultDecimal()},
		Min:        MinCollector{Int: maxSupportedInt, Float: maxSupportedFloat, Dec: maxSupportedDecimal(), Str: ""},
		Max:        MaxCollector{Int: minSupportedInt, Float: minSupportedFloat, Dec: minSupportedDecimal(), Str: ""}}
}

func NewPlainEvalCtxWithVars(aggEnabled bool, vars *VarValuesMap) EvalCtx {
	return EvalCtx{
		AggFunc:    AggUnknown,
		Vars:       vars,
		AggType:    AggTypeUnknown,
		AggEnabled: aggEnabled,
		Sum:        SumCollector{Dec: defaultDecimal()},
		Avg:        AvgCollector{Dec: defaultDecimal()},
		Min:        MinCollector{Int: maxSupportedInt, Float: maxSupportedFloat, Dec: maxSupportedDecimal(), Str: ""},
		Max:        MaxCollector{Int: minSupportedInt, Float: minSupportedFloat, Dec: minSupportedDecimal(), Str: ""}}
}

func checkArgs(funcName string, requiredArgCount int, actualArgCount int) error {
	if actualArgCount != requiredArgCount {
		return fmt.Errorf("cannot evaluate %s(), requires %d args, %d supplied", funcName, requiredArgCount, actualArgCount)
	} else {
		return nil
	}
}

func (eCtx *EvalCtx) EvalBinaryInt(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (result int64, finalErr error) {

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

func (eCtx *EvalCtx) EvalBinaryIntToBool(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (bool, error) {

	valLeft, ok := valLeftVolatile.(int64)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary int64 expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(int64)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary int64 expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if op == token.GTR && valLeft > valRight ||
		op == token.LSS && valLeft < valRight ||
		op == token.GEQ && valLeft >= valRight ||
		op == token.LEQ && valLeft <= valRight ||
		op == token.EQL && valLeft == valRight ||
		op == token.NEQ && valLeft != valRight {
		return true, nil
	} else {
		return false, nil
	}
}

func (eCtx *EvalCtx) EvalBinaryFloat64ToBool(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (bool, error) {

	valLeft, ok := valLeftVolatile.(float64)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary foat64 expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(float64)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary float64 expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if op == token.GTR && valLeft > valRight ||
		op == token.LSS && valLeft < valRight ||
		op == token.GEQ && valLeft >= valRight ||
		op == token.LEQ && valLeft <= valRight ||
		op == token.EQL && valLeft == valRight ||
		op == token.NEQ && valLeft != valRight {
		return true, nil
	} else {
		return false, nil
	}
}

func (eCtx *EvalCtx) EvalBinaryDecimal2ToBool(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (bool, error) {

	valLeft, ok := valLeftVolatile.(decimal.Decimal)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary decimal2 expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(decimal.Decimal)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary decimal2 expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if op == token.GTR && valLeft.Cmp(valRight) > 0 ||
		op == token.LSS && valLeft.Cmp(valRight) < 0 ||
		op == token.GEQ && valLeft.Cmp(valRight) >= 0 ||
		op == token.LEQ && valLeft.Cmp(valRight) <= 0 ||
		op == token.EQL && valLeft.Cmp(valRight) == 0 ||
		op == token.NEQ && valLeft.Cmp(valRight) != 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (eCtx *EvalCtx) EvalBinaryTimeToBool(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (bool, error) {

	valLeft, ok := valLeftVolatile.(time.Time)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary time expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(time.Time)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary time expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if op == token.GTR && valLeft.After(valRight) ||
		op == token.LSS && valLeft.Before(valRight) ||
		op == token.GEQ && (valLeft.After(valRight) || valLeft == valRight) ||
		op == token.LEQ && (valLeft.Before(valRight) || valLeft == valRight) ||
		op == token.EQL && valLeft == valRight ||
		op == token.NEQ && valLeft != valRight {
		return true, nil
	} else {
		return false, nil
	}
}

func (eCtx *EvalCtx) EvalBinaryFloat64(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (float64, error) {

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

func (eCtx *EvalCtx) EvalBinaryDecimal2(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (result decimal.Decimal, err error) {

	result = decimal.NewFromFloat(math.MaxFloat64)
	err = nil
	valLeft, ok := valLeftVolatile.(decimal.Decimal)
	if !ok {
		return decimal.NewFromInt(0), fmt.Errorf("cannot evaluate binary decimal2 expression '%v' with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(decimal.Decimal)
	if !ok {
		return decimal.NewFromInt(0), fmt.Errorf("cannot evaluate binary decimal2 expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	switch op {
	case token.ADD:
		return valLeft.Add(valRight).Round(2), nil
	case token.SUB:
		return valLeft.Sub(valRight).Round(2), nil
	case token.MUL:
		return valLeft.Mul(valRight).Round(2), nil
	case token.QUO:
		return valLeft.Div(valRight).Round(2), nil
	default:
		return decimal.NewFromInt(0), fmt.Errorf("cannot perform decimal2 op %v against decimal2 %v and float64 %v", op, valLeft, valRight)
	}
}

func (eCtx *EvalCtx) EvalBinaryBool(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (bool, error) {

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

func (eCtx *EvalCtx) EvalBinaryBoolToBool(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (bool, error) {

	valLeft, ok := valLeftVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary bool expression %v with %T on the left", op, valLeftVolatile)
	}

	valRight, ok := valRightVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary bool expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}

	if op == token.EQL || op == token.NEQ {
		if op == token.EQL && valLeft == valRight ||
			op == token.NEQ && valLeft != valRight {
			return true, nil
		} else {
			return false, nil
		}
	} else {
		return false, fmt.Errorf("cannot evaluate binary bool expression, op %v not supported (and will never be)", op)
	}
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

func (eCtx *EvalCtx) EvalUnaryMinus(exp ast.Expr) (interface{}, error) {
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

func (eCtx *EvalCtx) EvalBinaryString(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (string, error) {

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

func (eCtx *EvalCtx) EvalBinaryStringToBool(valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}) (bool, error) {

	valLeft, ok := valLeftVolatile.(string)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary string expression %v with '%v(%T)' on the left", op, valLeftVolatile, valLeftVolatile)
	}
	valLeft = strings.Replace(strings.Trim(valLeft, "\""), `\"`, `\`, -1)

	valRight, ok := valRightVolatile.(string)
	if !ok {
		return false, fmt.Errorf("cannot evaluate binary decimal2 expression '%v(%T) %v %v(%T)', invalid right arg", valLeft, valLeft, op, valRightVolatile, valRightVolatile)
	}
	valRight = strings.Replace(strings.Trim(valRight, "\""), `\"`, `"`, -1)

	if op == token.GTR && valLeft > valRight ||
		op == token.LSS && valLeft < valRight ||
		op == token.GEQ && valLeft >= valRight ||
		op == token.LEQ && valLeft <= valRight ||
		op == token.EQL && valLeft == valRight ||
		op == token.NEQ && valLeft != valRight {
		return true, nil
	} else {
		return false, nil
	}
}

func (eCtx *EvalCtx) EvalFunc(callExp *ast.CallExpr, funcName string, args []interface{}) (interface{}, error) {
	var err error = nil
	switch funcName {
	case "math.Sqrt":
		eCtx.Value, err = callMathSqrt(args)
	case "len":
		eCtx.Value, err = callLen(args)
	case "string":
		eCtx.Value, err = callString(args)
	case "float":
		eCtx.Value, err = callFloat(args)
	case "int":
		eCtx.Value, err = callInt(args)
	case "decimal2":
		eCtx.Value, err = callDecimal2(args)
	case "time.Parse":
		eCtx.Value, err = callTimeParse(args)
	case "time.Date":
		eCtx.Value, err = callTimeDate(args)
	case "time.Now":
		eCtx.Value, err = callTimeNow(args)
	case "time.Unix":
		eCtx.Value, err = callTimeUnix(args)
	case "time.UnixMilli":
		eCtx.Value, err = callTimeUnixMilli(args)
	case "time.DiffMilli":
		eCtx.Value, err = callTimeDiffMilli(args)
	case "time.FixedZone":
		eCtx.Value, err = callTimeFixedZone(args)
	case "re.MatchString":
		eCtx.Value, err = callReMatchString(args)
	case "strings.ReplaceAll":
		eCtx.Value, err = callStringsReplaceAll(args)

	// Aggregate functions, to be used only in grouped lookups

	case "sum":
		eCtx.Value, err = eCtx.CallAggSum(callExp, args)
	case "count":
		eCtx.Value, err = eCtx.CallAggCount(callExp, args)
	case "avg":
		eCtx.Value, err = eCtx.CallAggAvg(callExp, args)
	case "min":
		eCtx.Value, err = eCtx.CallAggMin(callExp, args)
	case "max":
		eCtx.Value, err = eCtx.CallAggMax(callExp, args)

	default:
		return nil, fmt.Errorf("cannot evaluate unsupported func '%s'", funcName)
	}
	return eCtx.Value, err
}

func (eCtx *EvalCtx) Eval(exp ast.Expr) (interface{}, error) {
	switch exp := exp.(type) {
	case *ast.BinaryExpr:
		valLeftVolatile, err := eCtx.Eval(exp.X)
		if err != nil {
			return nil, err
		}

		valRightVolatile, err := eCtx.Eval(exp.Y)
		if err != nil {
			return 0, err
		}

		if exp.Op == token.ADD || exp.Op == token.SUB || exp.Op == token.MUL || exp.Op == token.QUO || exp.Op == token.REM {
			switch valLeftVolatile.(type) {
			case string:
				eCtx.Value, err = eCtx.EvalBinaryString(valLeftVolatile, exp.Op, valRightVolatile)
				return eCtx.Value, err

			default:
				// Assume both args are numbers (int, float, dec)
				stdArgLeft, stdArgRight, err := castNumberPairToCommonType(valLeftVolatile, valRightVolatile)
				if err != nil {
					return nil, fmt.Errorf("cannot perform binary arithmetic op, incompatible arg types '%v(%T)' %v '%v(%T)' ", valLeftVolatile, valLeftVolatile, exp.Op, valRightVolatile, valRightVolatile)
				}
				switch stdArgLeft.(type) {
				case int64:
					eCtx.Value, err = eCtx.EvalBinaryInt(stdArgLeft, exp.Op, stdArgRight)
					return eCtx.Value, err
				case float64:
					eCtx.Value, err = eCtx.EvalBinaryFloat64(stdArgLeft, exp.Op, stdArgRight)
					return eCtx.Value, err
				case decimal.Decimal:
					eCtx.Value, err = eCtx.EvalBinaryDecimal2(stdArgLeft, exp.Op, stdArgRight)
					return eCtx.Value, err
				default:
					return nil, fmt.Errorf("cannot perform binary arithmetic op, unexpected std type '%v(%T)' %v '%v(%T)' ", valLeftVolatile, valLeftVolatile, exp.Op, valRightVolatile, valRightVolatile)
				}
			}
		} else if exp.Op == token.LOR || exp.Op == token.LAND {
			switch valLeftTyped := valLeftVolatile.(type) {
			case bool:
				eCtx.Value, err = eCtx.EvalBinaryBool(valLeftTyped, exp.Op, valRightVolatile)
				return eCtx.Value, err
			default:
				return nil, fmt.Errorf("cannot perform binary op %v against %T left", exp.Op, valLeftVolatile)
			}

		} else if exp.Op == token.GTR || exp.Op == token.GEQ || exp.Op == token.LSS || exp.Op == token.LEQ || exp.Op == token.EQL || exp.Op == token.NEQ {
			switch valLeftVolatile.(type) {
			case time.Time:
				eCtx.Value, err = eCtx.EvalBinaryTimeToBool(valLeftVolatile, exp.Op, valRightVolatile)
				return eCtx.Value, err
			case string:
				eCtx.Value, err = eCtx.EvalBinaryStringToBool(valLeftVolatile, exp.Op, valRightVolatile)
				return eCtx.Value, err
			case bool:
				eCtx.Value, err = eCtx.EvalBinaryBoolToBool(valLeftVolatile, exp.Op, valRightVolatile)
				return eCtx.Value, err
			default:
				// Assume both args are numbers (int, float, dec)
				stdArgLeft, stdArgRight, err := castNumberPairToCommonType(valLeftVolatile, valRightVolatile)
				if err != nil {
					return nil, fmt.Errorf("cannot perform binary comp op, incompatible arg types '%v(%T)' %v '%v(%T)' ", valLeftVolatile, valLeftVolatile, exp.Op, valRightVolatile, valRightVolatile)
				}
				switch stdArgLeft.(type) {
				case int64:
					eCtx.Value, err = eCtx.EvalBinaryIntToBool(stdArgLeft, exp.Op, stdArgRight)
					return eCtx.Value, err
				case float64:
					eCtx.Value, err = eCtx.EvalBinaryFloat64ToBool(stdArgLeft, exp.Op, stdArgRight)
					return eCtx.Value, err
				case decimal.Decimal:
					eCtx.Value, err = eCtx.EvalBinaryDecimal2ToBool(stdArgLeft, exp.Op, stdArgRight)
					return eCtx.Value, err
				default:
					return nil, fmt.Errorf("cannot perform binary comp op, unexpected std type '%v(%T)' %v '%v(%T)' ", valLeftVolatile, valLeftVolatile, exp.Op, valRightVolatile, valRightVolatile)
				}
			}
		} else {
			return nil, fmt.Errorf("cannot perform binary expression unknown op %v", exp.Op)
		}
	case *ast.BasicLit:
		switch exp.Kind {
		case token.INT:
			i, _ := strconv.ParseInt(exp.Value, 10, 64)
			eCtx.Value = i
			return i, nil
		case token.FLOAT:
			i, _ := strconv.ParseFloat(exp.Value, 64)
			eCtx.Value = i
			return i, nil
		case token.IDENT:
			return nil, fmt.Errorf("cannot evaluate expression %s of type token.IDENT", exp.Value)
		case token.STRING:
			eCtx.Value = exp.Value
			if exp.Value[0] == '"' {
				return strings.Trim(exp.Value, "\""), nil
			} else {
				return strings.Trim(exp.Value, "`"), nil
			}
		default:
			return nil, fmt.Errorf("cannot evaluate expression %s of type %v", exp.Value, exp.Kind)
		}
	case *ast.UnaryExpr:
		switch exp.Op {
		case token.NOT:
			var err error
			eCtx.Value, err = eCtx.EvalUnaryBoolNot(exp.X)
			return eCtx.Value, err
		case token.SUB:
			var err error
			eCtx.Value, err = eCtx.EvalUnaryMinus(exp.X)
			return eCtx.Value, err
		default:
			return nil, fmt.Errorf("cannot evaluate unary op %v, unkown op", exp.Op)
		}

	case *ast.Ident:
		if exp.Name == "true" {
			eCtx.Value = true
			return true, nil
		} else if exp.Name == "false" {
			eCtx.Value = false
			return false, nil
		} else {
			return nil, fmt.Errorf("cannot evaluate identifier %s", exp.Name)
		}

	case *ast.CallExpr:
		args := make([]interface{}, len(exp.Args))

		for i, v := range exp.Args {
			arg, err := eCtx.Eval(v)
			if err != nil {
				return nil, err
			}
			args[i] = arg
		}

		switch exp.Fun.(type) {
		case *ast.Ident:
			funcIdent, _ := exp.Fun.(*ast.Ident)
			var err error
			eCtx.Value, err = eCtx.EvalFunc(exp, funcIdent.Name, args)
			return eCtx.Value, err

		case *ast.SelectorExpr:
			expSel := exp.Fun.(*ast.SelectorExpr)
			switch expSel.X.(type) {
			case *ast.Ident:
				expIdent, _ := expSel.X.(*ast.Ident)
				var err error
				eCtx.Value, err = eCtx.EvalFunc(exp, fmt.Sprintf("%s.%s", expIdent.Name, expSel.Sel.Name), args)
				return eCtx.Value, err
			default:
				return nil, fmt.Errorf("cannot evaluate fun expression %v, unknown type of X: %T", expSel.X, expSel.X)
			}

		default:
			return nil, fmt.Errorf("cannot evaluate func call expression %v, unknown type of X: %T", exp.Fun, exp.Fun)
		}

	case *ast.SelectorExpr:
		switch exp.X.(type) {
		case *ast.Ident:
			objectIdent, _ := exp.X.(*ast.Ident)
			golangConst, ok := GolangConstants[fmt.Sprintf("%s.%s", objectIdent.Name, exp.Sel.Name)]
			if ok {
				eCtx.Value = golangConst
				return golangConst, nil
			}

			if eCtx.Vars == nil {
				return nil, fmt.Errorf("cannot evaluate expression '%s', no variables supplied to the context", objectIdent.Name)
			}

			objectAttributes, ok := (*eCtx.Vars)[objectIdent.Name]
			if !ok {
				return nil, fmt.Errorf("cannot evaluate expression '%s', variable not supplied, check table/alias name", objectIdent.Name)
			}

			val, ok := objectAttributes[exp.Sel.Name]
			if !ok {
				return nil, fmt.Errorf("cannot evaluate expression %s.%s, variable not supplied, check field name", objectIdent.Name, exp.Sel.Name)
			}
			eCtx.Value = val
			return val, nil
		default:
			return nil, fmt.Errorf("cannot evaluate selector expression %v, unknown type of X: %T", exp.X, exp.X)
		}

	default:
		return nil, fmt.Errorf("cannot evaluate generic expression %v of unknown type %T", exp, exp)
	}
}
