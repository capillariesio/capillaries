package wfmodel

import (
	"encoding/json"
	"fmt"
	"strings"
)

/*
MessagePayloadComment - generic paylod
Comment - unstructured, can be anything, like information about the sender of the signal
*/
type MessagePayloadComment struct {
	Comment string `json:"comment"`
}

// Message types, payload depends on it
const (
	MessageTypeDataBatch             = 1
	MessageTypeShutown               = 101 // pass processor_id
	MessageTypeSetLoggingLevel       = 102 // pass processor_id and logging level
	MessageTypeCancelProcessInstance = 103 // Pass process id and process instance
)

/*
Message - carries data and signals to processors/nodes
1. No version support. Premature optimization is the root of all evil.
2. Used for data transfer and for control signals.
3. For faster de/serialization, consider custom parser not involving reflection
4. Timestamps are int (not uint) because Unix epoch is int
*/
type Message struct {
	Ts          int64 `json:"ts"`
	MessageType int   `json:"message_type"`
	Payload     any   `json:"payload"` // This depends on MessageType
}

func (msg Message) ToString() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Ts:%d, MessageType:%d. ", msg.Ts, msg.MessageType))
	if msg.MessageType == MessageTypeDataBatch && msg.Payload != nil {
		batchPayload, ok := msg.Payload.(MessagePayloadDataBatch)
		if ok {
			sb.WriteString(batchPayload.ToString())
		}
	}
	return sb.String()
}

func (msg Message) Serialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		// This is really unexpected, log the whole msg
		return nil, fmt.Errorf("cannot serialize message: %s. %v", msg.ToString(), err)
	}
	return jsonBytes, nil
}

func (msg *Message) Deserialize(jsonBytes []byte) error {
	var payload json.RawMessage
	msg.Payload = &payload
	err := json.Unmarshal(jsonBytes, &msg)
	if err != nil {
		// This is really unexpected, log the whole json as bytes
		return fmt.Errorf("cannot deserialize message: %v. %v", jsonBytes, err)
	}

	switch msg.MessageType {
	case MessageTypeDataBatch:
		var payloadDataChunk MessagePayloadDataBatch
		err := json.Unmarshal(payload, &payloadDataChunk)
		if err != nil {
			return err
		}
		msg.Payload = payloadDataChunk
	case MessageTypeCancelProcessInstance:
		payloadComment := MessagePayloadComment{}
		err := json.Unmarshal(payload, &payloadComment)
		if err != nil {
			return err
		}
		msg.Payload = payloadComment
	default:
		return fmt.Errorf("cannot deserialize message, unknown message type: %s", msg.ToString())
	}

	return nil
}

// func (tgtMsg *Message) NewDataBatchFromCtx(context *ctx.MessageProcessingContext, targetNodeName string, firstToken int64, lastToken int64, batchIdx int16, batchesTotal int16) {
// 	tgtMsg.Ts = time.Now().UnixMilli()
// 	tgtMsg.MessageType = MessageTypeDataBatch
// 	tgtMsg.Payload = MessagePayloadDataBatch{
// 		ScriptURI:       context.BatchInfo.ScriptURI,
// 		ScriptParamsURI: context.BatchInfo.ScriptParamsURI,
// 		DataKeyspace:    context.BatchInfo.DataKeyspace,
// 		RunId:           context.BatchInfo.RunId,
// 		TargetNodeName:  targetNodeName,
// 		FirstToken:      firstToken,
// 		LastToken:       lastToken,
// 		BatchIdx:        batchIdx,
// 		BatchesTotal:    batchesTotal}
// }
