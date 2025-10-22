package mq

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type HeapType string

const (
	HeapTypeQ       HeapType = "q"
	HeapTypeWip     HeapType = "wip"
	HeapTypeUnknown HeapType = "unknown"
)

func StringToHeapType(qOrWip string) (HeapType, error) {
	switch qOrWip {
	case string(HeapTypeQ):
		return HeapTypeQ, nil
	case string(HeapTypeWip):
		return HeapTypeQ, nil
	default:
		return HeapTypeUnknown, fmt.Errorf("invalid heap type %s", qOrWip)
	}
}

type QueueReadType string

const (
	QueueReadHead    QueueReadType = "head"
	QueueReadTail    QueueReadType = "tail"
	QueueReadFilter  QueueReadType = "filter"
	QueueReadUnknown QueueReadType = "unknown"
)

func StringToQueueReadType(headOrTail string) (QueueReadType, error) {
	switch headOrTail {
	case string(QueueReadHead):
		return QueueReadHead, nil
	case string(QueueReadTail):
		return QueueReadTail, nil
	case string(QueueReadFilter):
		return QueueReadFilter, nil
	default:
		return QueueReadUnknown, fmt.Errorf("invalid head/tail %s", headOrTail)
	}
}

type MessageBroker struct {
	Q          []*wfmodel.Message
	Wip        map[uint64]*wfmodel.Message
	MsgCounter uint64
	QMutex     sync.RWMutex
	WipMutex   sync.RWMutex
}

func NewMessageBroker() *MessageBroker {
	mb := MessageBroker{
		Q:          make([]*wfmodel.Message, 0),
		Wip:        map[uint64]*wfmodel.Message{},
		MsgCounter: 0,
		QMutex:     sync.RWMutex{},
		WipMutex:   sync.RWMutex{},
	}
	return &mb
}

func (mb *MessageBroker) sortQ() {
	sort.Slice(mb.Q, func(i int, j int) bool { return mb.Q[i].DeliverEarlierThan(mb.Q[j]) })
}

func (mb *MessageBroker) ReturnDead(deadTimeoutMillis int64) int {
	latestAllowedHearbit := time.Now().UnixMilli() - deadTimeoutMillis
	msgs := make([]*wfmodel.Message, 0)

	mb.WipMutex.Lock()
	for _, msg := range mb.Wip {
		if msg.Heartbeat < latestAllowedHearbit {
			msg.Heartbeat = msg.Ts // Reset heartbeat
			msgs = append(msgs, msg)
		}
	}
	for _, msg := range msgs {
		delete(mb.Wip, msg.Id)
	}
	mb.WipMutex.Unlock()

	if len(msgs) == 0 {
		return 0
	}

	mb.QMutex.Lock()
	mb.Q = append(mb.Q, msgs...)
	mb.sortQ()
	mb.QMutex.Unlock()

	return len(msgs)
}

func (mb *MessageBroker) QBulk(msgs []*wfmodel.Message, maxMessages int) error {
	ts := time.Now().UnixMilli()

	mb.QMutex.Lock()

	curLen := len(mb.Q)
	if len(msgs)+curLen > maxMessages {
		mb.QMutex.Unlock()
		return fmt.Errorf("max_messages %d exceeded: already in queue %d, adding %d", maxMessages, curLen, len(msgs))
	}

	for _, msg := range msgs {
		mb.MsgCounter++
		msg.Id = mb.MsgCounter
		msg.Ts = ts
		msg.DeliverAfter = ts
		msg.Heartbeat = ts
		mb.Q = append(mb.Q, msg)
	}

	// Need to sort, newer items may need to be delivered earlier than those already in the q (returned)
	mb.sortQ()

	mb.QMutex.Unlock()

	return nil
}

func (mb *MessageBroker) Claim(claimComment string) (*wfmodel.Message, error) {
	now := time.Now().UnixMilli()
	mb.QMutex.Lock()
	if len(mb.Q) == 0 {
		mb.QMutex.Unlock()
		return nil, nil
	}
	// The earliest item is at the head of the slice.
	if mb.Q[0].DeliverAfter > now {
		// All messages are postponed for time later than now, nothing to return
		mb.QMutex.Unlock()
		return nil, nil
	}
	msg := mb.Q[0]
	mb.Q[0] = nil
	mb.Q = mb.Q[1:]
	mb.QMutex.Unlock()

	msg.ClaimComment = claimComment

	mb.WipMutex.Lock()
	mb.Wip[msg.Id] = msg
	mb.WipMutex.Unlock()

	return msg, nil
}

func (mb *MessageBroker) Ack(id uint64) error {
	mb.WipMutex.Lock()
	_, ok := mb.Wip[id]
	if !ok {
		mb.WipMutex.Unlock()
		return fmt.Errorf("cannot ack, message with id %d not found in wip", id)
	}
	delete(mb.Wip, id)
	mb.WipMutex.Unlock()

	return nil
}

func (mb *MessageBroker) Heartbeat(id uint64) error {
	mb.WipMutex.Lock()
	msg, ok := mb.Wip[id]
	if !ok {
		mb.WipMutex.Unlock()
		return fmt.Errorf("cannot heartbeat, message with id %d not found in wip", id)
	}
	msg.Heartbeat = time.Now().UnixMilli()
	mb.WipMutex.Unlock()

	return nil
}

