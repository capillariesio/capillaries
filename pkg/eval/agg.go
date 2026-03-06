package eval

import (
	"fmt"
	"go/ast"
	"math/big"
	"strings"

	"github.com/shopspring/decimal"
)

// IMPORTANT: please keep this eval core component TableFieldType- and custom function-agnostic.
// It should not be aware of things lice decimal2 or some math.iif() functions.

type AggFuncType string

const (
	AggStringAgg   AggFuncType = "string_agg"
	AggStringAggIf AggFuncType = "string_agg_if"
	AggSum         AggFuncType = "sum"
	AggSumIf       AggFuncType = "sum_if"
	AggCount       AggFuncType = "count"
	AggCountIf     AggFuncType = "count_if"
	AggAvg         AggFuncType = "avg"
	AggAvgIf       AggFuncType = "avg_if"
	AggMin         AggFuncType = "min"
	AggMinIf       AggFuncType = "min_if"
	AggMax         AggFuncType = "max"
	AggMaxIf       AggFuncType = "max_if"
	AggUnknown     AggFuncType = "unknown"
)

func StringToAggFunc(testString string) AggFuncType {
	switch testString {
	case string(AggStringAgg):
		return AggStringAgg
	case string(AggStringAggIf):
		return AggStringAggIf
	case string(AggSum):
		return AggSum
	case string(AggSumIf):
		return AggSumIf
	case string(AggCount):
		return AggCount
	case string(AggCountIf):
		return AggCountIf
	case string(AggAvg):
		return AggAvg
	case string(AggAvgIf):
		return AggAvgIf
	case string(AggMinIf):
		return AggMinIf
	case string(AggMin):
		return AggMin
	case string(AggMax):
		return AggMax
	case string(AggMaxIf):
		return AggMaxIf
	default:
		return AggUnknown
	}
}

type SumCollector struct {
	Int   int64
	Float float64
	Dec   decimal.Decimal
}
type AvgCollector struct {
	Int   *big.Int
	Float float64
	Dec   decimal.Decimal
	Count int64
}

type MinCollector struct {
	Int   int64
	Float float64
	Dec   decimal.Decimal
	Str   string
	Count int64
}

type MaxCollector struct {
	Int   int64
	Float float64
	Dec   decimal.Decimal
	Str   string
	Count int64
}

type StringAggCollector struct {
	Sb        strings.Builder
	Separator string
}

// Internal data type used for agg calculations only
type AggDataType string

const (
	AggTypeUnknown AggDataType = "unknown"
	AggTypeInt     AggDataType = "int"
	AggTypeFloat   AggDataType = "float"
	AggTypeDec     AggDataType = "decimal"
	AggTypeString  AggDataType = "string"
)

func (eCtx *EvalCtx) checkAgg(funcName string, callExp *ast.CallExpr, aggFunc AggFuncType) error {
	if eCtx.aggEnabled != AggFuncEnabled {
		return fmt.Errorf("cannot evaluate %s(), context aggregate not enabled (either agg function is not expected in this expression at all, or agg function is not in the root of the expression, like no sum(...)*x or sum(...)+y)", funcName)
	}
	if eCtx.aggCallExp != nil {
		if eCtx.aggCallExp != callExp {
			return fmt.Errorf("cannot evaluate more than one aggregate function in the expression, extra %s() found besides already used %s()", funcName, eCtx.aggFunc)
		}
	} else {
		eCtx.aggCallExp = callExp
		eCtx.aggFunc = aggFunc
	}
	return nil
}

func checkIf(funcName string, boolArg any) (bool, error) {
	switch typedArg := boolArg.(type) {
	case bool:
		return typedArg, nil

	default:
		return false, fmt.Errorf("cannot evaluate the if part of the agg function %s, unexpected argument %v of unsupported type %T", funcName, boolArg, boolArg)
	}
}

func (eCtx *EvalCtx) callAggStringAggInternal(funcName string, args []any, isApply bool) (any, error) {
	switch typedArg0 := args[0].(type) {
	case string:
		if isApply {
			if eCtx.stringAggCollector.Sb.Len() > 0 {
				eCtx.stringAggCollector.Sb.WriteString(eCtx.stringAggCollector.Separator)
			}
			eCtx.stringAggCollector.Sb.WriteString(typedArg0)
		}
		return eCtx.stringAggCollector.Sb.String(), nil

	default:
		return nil, fmt.Errorf("cannot evaluate %s(), unexpected argument %v of unsupported type %T", funcName, args[0], args[0])
	}
}

