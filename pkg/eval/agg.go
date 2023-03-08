package eval

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/shopspring/decimal"
)

type AggFuncType string

const (
	AggStringAgg AggFuncType = "string_agg"
	AggSum       AggFuncType = "sum"
	AggCount     AggFuncType = "count"
	AggAvg       AggFuncType = "avg"
	AggMin       AggFuncType = "min"
	AggMax       AggFuncType = "max"
	AggUnknown   AggFuncType = "unknown"
)

func StringToAggFunc(testString string) AggFuncType {
	switch testString {
	case string(AggStringAgg):
		return AggStringAgg
	case string(AggSum):
		return AggSum
	case string(AggCount):
		return AggCount
	case string(AggAvg):
		return AggAvg
	case string(AggMin):
		return AggMin
	case string(AggMax):
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
	if eCtx.AggCallExp != nil {
		if eCtx.AggCallExp != callExp {
			return fmt.Errorf("cannot evaluate more than one aggregate functions in the expression, extra %s() found besides %s()", funcName, eCtx.AggFunc)
		}
	} else {
		eCtx.AggCallExp = callExp
		eCtx.AggFunc = aggFunc
	}
	return nil
}

func (eCtx *EvalCtx) CallAggStringAgg(callExp *ast.CallExpr, args []interface{}) (interface{}, error) {
	if err := eCtx.checkAgg("string_agg", callExp, AggSum); err != nil {
		return nil, err
	}
	if err := checkArgs("string_agg", 2, len(args)); err != nil {
		return nil, err
	}

	switch typedArg0 := args[0].(type) {
	case string:
		if eCtx.AggEnabled != AggFuncEnabled {
			return nil, fmt.Errorf("cannot evaluate string_agg(string,separator), context aggregate not enabled")
		}
		if eCtx.StringAgg.Sb.Len() > 0 {
			eCtx.StringAgg.Sb.WriteString(eCtx.StringAgg.Separator)
		}
		eCtx.StringAgg.Sb.WriteString(typedArg0)
		return eCtx.StringAgg.Sb.String(), nil

	default:
		return nil, fmt.Errorf("cannot evaluate string_agg(), argument %v of unsupported type %T", args[0], args[0])
	}
}

func (eCtx *EvalCtx) CallAggSum(callExp *ast.CallExpr, args []interface{}) (interface{}, error) {
	if err := eCtx.checkAgg("sum", callExp, AggSum); err != nil {
		return nil, err
	}
	if err := checkArgs("sum", 1, len(args)); err != nil {
		return nil, err
	}
	stdTypedArg, err := castNumberToStandardType(args[0])
	if err != nil {
		return nil, err
	}
	switch typedArg0 := stdTypedArg.(type) {
	case int64:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeInt
		} else if eCtx.AggType != AggTypeInt {
			return nil, fmt.Errorf("cannot evaluate sum(), it started with type %s, now got int value %d", eCtx.AggType, typedArg0)
		}
		if eCtx.AggEnabled != AggFuncEnabled {
			return nil, fmt.Errorf("cannot evaluate sum(int64), context aggregate not enabled")
		}
		eCtx.Sum.Int += typedArg0
		return eCtx.Sum.Int, nil

	case float64:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeFloat
		} else if eCtx.AggType != AggTypeFloat {
			return nil, fmt.Errorf("cannot evaluate sum(), it started with type %s, now got float value %f", eCtx.AggType, typedArg0)
		}
		if eCtx.AggEnabled != AggFuncEnabled {
			return nil, fmt.Errorf("cannot evaluate sum(float64), context aggregate not enabled")
		}
		eCtx.Sum.Float += typedArg0
		return eCtx.Sum.Float, nil

	case decimal.Decimal:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeDec
		} else if eCtx.AggType != AggTypeDec {
			return nil, fmt.Errorf("cannot evaluate sum(), it started with type %s, now got decimal value %s", eCtx.AggType, typedArg0.String())
		}
		if eCtx.AggEnabled != AggFuncEnabled {
			return nil, fmt.Errorf("cannot evaluate sum(decimal2), context aggregate not enabled")
		}
		eCtx.Sum.Dec = eCtx.Sum.Dec.Add(typedArg0)
		return eCtx.Sum.Dec, nil

	default:
		return nil, fmt.Errorf("cannot evaluate sum(), argument %v of unsupported type %T", args[0], args[0])
	}
}

