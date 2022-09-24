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
type BatchHistory struct {
	Ts           time.Time           `header:"ts" format:"%-33v" column:"ts" type:"timestamp"`
	RunId        int16               `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true"`
	ScriptNode   string              `header:"script_node" format:"%20v" column:"script_node" type:"text" key:"true"`
	BatchIdx     int16               `header:"bnum" format:"%5v" column:"batch_idx" type:"int" key:"true"`
	BatchesTotal int16               `header:"tbtchs" format:"%6v" column:"batches_total" type:"int"`
	Status       NodeBatchStatusType `header:"sts" format:"%3v" column:"status" type:"tinyint" key:"true"`
	FirstToken   int64               `header:"ftoken" format:"%21v" column:"first_token" type:"bigint"`
	LastToken    int64               `header:"ltoken" format:"%21v" column:"last_token" type:"bigint"`
	Comment      string              `header:"comment" format:"%v" column:"comment" type:"text"`
}

func BatchHistoryAllFields() []string {
	return []string{"ts", "run_id", "script_node", "batch_idx", "batches_total", "status", "first_token", "last_token", "comment"}
}
func NewBatchHistoryFromMap(r map[string]interface{}, fields []string) (*BatchHistory, error) {
	res := &BatchHistory{}
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

//ToSpacedString - prints formatted field values, uses reflection, shoud not be used in prod
func (n BatchHistory) ToSpacedString() string {
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