func (eCtx *EvalCtx) CallAggStringAgg(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "string_agg"
	if err := eCtx.checkAgg(funcName, callExp, AggStringAgg); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 2, len(args)); err != nil {
		return nil, err
	}

	return eCtx.callAggStringAggInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggStringAggIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "string_agg_if"
	if err := eCtx.checkAgg(funcName, callExp, AggStringAgg); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 3, len(args)); err != nil {
		return nil, err
	}
	isApply, err := checkIf(funcName, args[2])
	if err != nil {
		return nil, err
	}
	return eCtx.callAggStringAggInternal(funcName, args, isApply)
}

func (eCtx *EvalCtx) callAggSumInternal(funcName string, args []any, isApply bool) (any, error) {
	stdTypedArg, err := castNumberToStandardType(args[0])
	if err != nil {
		return nil, err
	}
	switch typedArg0 := stdTypedArg.(type) {
	case int64:
		if eCtx.aggType == AggTypeUnknown {
			eCtx.aggType = AggTypeInt
		} else if eCtx.aggType != AggTypeInt {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got int value %d", funcName, eCtx.aggType, typedArg0)
		}
		if isApply {
			eCtx.sumCollector.Int += typedArg0
		}
		return eCtx.sumCollector.Int, nil

	case float64:
		if eCtx.aggType == AggTypeUnknown {
			eCtx.aggType = AggTypeFloat
		} else if eCtx.aggType != AggTypeFloat {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got float value %f", funcName, eCtx.aggType, typedArg0)
		}
		if isApply {
			eCtx.sumCollector.Float += typedArg0
		}
		return eCtx.sumCollector.Float, nil

	case decimal.Decimal:
		if eCtx.aggType == AggTypeUnknown {
			eCtx.aggType = AggTypeDec
		} else if eCtx.aggType != AggTypeDec {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got decimal value %s", funcName, eCtx.aggType, typedArg0.String())
		}
		if isApply {
			eCtx.sumCollector.Dec = eCtx.sumCollector.Dec.Add(typedArg0)
		}
		return eCtx.sumCollector.Dec, nil

	default:
		return nil, fmt.Errorf("cannot evaluate %s(), unexpected argument %v of unsupported type %T", funcName, args[0], args[0])
	}
}

func (eCtx *EvalCtx) CallAggSum(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "sum"
	if err := eCtx.checkAgg(funcName, callExp, AggSum); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 1, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggSumInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggSumIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "sum_if"
	if err := eCtx.checkAgg(funcName, callExp, AggSum); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 2, len(args)); err != nil {
		return nil, err
	}
	isApply, err := checkIf(funcName, args[1])
	if err != nil {
		return nil, err
	}
	return eCtx.callAggSumInternal(funcName, args, isApply)
}

func (eCtx *EvalCtx) callAggAvgInternal(funcName string, args []any, isApply bool) (any, error) {
	stdTypedArg, err := castNumberToStandardType(args[0])
	if err != nil {
		return nil, err
	}
	switch typedArg0 := stdTypedArg.(type) {
	case int64:
		if eCtx.aggType == AggTypeUnknown {
			eCtx.aggType = AggTypeInt
		} else if eCtx.aggType != AggTypeInt {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got int value %d", funcName, eCtx.aggType, typedArg0)
		}
		if isApply {
			eCtx.avgCollector.Int.Add(eCtx.avgCollector.Int, big.NewInt(typedArg0))
			eCtx.avgCollector.Count++
		}
		if eCtx.avgCollector.Count > 0 {
			bigintVal := big.NewInt(0).Div(eCtx.avgCollector.Int, big.NewInt(eCtx.avgCollector.Count))
			return bigintVal.Int64(), nil
		}
		return int64(0), nil

	case float64:
		if eCtx.aggType == AggTypeUnknown {
			eCtx.aggType = AggTypeFloat
		} else if eCtx.aggType != AggTypeFloat {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got float value %f", funcName, eCtx.aggType, typedArg0)
		}
		if isApply {
			eCtx.avgCollector.Float += typedArg0
			eCtx.avgCollector.Count++
		}
		if eCtx.avgCollector.Count > 0 {
			return eCtx.avgCollector.Float / float64(eCtx.avgCollector.Count), nil
		}
		return float64(0.0), nil

	case decimal.Decimal:
		if eCtx.aggType == AggTypeUnknown {
			eCtx.aggType = AggTypeDec
		} else if eCtx.aggType != AggTypeDec {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got decimal value %s", funcName, eCtx.aggType, typedArg0.String())
		}
		if isApply {
			eCtx.avgCollector.Dec = eCtx.avgCollector.Dec.Add(typedArg0)
			eCtx.avgCollector.Count++
		}
		if eCtx.avgCollector.Count > 0 {
			// Round(2) when needed
			val := eCtx.avgCollector.Dec.Div(decimal.NewFromInt(eCtx.avgCollector.Count))
			if eCtx.roundDec >= 0 {
				val = val.Round(eCtx.roundDec)
			}
			return val, nil
		}
		return defaultDecimal(), nil

	default:
		return nil, fmt.Errorf("cannot evaluate %s(), unexpected argument %v of unsupported type %T", funcName, args[0], args[0])
	}
}