func (eCtx *EvalCtx) CallAggAvg(callExp *ast.CallExpr, args []interface{}) (interface{}, error) {
	if err := eCtx.checkAgg("avg", callExp, AggAvg); err != nil {
		return nil, err
	}
	if err := checkArgs("avg", 1, len(args)); err != nil {
		return nil, err
	}
	stdTypedArg, err := castNumberToStandardType(args[0])
	if err != nil {
		return nil, err
	}
	switch typedArg0 := stdTypedArg.(type) {
	case int64:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeInt
		} else if eCtx.AggType != AggTypeInt {
			return nil, fmt.Errorf("cannot evaluate avg(), it started with type %s, now got int value %d", eCtx.AggType, typedArg0)
		}
		if eCtx.AggEnabled != AggFuncEnabled {
			return nil, fmt.Errorf("cannot evaluate avg(int64), context aggregate not enabled")
		}
		eCtx.Avg.Int += typedArg0
		eCtx.Avg.Count++
		return eCtx.Avg.Int / eCtx.Avg.Count, nil

	case float64:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeFloat
		} else if eCtx.AggType != AggTypeFloat {
			return nil, fmt.Errorf("cannot evaluate avg(), it started with type %s, now got float value %f", eCtx.AggType, typedArg0)
		}
		if eCtx.AggEnabled != AggFuncEnabled {
			return nil, fmt.Errorf("cannot evaluate avg(float64), context aggregate not enabled")
		}
		eCtx.Avg.Float += typedArg0
		eCtx.Avg.Count++
		return eCtx.Avg.Float / float64(eCtx.Avg.Count), nil

	case decimal.Decimal:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeDec
		} else if eCtx.AggType != AggTypeDec {
			return nil, fmt.Errorf("cannot evaluate avg(), it started with type %s, now got decimal value %s", eCtx.AggType, typedArg0.String())
		}
		if eCtx.AggEnabled != AggFuncEnabled {
			return nil, fmt.Errorf("cannot evaluate avg(decimal2), context aggregate not enabled")
		}
		eCtx.Avg.Dec = eCtx.Avg.Dec.Add(typedArg0)
		eCtx.Avg.Count++
		return eCtx.Avg.Dec.Div(decimal.NewFromInt(eCtx.Avg.Count)).Round(2), nil

	default:
		return nil, fmt.Errorf("cannot evaluate avg(), argument %v of unsupported type %T", args[0], args[0])
	}
}

func (eCtx *EvalCtx) CallAggCount(callExp *ast.CallExpr, args []interface{}) (interface{}, error) {
	if err := eCtx.checkAgg("count", callExp, AggCount); err != nil {
		return nil, err
	}
	if err := checkArgs("count", 0, len(args)); err != nil {
		return nil, err
	}
	if eCtx.AggEnabled != AggFuncEnabled {
		return nil, fmt.Errorf("cannot evaluate count(), context aggregate not enabled")
	}
	eCtx.Count++
	return eCtx.Count, nil
}

func (eCtx *EvalCtx) CallAggMin(callExp *ast.CallExpr, args []interface{}) (interface{}, error) {
	if err := eCtx.checkAgg("min", callExp, AggMin); err != nil {
		return nil, err
	}
	if err := checkArgs("min", 1, len(args)); err != nil {
		return nil, err
	}

	switch typedArg0 := args[0].(type) {
	case string:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeString
		} else if eCtx.AggType != AggTypeString {
			return nil, fmt.Errorf("cannot evaluate min(), it started with type %s, now got string value %s", eCtx.AggType, typedArg0)
		}
		if eCtx.AggEnabled != AggFuncEnabled {
			return nil, fmt.Errorf("cannot evaluate min(string), context aggregate not enabled")
		}
		eCtx.Min.Count++
		if len(eCtx.Min.Str) == 0 || typedArg0 < eCtx.Min.Str {
			eCtx.Min.Str = typedArg0
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
				return nil, fmt.Errorf("cannot evaluate min(), it started with type %s, now got int value %d", eCtx.AggType, typedNumberArg0)
			}
			if eCtx.AggEnabled != AggFuncEnabled {
				return nil, fmt.Errorf("cannot evaluate min(int64), context aggregate not enabled")
			}
			eCtx.Min.Count++
			if typedNumberArg0 < eCtx.Min.Int {
				eCtx.Min.Int = typedNumberArg0
			}
			return eCtx.Min.Int, nil

		case float64:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeFloat
			} else if eCtx.AggType != AggTypeFloat {
				return nil, fmt.Errorf("cannot evaluate min(), it started with type %s, now got float value %f", eCtx.AggType, typedNumberArg0)
			}
			if eCtx.AggEnabled != AggFuncEnabled {
				return nil, fmt.Errorf("cannot evaluate min(float64), context aggregate not enabled")
			}
			eCtx.Min.Count++
			if typedNumberArg0 < eCtx.Min.Float {
				eCtx.Min.Float = typedNumberArg0
			}
			return eCtx.Min.Float, nil

		case decimal.Decimal:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeDec
			} else if eCtx.AggType != AggTypeDec {
				return nil, fmt.Errorf("cannot evaluate min(), it started with type %s, now got decimal value %s", eCtx.AggType, typedNumberArg0.String())
			}
			if eCtx.AggEnabled != AggFuncEnabled {
				return nil, fmt.Errorf("cannot evaluate min(decimal2), context aggregate not enabled")
			}
			eCtx.Min.Count++
			if typedNumberArg0.LessThan(eCtx.Min.Dec) {
				eCtx.Min.Dec = typedNumberArg0
			}
			return eCtx.Min.Dec, nil

		default:
			return nil, fmt.Errorf("cannot evaluate max(), argument %v of unsupported type %T", args[0], args[0])
		}
	}
}

