package tag_and_denormalize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/sethvargo/go-envconfig"
)

const ProcessorTagAndDenormalizeName string = "tag_and_denormalize"

// All processor settings, stored in Capillaries script, root values coming from node
type TagAndDenormalizeProcessorDef struct {
	TagFieldName         string            `json:"tag_field_name"`
	RawTagCriteria       map[string]string `json:"tag_criteria"`
	RawTagCriteriaUrl    string            `json:"tag_criteria_url"`
	ParsedTagCriteria    map[string]ast.Expr
	UsedInCriteriaFields sc.FieldRefs
}

func (procDef *TagAndDenormalizeProcessorDef) GetFieldRefs() *sc.FieldRefs {
	return &sc.FieldRefs{
		{
			TableName: sc.CustomProcessorAlias,
			FieldName: procDef.TagFieldName,
			FieldType: sc.FieldTypeString}}
}

func (procDef *TagAndDenormalizeProcessorDef) GetUsedInTargetExpressionsFields() *sc.FieldRefs {
	return &procDef.UsedInCriteriaFields
}

func (procDef *TagAndDenormalizeProcessorDef) Deserialize(raw json.RawMessage, _ json.RawMessage, scriptType sc.ScriptType, caPath string, privateKeys map[string]string) error {
	var err error
	if scriptType == sc.ScriptJson {
		if err = json.Unmarshal(raw, procDef); err != nil {
			return fmt.Errorf("cannot unmarshal tag_and_denormalize processor def json: %s", err.Error())
		}
	} else if scriptType == sc.ScriptYaml {
		if err = json.Unmarshal(raw, procDef); err != nil {
			return fmt.Errorf("cannot unmarshal tag_and_denormalize processor def yaml: %s", err.Error())
		}
	} else {
		return errors.New("cannot unmarshal tag_and_denormalize processor def: json or yaml expected")
	}

	if err := envconfig.Process(context.TODO(), procDef); err != nil {
		return fmt.Errorf("cannot process tag_and_denormalize env variables: %s", err.Error())
	}

	foundErrors := make([]string, 0)
	procDef.ParsedTagCriteria = map[string]ast.Expr{}

	if len(procDef.RawTagCriteriaUrl) > 0 {
		if len(procDef.RawTagCriteria) > 0 {
			return errors.New("cannot unmarshal both tag_criteria and tag_criteria_url - pick one")
		}

		criteriaBytes, err := xfer.GetFileBytes(procDef.RawTagCriteriaUrl, caPath, privateKeys)
		if err != nil {
			return fmt.Errorf("cannot get criteria file [%s]: %s", procDef.RawTagCriteriaUrl, err.Error())
		}

		if len(criteriaBytes) == 0 {
			return fmt.Errorf("criteria file [%s] is empty", procDef.RawTagCriteriaUrl)
		}

		if criteriaBytes != nil {
			if err := json.Unmarshal(criteriaBytes, &procDef.RawTagCriteria); err != nil {
				return fmt.Errorf("cannot unmarshal tag criteria file [%s]: [%s]", procDef.RawTagCriteriaUrl, err.Error())
			}
		}
	} else if len(procDef.RawTagCriteria) == 0 {
		return errors.New("cannot unmarshal with tag_criteria and tag_criteria_url missing")
	}

	for tag, rawExp := range procDef.RawTagCriteria {
		if procDef.ParsedTagCriteria[tag], err = sc.ParseRawGolangExpressionStringAndHarvestFieldRefs(rawExp, &procDef.UsedInCriteriaFields); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("cannot parse tag criteria expression [%s]: [%s]", rawExp, err.Error()))
		}
	}

	// Later on, checkFieldUsageInCustomProcessor() will verify all fields from procDef.UsedInCriteriaFields are valid reader fields

	if len(foundErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(foundErrors, "; "))
	}
	return nil
}

const tagAndDenormalizeFlushBufferSize int = 1000

func (procDef *TagAndDenormalizeProcessorDef) tagAndDenormalize(rsIn *proc.Rowset, flushVarsArray func(varsArray []*eval.VarValuesMap, varsArrayCount int) error) error {
	varsArray := make([]*eval.VarValuesMap, tagAndDenormalizeFlushBufferSize)
	varsArrayCount := 0

	for rowIdx := 0; rowIdx < rsIn.RowCount; rowIdx++ {
		vars := eval.VarValuesMap{}
		if err := rsIn.ExportToVars(rowIdx, &vars); err != nil {
			return err
		}

		for tag, tagCriteria := range procDef.ParsedTagCriteria {
			eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, &vars)
			valVolatile, err := eCtx.Eval(tagCriteria)
			if err != nil {
				return fmt.Errorf("cannot evaluate expression for tag %s criteria: [%s]", tag, err.Error())
			}
			valBool, ok := valVolatile.(bool)
			if !ok {
				return fmt.Errorf("tag %s criteria returned type %T, expected bool", tag, valVolatile)
			}

			if !valBool {
				// This tag criteria were not met, skip it
				continue
			}

			// Add new tag field to the output

			varsArray[varsArrayCount] = &eval.VarValuesMap{}
			// Write tag
			(*varsArray[varsArrayCount])[sc.CustomProcessorAlias] = map[string]any{procDef.TagFieldName: tag}
			// Write r values
			(*varsArray[varsArrayCount])[sc.ReaderAlias] = map[string]any{}
			for fieldName, fieldVal := range vars[sc.ReaderAlias] {
				(*varsArray[varsArrayCount])[sc.ReaderAlias][fieldName] = fieldVal
			}
			varsArrayCount++

			if varsArrayCount == len(varsArray) {
				if err = flushVarsArray(varsArray, varsArrayCount); err != nil {
					return fmt.Errorf("error flushing vars array of size %d: %s", varsArrayCount, err.Error())
				}
				varsArray = make([]*eval.VarValuesMap, tagAndDenormalizeFlushBufferSize)
				varsArrayCount = 0
			}
		}
	}

	if varsArrayCount > 0 {
		if err := flushVarsArray(varsArray, varsArrayCount); err != nil {
			return fmt.Errorf("error flushing leftovers vars array of size %d: %s", varsArrayCount, err.Error())
		}
	}

	return nil
}