func (mb *MessageBroker) Return(id uint64, delay int64) error {
	mb.WipMutex.Lock()
	msg, ok := mb.Wip[id]
	if !ok {
		mb.WipMutex.Unlock()
		return fmt.Errorf("cannot return, message with id %d not found in wip", id)
	}
	delete(mb.Wip, id)
	mb.WipMutex.Unlock()

	ks := msg.DataKeyspace
	runId := msg.RunId
	nodeName := msg.TargetNodeName
	newDeliverAfter := time.Now().UnixMilli() + delay

	msg.ClaimComment = ""

	mb.QMutex.Lock()
	mb.Q = append(mb.Q, msg)
	for _, msg := range mb.Q {
		if msg.DataKeyspace == ks && msg.RunId == runId && msg.TargetNodeName == nodeName {
			msg.DeliverAfter = newDeliverAfter
		}
	}
	mb.sortQ()
	mb.QMutex.Unlock()

	return nil
}

func (mb *MessageBroker) Ks() []string {
	ksMap := map[string]struct{}{}

	mb.QMutex.RLock()

	for _, msg := range mb.Q {
		ksMap[msg.DataKeyspace] = struct{}{}
	}

	mb.QMutex.RUnlock()

	result := make([]string, len(ksMap))
	ksCount := 0
	for ks := range ksMap {
		result[ksCount] = ks
		ksCount++
	}

	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })

	return result
}

func (mb *MessageBroker) Count(heapType HeapType, ks string, runId int16, nodeName string) int {
	count := 0
	switch heapType {
	case HeapTypeQ:
		mb.QMutex.RLock()
		for _, msg := range mb.Q {
			if (ks == "" || ks == msg.DataKeyspace) && (nodeName == "" || nodeName == msg.TargetNodeName) && (runId == 0 || runId == msg.RunId) {
				count++
			}
		}
		mb.QMutex.RUnlock()
		return count
	case HeapTypeWip:
		mb.WipMutex.RLock()
		for _, msg := range mb.Wip {
			if (ks == "" || ks == msg.DataKeyspace) && (nodeName == "" || nodeName == msg.TargetNodeName) && (runId == 0 || runId == msg.RunId) {
				count++
			}
		}
		mb.WipMutex.RUnlock()
		return count
	default:
		return 0
	}
}

func (mb *MessageBroker) Delete(heapType HeapType, ks string, runId int16, nodeName string) int {
	switch heapType {
	case HeapTypeQ:
		count := 0
		mb.QMutex.Lock()
		i := 0
		for i < len(mb.Q) {
			if (ks == "" || ks == mb.Q[i].DataKeyspace) && (nodeName == "" || nodeName == mb.Q[i].TargetNodeName) && (runId == 0 || runId == mb.Q[i].RunId) {
				count++
				mb.Q[i] = nil
				mb.Q = append(mb.Q[:i], mb.Q[i+1:]...)
			}
			i++
		}
		mb.QMutex.Unlock()
		return count
	case HeapTypeWip:
		idsToDelete := make([]uint64, 0)
		mb.WipMutex.Lock()
		for id, msg := range mb.Wip {
			if (ks == "" || ks == msg.DataKeyspace) && (nodeName == "" || nodeName == msg.TargetNodeName) && (runId == 0 || runId == msg.RunId) {
				idsToDelete = append(idsToDelete, id)
			}
		}
		for _, id := range idsToDelete {
			delete(mb.Wip, id)
		}
		mb.WipMutex.Unlock()
		return len(idsToDelete)
	default:
		return 0
	}
}

func (mb *MessageBroker) HeadTail(heapType HeapType, queueRead QueueReadType, from int, count int) []*wfmodel.Message {
	msgs := make([]*wfmodel.Message, 0)
	switch heapType {
	case HeapTypeQ:
		mb.QMutex.RLock()
		i := from
		inc := 1
		if queueRead == QueueReadTail {
			i = len(mb.Q) - 1 - from
			inc = -1
		}
		for range count {
			if i < 0 || i >= len(mb.Q) {
				break
			}
			msgs = append(msgs, mb.Q[i])
			i += inc
		}
		mb.QMutex.RUnlock()
	case HeapTypeWip:
		allMsgs := make([]*wfmodel.Message, 0)
		mb.WipMutex.RLock()
		for _, msg := range mb.Wip {
			allMsgs = append(allMsgs, msg)
		}
		mb.WipMutex.RUnlock()

		sort.Slice(allMsgs, func(i int, j int) bool { return allMsgs[i].DeliverEarlierThan(allMsgs[j]) })

		i := from
		inc := 1
		if queueRead == QueueReadTail {
			i = len(allMsgs) - 1 - from
			inc = -1
		}
		for range count {
			if i < 0 || i >= len(allMsgs) {
				break
			}
			msgs = append(msgs, allMsgs[i])
			i += inc
		}
	}

	return msgs
}

func (mb *MessageBroker) Filter(heapType HeapType, ks string, runId int16, nodeName string) []*wfmodel.Message {
	msgs := make([]*wfmodel.Message, 0)
	switch heapType {
	case HeapTypeQ:
		mb.QMutex.RLock()
		for _, msg := range mb.Q {
			// Empty ks not accepted - will result in too many msgs
			if ks == msg.DataKeyspace && (nodeName == "" || nodeName == msg.TargetNodeName) && (runId == 0 || runId == msg.RunId) {
				msgs = append(msgs, msg)
			}
		}
		mb.QMutex.RUnlock()
	case HeapTypeWip:
		mb.WipMutex.RLock()
		for _, msg := range mb.Wip {
			// Empty ks not accepted - will result in too many msgs
			if ks == msg.DataKeyspace && (nodeName == "" || nodeName == msg.TargetNodeName) && (runId == 0 || runId == msg.RunId) {
				msgs = append(msgs, msg)
			}
		}
		mb.WipMutex.RUnlock()
	}

	return msgs
}