func (eCtx *EvalCtx) CallAggMax(callExp *ast.CallExpr, args []interface{}) (interface{}, error) {
	if err := eCtx.checkAgg("max", callExp, AggMax); err != nil {
		return nil, err
	}
	if err := checkArgs("max", 1, len(args)); err != nil {
		return nil, err
	}

	switch typedArg0 := args[0].(type) {
	case string:
		if eCtx.AggType == AggTypeUnknown {
			eCtx.AggType = AggTypeString
		} else if eCtx.AggType != AggTypeString {
			return nil, fmt.Errorf("cannot evaluate max(), it started with type %s, now got string value %s", eCtx.AggType, typedArg0)
		}
		if eCtx.AggEnabled != AggFuncEnabled {
			return nil, fmt.Errorf("cannot evaluate max(string), context aggregate not enabled")
		}
		eCtx.Max.Count++
		if len(eCtx.Max.Str) == 0 || typedArg0 > eCtx.Max.Str {
			eCtx.Max.Str = typedArg0
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
				return nil, fmt.Errorf("cannot evaluate max(), it started with type %s, now got int value %d", eCtx.AggType, typedNumberArg0)
			}
			if eCtx.AggEnabled != AggFuncEnabled {
				return nil, fmt.Errorf("cannot evaluate max(int64), context aggregate not enabled")
			}
			eCtx.Max.Count++
			if typedNumberArg0 > eCtx.Max.Int {
				eCtx.Max.Int = typedNumberArg0
			}
			return eCtx.Max.Int, nil

		case float64:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeFloat
			} else if eCtx.AggType != AggTypeFloat {
				return nil, fmt.Errorf("cannot evaluate max(), it started with type %s, now got float value %f", eCtx.AggType, typedNumberArg0)
			}
			if eCtx.AggEnabled != AggFuncEnabled {
				return nil, fmt.Errorf("cannot evaluate max(float64), context aggregate not enabled")
			}
			eCtx.Max.Count++
			if typedNumberArg0 > eCtx.Max.Float {
				eCtx.Max.Float = typedNumberArg0
			}
			return eCtx.Max.Float, nil

		case decimal.Decimal:
			if eCtx.AggType == AggTypeUnknown {
				eCtx.AggType = AggTypeDec
			} else if eCtx.AggType != AggTypeDec {
				return nil, fmt.Errorf("cannot evaluate max(), it started with type %s, now got decimal value %s", eCtx.AggType, typedNumberArg0.String())
			}
			if eCtx.AggEnabled != AggFuncEnabled {
				return nil, fmt.Errorf("cannot evaluate max(decimal2), context aggregate not enabled")
			}
			eCtx.Max.Count++
			if typedNumberArg0.GreaterThan(eCtx.Max.Dec) {
				eCtx.Max.Dec = typedNumberArg0
			}
			return eCtx.Max.Dec, nil

		default:
			return nil, fmt.Errorf("cannot evaluate max(), argument %v of unsupported type %T", args[0], args[0])
		}
	}
}
