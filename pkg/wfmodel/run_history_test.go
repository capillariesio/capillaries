package wfmodel

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func (e *RunHistoryEvent) ToMap() map[string]any {
	m := map[string]any{}
	for _, fieldName := range RunHistoryEventAllFields() {
		switch fieldName {
		case "ts":
			m[fieldName] = e.Ts
		case "run_id":
			m[fieldName] = e.RunId
		case "status":
			m[fieldName] = int8(e.Status) // Pretend this is returned by Cassandra
		case "comment":
			m[fieldName] = e.Comment
		default:
		}
	}
	return m
}

func TestRunHistoryRowsToEventsGood(t *testing.T) {
	rows := []map[string]any{
		(&RunHistoryEvent{
			Ts:     time.Date(2001, 1, 1, 1, 1, 2, 0, time.UTC),
			RunId:  int16(1),
			Status: RunStart,
		}).ToMap(),
		(&RunHistoryEvent{
			Ts:     time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
			RunId:  int16(1),
			Status: RunStart,
		}).ToMap(),
	}

	events, err := RunHistoryRowsToEvents(rows)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(events))
	assert.Equal(t, "{2001-01-01 01:01:01 +0000 UTC 1 1 }", fmt.Sprintf("%v", *events[0]))
	assert.Equal(t, "{2001-01-01 01:01:02 +0000 UTC 1 1 }", fmt.Sprintf("%v", *events[1]))
}

func TestRunHistoryRowsToEventsBad(t *testing.T) {
	rows := []map[string]any{
		(&NodeHistoryEvent{
			Ts:     time.Date(2001, 1, 1, 1, 1, 2, 0, time.UTC),
			RunId:  int16(1),
			Status: NodeBatchStart,
		}).ToMap(),
	}

	rows[0]["ts"] = "a"

	_, err := RunHistoryRowsToEvents(rows)
	assert.Contains(t, err.Error(), "cannot read time ts")
}

func TestRunHistoryEventsToRunStatusMap(t *testing.T) {
	events := []*RunHistoryEvent{
		(&RunHistoryEvent{
			RunId:  int16(1),
			Status: RunNone,
		}),
		(&RunHistoryEvent{
			RunId:  int16(2),
			Status: RunStart,
		}),
		(&RunHistoryEvent{
			RunId:  int16(3),
			Status: RunComplete,
		}),
		(&RunHistoryEvent{
			RunId:  int16(4),
			Status: RunStop,
		}),

		// Change

		(&RunHistoryEvent{
			RunId:  int16(1),
			Status: RunStart,
		}),
		(&RunHistoryEvent{
			RunId:  int16(2),
			Status: RunComplete,
		}),
		(&RunHistoryEvent{
			RunId:  int16(3),
			Status: RunStop,
		}),
	}

	m := RunHistoryEventsToRunStatusMap(events)
	assert.Equal(t, 4, len(m))
	assert.Equal(t, RunStart, m[1])
	assert.Equal(t, RunComplete, m[2])
	assert.Equal(t, RunStop, m[3])
	assert.Equal(t, RunStop, m[4])
}

func TestRunHistoryEventsToLifespanMap(t *testing.T) {
	events := []*RunHistoryEvent{
		(&RunHistoryEvent{
			Ts:     time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
			RunId:  int16(1),
			Status: RunStart,
		}),
		(&RunHistoryEvent{
			Ts:      time.Date(2001, 1, 1, 1, 1, 2, 0, time.UTC),
			RunId:   int16(1),
			Status:  RunComplete,
			Comment: "completecomment",
		}),
		(&RunHistoryEvent{
			Ts:      time.Date(2001, 1, 1, 1, 1, 3, 0, time.UTC),
			RunId:   int16(1),
			Status:  RunStop,
			Comment: "stopcomment",
		}),
	}

	m, err := RunHistoryEventsToLifespanMap(events)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(m))
	assert.Equal(t, "{1:{run_id: 1, start_ts:\"2001-01-01T01:01:01.000+0000\", final_status:stop, completed_ts:\"2001-01-01T01:01:02.000+0000\", stopped_ts:\"2001-01-01T01:01:03.000+0000\", completed_comment:completecomment, stopped_comment:stopcomment}}", m.ToString())
}

func TestRunStatusToString(t *testing.T) {
	assert.Equal(t, "none", RunNone.ToString())
	assert.Equal(t, "start", RunStart.ToString())
	assert.Equal(t, "complete", RunComplete.ToString())
	assert.Equal(t, "stop", RunStop.ToString())
}
