package sc

import (
	"encoding/json"
)

type CustomProcessorDefFactory interface {
	Create(processorType string) (CustomProcessorDef, bool)
}

type CustomProcessorDef interface {
	Deserialize(raw json.RawMessage, customProcSettings json.RawMessage, caPath string) error
	GetFieldRefs() *FieldRefs
	GetUsedInTargetExpressionsFields() *FieldRefs
}
