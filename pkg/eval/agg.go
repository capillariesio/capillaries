package eval

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/shopspring/decimal"
)

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
		return AggStringAgg
	case string(AggSum):
		return AggSum
	case string(AggSumIf):
		return AggSum
	case string(AggCount):
		return AggCount
	case string(AggCountIf):
		return AggCount
	case string(AggAvg):
		return AggAvg
	case string(AggAvgIf):
		return AggAvg
	case string(AggMin):
		return AggMin
	case string(AggMinIf):
		return AggMin
	case string(AggMax):
		return AggMax
	case string(AggMaxIf):
		return AggMax
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
	Int   int64
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

type AggDataType string

const (
	AggTypeUnknown AggDataType = "unknown"
	AggTypeInt     AggDataType = "int"
	AggTypeFloat   AggDataType = "float"
	AggTypeDec     AggDataType = "decimal"
	AggTypeString  AggDataType = "string"
)

func (eCtx *EvalCtx) checkAgg(funcName string, callExp *ast.CallExpr, aggFunc AggFuncType) error {
	if eCtx.AggEnabled != AggFuncEnabled {
		return fmt.Errorf("cannot evaluate %s(), context aggregate not enabled", funcName)
	}
	if eCtx.AggCallExp != nil {
		if eCtx.AggCallExp != callExp {
			return fmt.Errorf("cannot evaluate more than one aggregate function in the expression, extra %s() found besides already used %s()", funcName, eCtx.AggFunc)
		}
	} else {
		eCtx.AggCallExp = callExp
		eCtx.AggFunc = aggFunc
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
			if eCtx.StringAgg.Sb.Len() > 0 {
				eCtx.StringAgg.Sb.WriteString(eCtx.StringAgg.Separator)
			}
			eCtx.StringAgg.Sb.WriteString(typedArg0)
		}
		return eCtx.StringAgg.Sb.String(), nil

	default:
		return nil, fmt.Errorf("cannot evaluate %s(), unexpected argument %v of unsupported type %T", funcName, args[0], args[0])
	}
}

func (eCtx *EvalCtx) CallAggStringAgg(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "string_agg"
	if err := eCtx.checkAgg(funcName, callExp, AggStringAgg); err != nil {
		return nil, err
	}
	if err := checkArgs(funcName, 2, len(args)); err != nil {
		return nil, err
	}

	return eCtx.callAggStringAggInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggStringAggIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "string_agg_if"
	if err := eCtx.checkAgg(funcName, callExp, AggStringAgg); err != nil {
		return nil, err
	}
	if err := checkArgs(funcName, 3, len(args)); err != nil {
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
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeInt
		} else if eCtx.AggType != AggTypeInt {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got int value %d", funcName, eCtx.AggType, typedArg0)
		}
		if isApply {
			eCtx.Sum.Int += typedArg0
		}
		return eCtx.Sum.Int, nil

	case float64:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeFloat
		} else if eCtx.AggType != AggTypeFloat {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got float value %f", funcName, eCtx.AggType, typedArg0)
		}
		if isApply {
			eCtx.Sum.Float += typedArg0
		}
		return eCtx.Sum.Float, nil

	case decimal.Decimal:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeDec
		} else if eCtx.AggType != AggTypeDec {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got decimal value %s", funcName, eCtx.AggType, typedArg0.String())
		}
		if isApply {
			eCtx.Sum.Dec = eCtx.Sum.Dec.Add(typedArg0)
		}
		return eCtx.Sum.Dec, nil

	default:
		return nil, fmt.Errorf("cannot evaluate %s(), unexpected argument %v of unsupported type %T", funcName, args[0], args[0])
	}
}

