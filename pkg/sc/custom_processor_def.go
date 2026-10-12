package sc

import "encoding/json/jsontext"

type CustomProcessorDefFactory interface {
	Create(processorType string) (CustomProcessorDef, bool)
}

type CustomProcessorDef interface {
	Deserialize(raw jsontext.Value, customProcSettings jsontext.Value, scriptType ScriptType, caPath string, privateKeys map[string]string) error
	GetFieldRefs() *FieldRefs
	GetUsedInTargetExpressionsFields() *FieldRefs
}
