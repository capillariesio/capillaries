package eval

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func DetectRootAggFunc(exp ast.Expr) (AggEnabledType, AggFuncType, []ast.Expr) {
	if callExp, ok := exp.(*ast.CallExpr); ok {
		funExp := callExp.Fun
		if funIdentExp, ok := funExp.(*ast.Ident); ok {
			if StringToAggFunc(funIdentExp.Name) != AggUnknown {
				return AggFuncEnabled, StringToAggFunc(funIdentExp.Name), callExp.Args
			}
		}
	}
	return AggFuncDisabled, AggUnknown, nil
}

func GetAggStringSeparator(aggFuncArgs []ast.Expr) (string, error) {
	if len(aggFuncArgs) < 2 {
		return "", fmt.Errorf("agg_string must have two parameters")
	}
	switch separatorExpTyped := aggFuncArgs[1].(type) {
	case *ast.BasicLit:
		switch separatorExpTyped.Kind {
		case token.STRING:
			return strings.Trim(separatorExpTyped.Value, "\""), nil
		default:
			return "", fmt.Errorf("agg_string second parameter must be a constant string")
		}
	default:
		return "", fmt.Errorf("agg_string second parameter must be a basic literal")
	}
}