func (eCtx *EvalCtx) CallAggSum(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "sum"
	if err := eCtx.checkAgg(funcName, callExp, AggSum); err != nil {
		return nil, err
	}
	if err := checkArgs(funcName, 1, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggSumInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggSumIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "sum_if"
	if err := eCtx.checkAgg(funcName, callExp, AggSum); err != nil {
		return nil, err
	}
	if err := checkArgs(funcName, 2, len(args)); err != nil {
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
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeInt
		} else if eCtx.AggType != AggTypeInt {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got int value %d", funcName, eCtx.AggType, typedArg0)
		}
		if isApply {
			eCtx.Avg.Int += typedArg0
			eCtx.Avg.Count++
		}
		if eCtx.Avg.Count > 0 {
			return eCtx.Avg.Int / eCtx.Avg.Count, nil
		}
		return int64(0), nil

	case float64:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeFloat
		} else if eCtx.AggType != AggTypeFloat {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got float value %f", funcName, eCtx.AggType, typedArg0)
		}
		if isApply {
			eCtx.Avg.Float += typedArg0
			eCtx.Avg.Count++
		}
		if eCtx.Avg.Count > 0 {
			return eCtx.Avg.Float / float64(eCtx.Avg.Count), nil
		}
		return float64(0.0), nil

	case decimal.Decimal:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeDec
		} else if eCtx.AggType != AggTypeDec {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got decimal value %s", funcName, eCtx.AggType, typedArg0.String())
		}
		if isApply {
			eCtx.Avg.Dec = eCtx.Avg.Dec.Add(typedArg0)
			eCtx.Avg.Count++
		}
		if eCtx.Avg.Count > 0 {
			return eCtx.Avg.Dec.Div(decimal.NewFromInt(eCtx.Avg.Count)).Round(2), nil
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
	if err := checkArgs(funcName, 1, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggAvgInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggAvgIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "avg_if"
	if err := eCtx.checkAgg(funcName, callExp, AggAvg); err != nil {
		return nil, err
	}
	if err := checkArgs(funcName, 2, len(args)); err != nil {
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
		eCtx.Count++
	}
	return eCtx.Count, nil
}

func (eCtx *EvalCtx) CallAggCount(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "count"
	if err := eCtx.checkAgg(funcName, callExp, AggCount); err != nil {
		return nil, err
	}
	if err := checkArgs(funcName, 0, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggCountInternal(true)
}

func (eCtx *EvalCtx) CallAggCountIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "count_if"
	if err := eCtx.checkAgg(funcName, callExp, AggCount); err != nil {
		return nil, err
	}
	if err := checkArgs(funcName, 1, len(args)); err != nil {
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
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeString
		} else if eCtx.AggType != AggTypeString {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got string value %s", funcName, eCtx.AggType, typedArg0)
		}
		if isApply {
			eCtx.Min.Count++
			if len(eCtx.Min.Str) == 0 || typedArg0 < eCtx.Min.Str {
				eCtx.Min.Str = typedArg0
			}
		}
		return eCtx.Min.Str, nil

	default:
		stdTypedArg0, err := castNumberToStandardType(args[0])
		if err != nil {
			return nil, err
		}
		switch typedNumberArg0 := stdTypedArg0.(type) {
		case int64:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeInt
			} else if eCtx.AggType != AggTypeInt {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got int value %d", funcName, eCtx.AggType, typedNumberArg0)
			}
			if isApply {
				eCtx.Min.Count++
				if typedNumberArg0 < eCtx.Min.Int {
					eCtx.Min.Int = typedNumberArg0
				}
			}
			return eCtx.Min.Int, nil

		case float64:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeFloat
			} else if eCtx.AggType != AggTypeFloat {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got float value %f", funcName, eCtx.AggType, typedNumberArg0)
			}
			if isApply {
				eCtx.Min.Count++
				if typedNumberArg0 < eCtx.Min.Float {
					eCtx.Min.Float = typedNumberArg0
				}
			}
			return eCtx.Min.Float, nil

		case decimal.Decimal:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeDec
			} else if eCtx.AggType != AggTypeDec {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got decimal value %s", funcName, eCtx.AggType, typedNumberArg0.String())
			}
			if isApply {
				eCtx.Min.Count++
				if typedNumberArg0.LessThan(eCtx.Min.Dec) {
					eCtx.Min.Dec = typedNumberArg0
				}
			}
			return eCtx.Min.Dec, nil

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
	if err := checkArgs(funcName, 1, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggMinInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggMinIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "min_if"
	if err := eCtx.checkAgg(funcName, callExp, AggMin); err != nil {
		return nil, err
	}
	if err := checkArgs(funcName, 2, len(args)); err != nil {
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
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeString
		} else if eCtx.AggType != AggTypeString {
			return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got string value %s", funcName, eCtx.AggType, typedArg0)
		}
		if isApply {
			eCtx.Max.Count++
			if len(eCtx.Max.Str) == 0 || typedArg0 > eCtx.Max.Str {
				eCtx.Max.Str = typedArg0
			}
		}
		return eCtx.Max.Str, nil
	default:
		stdTypedNumberArg0, err := castNumberToStandardType(args[0])
		if err != nil {
			return nil, err
		}
		switch typedNumberArg0 := stdTypedNumberArg0.(type) {
		case int64:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeInt
			} else if eCtx.AggType != AggTypeInt {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got int value %d", funcName, eCtx.AggType, typedNumberArg0)
			}
			if isApply {
				eCtx.Max.Count++
				if typedNumberArg0 > eCtx.Max.Int {
					eCtx.Max.Int = typedNumberArg0
				}
			}
			return eCtx.Max.Int, nil

		case float64:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeFloat
			} else if eCtx.AggType != AggTypeFloat {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got float value %f", funcName, eCtx.AggType, typedNumberArg0)
			}
			if isApply {
				eCtx.Max.Count++
				if typedNumberArg0 > eCtx.Max.Float {
					eCtx.Max.Float = typedNumberArg0
				}
			}
			return eCtx.Max.Float, nil

		case decimal.Decimal:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeDec
			} else if eCtx.AggType != AggTypeDec {
				return nil, fmt.Errorf("cannot evaluate %s(), it started with type %s, now got decimal value %s", funcName, eCtx.AggType, typedNumberArg0.String())
			}
			if isApply {
				eCtx.Max.Count++
				if typedNumberArg0.GreaterThan(eCtx.Max.Dec) {
					eCtx.Max.Dec = typedNumberArg0
				}
			}
			return eCtx.Max.Dec, nil

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
	if err := checkArgs(funcName, 1, len(args)); err != nil {
		return nil, err
	}
	return eCtx.callAggMaxInternal(funcName, args, true)
}

func (eCtx *EvalCtx) CallAggMaxIf(callExp *ast.CallExpr, args []any) (any, error) {
	funcName := "max_if"
	if err := eCtx.checkAgg(funcName, callExp, AggMax); err != nil {
		return nil, err
	}
	if err := checkArgs(funcName, 2, len(args)); err != nil {
		return nil, err
	}
	isApply, err := checkIf(funcName, args[1])
	if err != nil {
		return nil, err
	}
	return eCtx.callAggMaxInternal(funcName, args, isApply)
}
