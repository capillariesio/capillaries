package eval

import "go/ast"

func StringSliceToStringSet(slice []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, s := range slice {
		m[s] = struct{}{}
	}
	return m
}

func StringSetToStringSlice(m map[string]struct{}) []string {
	slice := make([]string, len(m))
	i := 0
	for k, _ := range m {
		slice[i] = k
		i++
	}
	return slice
}

func IsRootAggFunc(exp ast.Expr) bool {
	isAgg := false
	if callExp, ok := exp.(*ast.CallExpr); ok {
		funName := callExp.Fun.(*ast.Ident).Name
		if StringToAggFunc(funName) != AggUnknown {
			isAgg = true
		}
	}
	return isAgg
}
