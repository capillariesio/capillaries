package capimq_message_broker

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClaimReturnAck(t *testing.T) {
	msg := CapimqInternalMessage{
		CapimqWaitRetryGroup: "ks1/1",
	}

	msgs := make([]*CapimqInternalMessage, 0)
	for i := range int16(10) {
		msgs = append(msgs, &CapimqInternalMessage{
			Id:                   fmt.Sprintf("%05d", i+1),
			CapimqWaitRetryGroup: msg.CapimqWaitRetryGroup + "/node1",
		})
	}

	for i := range int16(10) {
		msgs = append(msgs, &CapimqInternalMessage{
			Id:                   fmt.Sprintf("%05d", 10+i+1),
			CapimqWaitRetryGroup: msg.CapimqWaitRetryGroup + "/node2",
		})
	}

	mb := NewMessageBroker(1000)
	assert.Nil(t, mb.QBulk(msgs))

	assert.Equal(t, 10, mb.Count(HeapTypeQ, msg.CapimqWaitRetryGroup+"/node2"))
	assert.Equal(t, 20, mb.Count(HeapTypeQ, msg.CapimqWaitRetryGroup))

	msgsHead := mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 20)
	assert.Equal(t, 20, len(msgsHead))
	assert.Equal(t, fmt.Sprintf("%05d", 1), msgsHead[0].Id)
	assert.Equal(t, msg.CapimqWaitRetryGroup+"/node1", msgsHead[0].CapimqWaitRetryGroup)
	assert.Equal(t, fmt.Sprintf("%05d", 10), msgsHead[9].Id)
	assert.Equal(t, msg.CapimqWaitRetryGroup+"/node1", msgsHead[9].CapimqWaitRetryGroup)
	assert.Equal(t, fmt.Sprintf("%05d", 11), msgsHead[10].Id)
	assert.Equal(t, msg.CapimqWaitRetryGroup+"/node2", msgsHead[10].CapimqWaitRetryGroup)
	assert.Equal(t, fmt.Sprintf("%05d", 20), msgsHead[19].Id)
	assert.Equal(t, msg.CapimqWaitRetryGroup+"/node2", msgsHead[19].CapimqWaitRetryGroup)

	// Claim all
	for i := range 20 {
		claimedMsg, err := mb.Claim("test worker")
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("%05d", i+1), claimedMsg.Id)
	}

	// Check wip 20
	assert.Equal(t, 20, mb.Count(HeapTypeWip, msg.CapimqWaitRetryGroup))
	msgsHead = mb.HeadTail(HeapTypeWip, QueueReadHead, 0, 20)
	assert.Equal(t, 20, len(msgsHead))

	// Return 15
	assert.Nil(t, mb.Return(fmt.Sprintf("%05d", 15), 1000))

	// Check q 1
	msgsHead = mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 10)
	assert.Equal(t, 1, len(msgsHead))
	assert.Equal(t, fmt.Sprintf("%05d", 15), msgsHead[0].Id)

	time.Sleep(time.Duration(200) * time.Millisecond)

	// Return 14
	assert.Nil(t, mb.Return(fmt.Sprintf("%05d", 14), 1000))

	// Check q 2
	msgsHead = mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 10)
	assert.Equal(t, 2, len(msgsHead))
	// Same runId and node, so order by Id
	assert.Equal(t, fmt.Sprintf("%05d", 14), msgsHead[0].Id)
	assert.Equal(t, fmt.Sprintf("%05d", 15), msgsHead[1].Id)
	// Their delivery times must match, they belong to the same node and both are postponed
	assert.Equal(t, msgsHead[0].DeliverAfter, msgsHead[1].DeliverAfter)

	time.Sleep(time.Duration(200) * time.Millisecond)

	// Return 2
	assert.Nil(t, mb.Return(fmt.Sprintf("%05d", 2), 1000))

	// Check q 2
	msgsHead = mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 10)
	assert.Equal(t, 3, len(msgsHead))
	// 14 and 15 were returned earlier, so they go first
	assert.Equal(t, fmt.Sprintf("%05d", 14), msgsHead[0].Id)
	assert.Equal(t, fmt.Sprintf("%05d", 15), msgsHead[1].Id)
	// Their delivery times must match, they belong to the same node and both are postponed
	assert.Equal(t, msgsHead[0].DeliverAfter, msgsHead[1].DeliverAfter)
	// 2 goes next, and its delivery time is at least 200ms later
	assert.Equal(t, fmt.Sprintf("%05d", 2), msgsHead[2].Id)
	assert.LessOrEqual(t, msgsHead[0].DeliverAfter+200, msgsHead[2].DeliverAfter)

	// Check wip size 17
	msgsHead = mb.HeadTail(HeapTypeWip, QueueReadHead, 0, 20)
	assert.Equal(t, 17, len(msgsHead))

	// Ack one, remove from wip
	assert.Nil(t, mb.Ack(fmt.Sprintf("%05d", 6)))

	// Check wip size 16
	msgsHead = mb.HeadTail(HeapTypeWip, QueueReadHead, 0, 20)
	assert.Equal(t, 16, len(msgsHead))

	// Filter q
	assert.Equal(t, 1, len(mb.Filter(HeapTypeQ, msg.CapimqWaitRetryGroup+"/node1")))

	// Filter wip
	assert.Equal(t, 8, len(mb.Filter(HeapTypeWip, msg.CapimqWaitRetryGroup+"/node1")))

	// Delete q
	assert.Equal(t, 1, mb.Delete(HeapTypeQ, msg.CapimqWaitRetryGroup+"/node1"))

	// Delete wip
	assert.Equal(t, 8, mb.Delete(HeapTypeWip, msg.CapimqWaitRetryGroup+"/node1"))

}

