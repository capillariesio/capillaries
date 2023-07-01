package deploy

import (
	"fmt"
	"strings"
	"time"
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

func ExecLocalAndParseOpenstackOutput(prj *Project, cmdPath string, params []string) ([]map[string]string, ExecResult) {
	er := ExecLocal(prj, cmdPath, params, prj.OpenstackVars, "")
	if er.Error != nil {
		return nil, er
	}
	rows, err := ParseOpenstackOutput(er.Stdout)
	if err != nil {
		return nil, er
	}
	return rows, er
}

func WaitForEntityToBeCreated(prj *Project, entityType string, entityName string, entityId string, timeoutSeconds int, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("WaitForEntityToBeCreated: "+entityName, isVerbose)
	startWaitTs := time.Now()
	for {
		rows, er := ExecLocalAndParseOpenstackOutput(prj, "openstack", []string{entityType, "show", entityId})
		lb.Add(er.ToString())
		if er.Error != nil {
			return lb.Complete(er.Error)
		}
		status := FindOpenstackFieldValue(rows, "status")
		if status == "" {
			return lb.Complete(fmt.Errorf("openstack returned empty %s status for %s(%s)", entityType, entityName, entityId))
		}
		if status == "ACTIVE" {
			return lb.Complete(nil)
		}
		if status != "BUILD" {
			return lb.Complete(fmt.Errorf("%s %s(%s) was built, but the status is %s", entityType, entityName, entityId, status))
		}
		if time.Since(startWaitTs).Seconds() > float64(timeoutSeconds) {
			return lb.Complete(fmt.Errorf("giving up after waiting for %s %s(%s) to be created", entityType, entityName, entityId))
		}
		time.Sleep(10 * time.Second)
	}
}