func (eCtx *EvalCtx) CallAggAvg(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "avg"
	if err := eCtx.checkAgg(funcName, callExp, AggAvg); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 1, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggAvgInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggAvgIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "avg_if"
	if err := eCtx.checkAgg(funcName, callExp, AggAvg); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 2, len(args)); err != nil {
		return nil, err
	}
	isApply, err := checkIf(funcName, args[1])
	if err != nil {
		return nil, err
	}
	return eCtx.callAggAvgInternal(funcName, args, isApply)
}

func (eCtx *EvalCtx) callAggCountInternal(isApply bool) (any, error) {
	if isApply {
		eCtx.count++
	}
	return eCtx.count, nil
}

func (eCtx *EvalCtx) CallAggCount(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "count"
	if err := eCtx.checkAgg(funcName, callExp, AggCount); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 0, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggCountInternal(true)
}

func (eCtx *EvalCtx) CallAggCountIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "count_if"
	if err := eCtx.checkAgg(funcName, callExp, AggCount); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 1, len(args)); err != nil {
		return nil, err
	}
	isApply, err := checkIf(funcName, args[0])
	if err != nil {
		return nil, err
	}
	return eCtx.callAggCountInternal(isApply)
}

func (eCtx *EvalCtx) callAggMinInternal(funcName string, args []any, isApply bool) (any, error) {
	switch typedArg0 := args[0].(type) {
	case string:
		if eCtx.aggType == AggTypeUnknown {
			eCtx.aggType = AggTypeString
		} else if eCtx.aggType != AggTypeString {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got string value %s", funcName, eCtx.aggType, typedArg0)
		}
		if isApply {
			eCtx.minCollector.Count++
			if len(eCtx.minCollector.Str) == 0 || typedArg0 < eCtx.minCollector.Str {
				eCtx.minCollector.Str = typedArg0
			}
		}
		return eCtx.minCollector.Str, nil

	default:
		stdTypedArg0, err := castNumberToStandardType(args[0])
		if err != nil {
			return nil, err
		}
		switch typedNumberArg0 := stdTypedArg0.(type) {
		case int64:
			if eCtx.aggType == AggTypeUnknown {
				eCtx.aggType = AggTypeInt
			} else if eCtx.aggType != AggTypeInt {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got int value %d", funcName, eCtx.aggType, typedNumberArg0)
			}
			if isApply {
				eCtx.minCollector.Count++
				if typedNumberArg0 < eCtx.minCollector.Int {
					eCtx.minCollector.Int = typedNumberArg0
				}
			}
			return eCtx.minCollector.Int, nil

		case float64:
			if eCtx.aggType == AggTypeUnknown {
				eCtx.aggType = AggTypeFloat
			} else if eCtx.aggType != AggTypeFloat {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got float value %f", funcName, eCtx.aggType, typedNumberArg0)
			}
			if isApply {
				eCtx.minCollector.Count++
				if typedNumberArg0 < eCtx.minCollector.Float {
					eCtx.minCollector.Float = typedNumberArg0
				}
			}
			return eCtx.minCollector.Float, nil

		case decimal.Decimal:
			if eCtx.aggType == AggTypeUnknown {
				eCtx.aggType = AggTypeDec
			} else if eCtx.aggType != AggTypeDec {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got decimal value %s", funcName, eCtx.aggType, typedNumberArg0.String())
			}
			if isApply {
				eCtx.minCollector.Count++
				if typedNumberArg0.LessThan(eCtx.minCollector.Dec) {
					eCtx.minCollector.Dec = typedNumberArg0
				}
			}
			return eCtx.minCollector.Dec, nil

		default:
			return nil, fmt.Errorf("cannot evaluate %s(), unexpected argument %v of unsupported type %T", funcName, args[0], args[0])
		}
	}
}

