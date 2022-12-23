package deploy

import (
	"fmt"
	"strings"
)

func ParseOpenstackOutput(input string) ([]map[string]string, error) {
	var columnHeaders []string
	lines := strings.Split(input, "\n")
	expectedDataRows := len(lines) - 4
	if expectedDataRows <= 0 {
		return make([]map[string]string, 0), nil
	}
	result := make([]map[string]string, expectedDataRows)
	dataRowIdx := 0
	for _, line := range lines {
		if len(line) > 0 && line[0] == '|' {
			columns := strings.Split(line, "|")
			// Remove first and last
			if len(columns) < 3 {
				return nil, fmt.Errorf("short openstack row: %s", line)
			}
			columns = columns[1 : len(columns)-1]
			if len(columnHeaders) == 0 {
				columnHeaders = make([]string, len(columns))
				for colIdx, colHeader := range columns {
					columnHeaders[colIdx] = strings.TrimSpace(colHeader)
				}
			} else {
				if len(columns) != len(columnHeaders) {
					return nil, fmt.Errorf("bad openstack row: %s", line)
				}
				result[dataRowIdx] = map[string]string{}
				for colIdx, colHeader := range columnHeaders {
					result[dataRowIdx][colHeader] = strings.TrimSpace(columns[colIdx])
				}
				dataRowIdx++
			}
		}
	}
	return result[:dataRowIdx], nil
}

func FindOpenstackColumnValue(rows []map[string]string, fieldNameToReturn string, fieldNameToSearch string, fieldValueToSearch string) string {
	for _, fields := range rows {
		if fields[fieldNameToSearch] == fieldValueToSearch {
			return fields[fieldNameToReturn]
		}
	}
	return ""
}

func FindOpenstackFieldValue(rows []map[string]string, fieldName string) string {
	// Handle value list, like security group show:
	// | rules | rule1 |
	// |       | rule2 |
	var resultArray []string
	lastFieldName := ""
	for _, fields := range rows {
		if fields["Field"] == fieldName || fields["Field"] == "" && lastFieldName == fieldName {
			resultArray = append(resultArray, fields["Value"])
		}
		if fields["Field"] != "" {
			lastFieldName = fields["Field"]
		}
	}
	return strings.Join(resultArray, "\n")
}

func ExecLocalAndParseOpenstackOutput(prj *Project, logBuilder *strings.Builder, cmdPath string, params []string) ([]map[string]string, ExecResult) {
	er := ExecLocal(prj, cmdPath, params)
	defer logBuilder.WriteString(er.ToString())

	if er.Error != nil {
		return nil, er
	}
	rows, err := ParseOpenstackOutput(er.Stdout)
	if err != nil {
		return nil, er
	}
	return rows, er
}
