package mq

import (
	"testing"
	"time"

	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/stretchr/testify/assert"
)

func TestClaimReturnAck(t *testing.T) {
	msg := wfmodel.Message{
		DataKeyspace: "ks1",
		RunId:        1,
	}

	msgs := make([]*wfmodel.Message, 0)
	for i := range int16(10) {
		msgs = append(msgs, &wfmodel.Message{
			DataKeyspace:   msg.DataKeyspace,
			RunId:          msg.RunId,
			TargetNodeName: "node1",
			BatchIdx:       i,
		})
	}
	for i := range int16(10) {
		msgs = append(msgs, &wfmodel.Message{
			DataKeyspace:   msg.DataKeyspace,
			RunId:          msg.RunId,
			TargetNodeName: "node2",
			BatchIdx:       i,
		})
	}

	mb := NewMessageBroker()
	assert.Nil(t, mb.QBulk(msgs, 1000))

	msgsHead := mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 20)
	assert.Equal(t, 20, len(msgsHead))
	assert.Equal(t, uint64(1), msgsHead[0].Id)
	assert.Equal(t, "node1", msgsHead[0].TargetNodeName)
	assert.Equal(t, uint64(10), msgsHead[9].Id)
	assert.Equal(t, "node1", msgsHead[9].TargetNodeName)
	assert.Equal(t, uint64(11), msgsHead[10].Id)
	assert.Equal(t, "node2", msgsHead[10].TargetNodeName)
	assert.Equal(t, uint64(20), msgsHead[19].Id)
	assert.Equal(t, "node2", msgsHead[19].TargetNodeName)

	// Claim all
	for i := range 20 {
		claimedMsg, err := mb.Claim("test worker")
		assert.Nil(t, err)
		assert.Equal(t, uint64(i+1), claimedMsg.Id)
	}

	// Check wip 20
	msgsHead = mb.HeadTail(HeapTypeWip, QueueReadHead, 0, 20)
	assert.Equal(t, 20, len(msgsHead))

	// Return 15
	assert.Nil(t, mb.Return(15, 1000))

	// Check q 1
	msgsHead = mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 10)
	assert.Equal(t, 1, len(msgsHead))
	assert.Equal(t, uint64(15), msgsHead[0].Id)

	time.Sleep(time.Duration(200) * time.Millisecond)

	// Return 14
	assert.Nil(t, mb.Return(14, 1000))

	// Check q 2
	msgsHead = mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 10)
	assert.Equal(t, 2, len(msgsHead))
	// Same runId and node, so order by Id
	assert.Equal(t, uint64(14), msgsHead[0].Id)
	assert.Equal(t, uint64(15), msgsHead[1].Id)
	// Their delivery times must match, they belong to the same node and both are postponed
	assert.Equal(t, msgsHead[0].DeliverAfter, msgsHead[1].DeliverAfter)

	time.Sleep(time.Duration(200) * time.Millisecond)

	// Return 2
	assert.Nil(t, mb.Return(2, 1000))

	// Check q 2
	msgsHead = mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 10)
	assert.Equal(t, 3, len(msgsHead))
	// 14 and 15 were returned earlier, so they go first
	assert.Equal(t, uint64(14), msgsHead[0].Id)
	assert.Equal(t, uint64(15), msgsHead[1].Id)
	// Their delivery times must match, they belong to the same node and both are postponed
	assert.Equal(t, msgsHead[0].DeliverAfter, msgsHead[1].DeliverAfter)
	// 2 goes next, and its delivery time is at least 200ms later
	assert.Equal(t, uint64(2), msgsHead[2].Id)
	assert.LessOrEqual(t, msgsHead[0].DeliverAfter+200, msgsHead[2].DeliverAfter)

	// Check wip size 17
	msgsHead = mb.HeadTail(HeapTypeWip, QueueReadHead, 0, 20)
	assert.Equal(t, 17, len(msgsHead))

	// Ack one, remove from wip
	assert.Nil(t, mb.Ack(6))

	// Check wip size 16
	msgsHead = mb.HeadTail(HeapTypeWip, QueueReadHead, 0, 20)
	assert.Equal(t, 16, len(msgsHead))
}

func TestHeartbeat(t *testing.T) {
	msg := wfmodel.Message{
		DataKeyspace:   "ks1",
		RunId:          1,
		TargetNodeName: "node1",
	}

	msgs := make([]*wfmodel.Message, 0)
	for i := range int16(3) {
		msgs = append(msgs, &wfmodel.Message{
			DataKeyspace:   msg.DataKeyspace,
			RunId:          msg.RunId,
			TargetNodeName: msg.TargetNodeName,
			BatchIdx:       i,
		})
	}

	mb := NewMessageBroker()
	assert.Nil(t, mb.QBulk(msgs, 1000))

	// Claim 1 and 2
	claimedMsg, err := mb.Claim("test worker")
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), claimedMsg.Id)
	claimedMsg, err = mb.Claim("test worker")
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), claimedMsg.Id)

	time.Sleep(time.Duration(200) * time.Millisecond)

	// Bad heartbeat
	assert.Contains(t, mb.Heartbeat(100).Error(), "cannot heartbeat, message with id 100 not found in wip")

	// Hertbeat 2, but not 1
	assert.Nil(t, mb.Heartbeat(2))

	// Return count 1 (msg id 1)
	assert.Equal(t, 1, mb.ReturnDead(150))

	// Check 1 (returned) and 3 (never claimed) in q
	msgsHead := mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 10)
	assert.Equal(t, 2, len(msgsHead))
	assert.Equal(t, uint64(1), msgsHead[0].Id)
	assert.Equal(t, uint64(3), msgsHead[1].Id)

	// Check 2 still in wip
	msgsHead = mb.HeadTail(HeapTypeWip, QueueReadHead, 0, 10)
	assert.Equal(t, 1, len(msgsHead))
	assert.Equal(t, uint64(2), msgsHead[0].Id)
}