func (eCtx *EvalCtx) CallAggMin(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "min"
	if err := eCtx.checkAgg(funcName, callExp, AggMin); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 1, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggMinInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggMinIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "min_if"
	if err := eCtx.checkAgg(funcName, callExp, AggMin); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 2, len(args)); err != nil {
		return nil, err
	}
	isApply, err := checkIf(funcName, args[1])
	if err != nil {
		return nil, err
	}
	return eCtx.callAggMinInternal(funcName, args, isApply)
}

func (eCtx *EvalCtx) callAggMaxInternal(funcName string, args []any, isApply bool) (any, error) {
	switch typedArg0 := args[0].(type) {
	case string:
		if eCtx.aggType == AggTypeUnknown {
			eCtx.aggType = AggTypeString
		} else if eCtx.aggType != AggTypeString {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got string value %s", funcName, eCtx.aggType, typedArg0)
		}
		if isApply {
			eCtx.maxCollector.Count++
			if len(eCtx.maxCollector.Str) == 0 || typedArg0 > eCtx.maxCollector.Str {
				eCtx.maxCollector.Str = typedArg0
			}
		}
		return eCtx.maxCollector.Str, nil
	default:
		stdTypedNumberArg0, err := castNumberToStandardType(args[0])
		if err != nil {
			return nil, err
		}
		switch typedNumberArg0 := stdTypedNumberArg0.(type) {
		case int64:
			if eCtx.aggType == AggTypeUnknown {
				eCtx.aggType = AggTypeInt
			} else if eCtx.aggType != AggTypeInt {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got int value %d", funcName, eCtx.aggType, typedNumberArg0)
			}
			if isApply {
				eCtx.maxCollector.Count++
				if typedNumberArg0 > eCtx.maxCollector.Int {
					eCtx.maxCollector.Int = typedNumberArg0
				}
			}
			return eCtx.maxCollector.Int, nil

		case float64:
			if eCtx.aggType == AggTypeUnknown {
				eCtx.aggType = AggTypeFloat
			} else if eCtx.aggType != AggTypeFloat {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got float value %f", funcName, eCtx.aggType, typedNumberArg0)
			}
			if isApply {
				eCtx.maxCollector.Count++
				if typedNumberArg0 > eCtx.maxCollector.Float {
					eCtx.maxCollector.Float = typedNumberArg0
				}
			}
			return eCtx.maxCollector.Float, nil

		case decimal.Decimal:
			if eCtx.aggType == AggTypeUnknown {
				eCtx.aggType = AggTypeDec
			} else if eCtx.aggType != AggTypeDec {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got decimal value %s", funcName, eCtx.aggType, typedNumberArg0.String())
			}
			if isApply {
				eCtx.maxCollector.Count++
				if typedNumberArg0.GreaterThan(eCtx.maxCollector.Dec) {
					eCtx.maxCollector.Dec = typedNumberArg0
				}
			}
			return eCtx.maxCollector.Dec, nil

		default:
			return nil, fmt.Errorf("cannot evaluate %s(), unexpected argument %v of unsupported type %T", funcName, args[0], args[0])
		}
	}
}

func (eCtx *EvalCtx) CallAggMax(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "max"
	if err := eCtx.checkAgg(funcName, callExp, AggMax); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 1, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggMaxInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggMaxIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "max_if"
	if err := eCtx.checkAgg(funcName, callExp, AggMax); err != nil {
		return nil, err
	}
	if err := CheckArgs(funcName, 2, len(args)); err != nil {
		return nil, err
	}
	isApply, err := checkIf(funcName, args[1])
	if err != nil {
		return nil, err
	}
	return eCtx.callAggMaxInternal(funcName, args, isApply)
}