func TestHeartbeat(t *testing.T) {
	msg := CapimqInternalMessage{
		CapimqWaitRetryGroup: "ks1/1/node1",
	}

	msgs := make([]*CapimqInternalMessage, 0)
	for i := range int16(3) {
		msgs = append(msgs, &CapimqInternalMessage{
			Id:                   fmt.Sprintf("%05d", i+1),
			CapimqWaitRetryGroup: msg.CapimqWaitRetryGroup,
		})
	}

	mb := NewMessageBroker(1000)
	assert.Nil(t, mb.QBulk(msgs))

	// Claim 1 and 2
	claimedMsg, err := mb.Claim("test worker")
	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf("%05d", 1), claimedMsg.Id)
	claimedMsg, err = mb.Claim("test worker")
	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf("%05d", 2), claimedMsg.Id)

	time.Sleep(time.Duration(200) * time.Millisecond)

	// Bad heartbeat
	assert.Contains(t, mb.Heartbeat(fmt.Sprintf("%05d", 100)).Error(), "cannot heartbeat, message with id 00100 not found in wip")

	// Hertbeat 2, but not 1
	assert.Nil(t, mb.Heartbeat(fmt.Sprintf("%05d", 2)))

	// Return count 1 (msg id 1)
	deadMessages := mb.ReturnDead(150)
	assert.Equal(t, 1, len(deadMessages))
	assert.True(t, strings.HasPrefix(deadMessages[0], "00001 ks1/1/node1"))

	// Check 1 (returned) and 3 (never claimed) in q
	msgsHead := mb.HeadTail(HeapTypeQ, QueueReadHead, 0, 10)
	assert.Equal(t, 2, len(msgsHead))
	assert.Equal(t, fmt.Sprintf("%05d", 1), msgsHead[0].Id)
	assert.Equal(t, fmt.Sprintf("%05d", 3), msgsHead[1].Id)

	// Check 2 still in wip
	msgsHead = mb.HeadTail(HeapTypeWip, QueueReadHead, 0, 10)
	assert.Equal(t, 1, len(msgsHead))
	assert.Equal(t, fmt.Sprintf("%05d", 2), msgsHead[0].Id)
}
