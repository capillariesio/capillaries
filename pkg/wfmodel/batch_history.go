package wfmodel

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type NodeBatchStatusType int8

// In priority order
const (
	NodeBatchNone            NodeBatchStatusType = 0
	NodeBatchStart           NodeBatchStatusType = 1
	NodeBatchSuccess         NodeBatchStatusType = 2
	NodeBatchFail            NodeBatchStatusType = 3 // Biz logicerror or data tble (not WF) error
	NodeBatchRunStopReceived NodeBatchStatusType = 104
)

const TableNameBatchHistory = "wf_batch_history"

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type BatchHistoryEvent struct {
	Ts           time.Time           `header:"ts" format:"%-33v" column:"ts" type:"timestamp" json:"ts"`
	RunId        int16               `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true" json:"run_id"`
	ScriptNode   string              `header:"script_node" format:"%20v" column:"script_node" type:"text" key:"true" json:"script_node"`
	BatchIdx     int16               `header:"bnum" format:"%5v" column:"batch_idx" type:"int" key:"true" json:"batch_idx"`
	BatchesTotal int16               `header:"tbtchs" format:"%6v" column:"batches_total" type:"int" json:"batches_total"`
	Status       NodeBatchStatusType `header:"sts" format:"%3v" column:"status" type:"tinyint" key:"true" json:"status"`
	FirstToken   int64               `header:"ftoken" format:"%21v" column:"first_token" type:"bigint" json:"first_token"`
	LastToken    int64               `header:"ltoken" format:"%21v" column:"last_token" type:"bigint" json:"last_token"`
	Instance     string              `header:"instance" format:"%21v" column:"instance" type:"text" json:"instance"`
	Thread       int64               `header:"thread" format:"%4v" column:"thread" type:"bigint" json:"thread"`
	Comment      string              `header:"comment" format:"%v" column:"comment" type:"text" json:"comment"`
}

func BatchHistoryEventAllFields() []string {
	return []string{"ts", "run_id", "script_node", "batch_idx", "batches_total", "status", "first_token", "last_token", "instance", "thread", "comment"}
}
func NewBatchHistoryEventFromMap(r map[string]interface{}, fields []string) (*BatchHistoryEvent, error) {
	res := &BatchHistoryEvent{}
	for _, fieldName := range fields {
		var err error
		switch fieldName {
		case "ts":
			res.Ts, err = ReadTimeFromRow(fieldName, r)
		case "run_id":
			res.RunId, err = ReadInt16FromRow(fieldName, r)
		case "script_node":
			res.ScriptNode, err = ReadStringFromRow(fieldName, r)
		case "batch_idx":
			res.BatchIdx, err = ReadInt16FromRow(fieldName, r)
		case "batches_total":
			res.BatchesTotal, err = ReadInt16FromRow(fieldName, r)
		case "status":
			res.Status, err = ReadNodeBatchStatusFromRow(fieldName, r)
		case "first_token":
			res.FirstToken, err = ReadInt64FromRow(fieldName, r)
		case "last_token":
			res.LastToken, err = ReadInt64FromRow(fieldName, r)
		case "instance":
			res.Instance, err = ReadStringFromRow(fieldName, r)
		case "thread":
			res.Thread, err = ReadInt64FromRow(fieldName, r)
		case "comment":
			res.Comment, err = ReadStringFromRow(fieldName, r)
		default:
			return nil, fmt.Errorf("unknown %s field %s", fieldName, TableNameNodeHistory)
		}
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// ToSpacedString - prints formatted field values, uses reflection, shoud not be used in prod
func (n BatchHistoryEvent) ToSpacedString() string {
	t := reflect.TypeOf(n)
	formats := GetObjectModelFieldFormats(t)
	values := make([]string, t.NumField())

	v := reflect.ValueOf(&n).Elem()
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		values[i] = fmt.Sprintf(formats[i], fv)
	}
	return strings.Join(values, PrintTableDelimiter)
}
