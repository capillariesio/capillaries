package eval

import (
	"fmt"
	"strings"
)

type VarValuesMap map[string]map[string]interface{}

func (vars *VarValuesMap) Tables() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for table, _ := range *vars {
		sb.WriteString(fmt.Sprintf("%s ", table))
	}
	sb.WriteString("]")
	return sb.String()
}

func (vars *VarValuesMap) Names() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for table, fldMap := range *vars {
		for fld, _ := range fldMap {
			sb.WriteString(fmt.Sprintf("%s.%s ", table, fld))
		}
	}
	sb.WriteString("]")
	return sb.String()
}

// func (vars *VarValuesMap) NamesByTable(tableName string) string {
// 	sb := strings.Builder{}
// 	sb.WriteString("[")
// 	if fldMap, ok := (*vars)[tableName]; ok {
// 		for fld, _ := range fldMap {
// 			sb.WriteString(fmt.Sprintf("%s ", fld))
// 		}
// 	}
// 	sb.WriteString("]")
// 	return sb.String()
// }
