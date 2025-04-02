package eval

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func DetectRootAggFunc(exp ast.Expr) (string, AggEnabledType, AggFuncType, []ast.Expr) {
	if callExp, ok := exp.(*ast.CallExpr); ok {
		funExp := callExp.Fun
		if funIdentExp, ok := funExp.(*ast.Ident); ok {
			if StringToAggFunc(funIdentExp.Name) != AggUnknown {
				return funIdentExp.Name, AggFuncEnabled, StringToAggFunc(funIdentExp.Name), callExp.Args
			}
		}
	}
	return "", AggFuncDisabled, AggUnknown, nil
}

func GetAggStringSeparator(funcName string, aggFuncArgs []ast.Expr) (string, error) {
	if funcName == string(AggStringAgg) && len(aggFuncArgs) != 2 {
		return "", fmt.Errorf("%s must have two parameters", funcName)
	} else if funcName == string(AggStringAggIf) && len(aggFuncArgs) != 3 {
		return "", fmt.Errorf("%s must have three parameters", funcName)
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
