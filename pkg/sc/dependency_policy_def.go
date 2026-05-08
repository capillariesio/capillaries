package sc

import (
	"encoding/json"
	"fmt"
	"go/ast"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/evalcapi"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

// This conf should be never referenced in prod code. It's always in the the config.json. Or in the unit tests. Or in helper tools.
const DefaultPolicyCheckerConfJson string = `
{
	"is_default": true,
	"event_priority_order": "run_is_current(desc), run_id(desc)",
	"rules": [
		{"cmd": "go",   "expression": "nrs.run_is_current == true && nrs.run_status == wfmodel.RunStart && nrs.node_status == wfmodel.NodeBatchSuccess"	},
		{"cmd": "wait", "expression": "nrs.run_is_current == true && (nrs.run_status == wfmodel.RunStart || nrs.run_status == wfmodel.RunNone) && nrs.node_status == wfmodel.NodeBatchNone"	    },
		{"cmd": "wait", "expression": "nrs.run_is_current == true && (nrs.run_status == wfmodel.RunStart || nrs.run_status == wfmodel.RunNone) && nrs.node_status == wfmodel.NodeBatchStart"	    },
		{"cmd": "nogo", "expression": "nrs.run_is_current == true && nrs.run_status == wfmodel.RunStart && nrs.node_status == wfmodel.NodeBatchFail"	    },

		{"cmd": "go",   "expression": "nrs.run_is_current == false && nrs.run_status == wfmodel.RunStart && nrs.node_status == wfmodel.NodeBatchSuccess"	},
		{"cmd": "wait",   "expression": "nrs.run_is_current == false && (nrs.run_status == wfmodel.RunStart || nrs.run_status == wfmodel.RunNone) && nrs.node_status == wfmodel.NodeBatchNone"	},
		{"cmd": "wait",   "expression": "nrs.run_is_current == false && (nrs.run_status == wfmodel.RunStart || nrs.run_status == wfmodel.RunNone) && nrs.node_status == wfmodel.NodeBatchStart"	},

		{"cmd": "go",   "expression": "nrs.run_is_current == false && nrs.run_status == wfmodel.RunComplete && nrs.node_status == wfmodel.NodeBatchSuccess"	},
		{"cmd": "nogo", "expression": "nrs.run_is_current == false && nrs.run_status == wfmodel.RunComplete && nrs.node_status == wfmodel.NodeBatchFail"	},
        {"cmd": "nogo", "expression": "nrs.run_is_current == false && nrs.run_status == wfmodel.RunStop" }
	]
}`

type ReadyToRunNodeCmdType string

const (
	NodeNone ReadyToRunNodeCmdType = "none"
	NodeGo   ReadyToRunNodeCmdType = "go"
	NodeWait ReadyToRunNodeCmdType = "wait"
	NodeNogo ReadyToRunNodeCmdType = "nogo"
)

func ReadyToRunNodeCmdTypeFromString(s string) (ReadyToRunNodeCmdType, error) {
	switch s {
	case string(NodeNone):
		return NodeNone, nil
	case string(NodeGo):
		return NodeGo, nil
	case string(NodeWait):
		return NodeWait, nil
	case string(NodeNogo):
		return NodeNogo, nil
	default:
		return NodeNone, fmt.Errorf("invalid ReadyToRunNodeCmdType %s", s)
	}
}

type DependencyRule struct {
	Cmd              ReadyToRunNodeCmdType `json:"cmd" yaml:"cmd"`
	RawExpression    string                `json:"expression" yaml:"expression"`
	ParsedExpression ast.Expr
}

type DependencyPolicyDef struct {
	EventPriorityOrderString string           `json:"event_priority_order" yaml:"event_priority_order"`
	IsDefault                bool             `json:"is_default" yaml:"is_default"`
	Rules                    []DependencyRule `json:"rules" yaml:"rules"`
	OrderIdxDef              IdxDef
}

func NewFieldRefsFromNodeEvent() *FieldRefs {
	return &FieldRefs{
		{TableName: wfmodel.DependencyNodeRunStatusTableName, FieldName: "run_id", FieldType: evalcapi.FieldTypeInt},
		{TableName: wfmodel.DependencyNodeRunStatusTableName, FieldName: "run_is_current", FieldType: evalcapi.FieldTypeBool},
		{TableName: wfmodel.DependencyNodeRunStatusTableName, FieldName: "run_status", FieldType: evalcapi.FieldTypeInt},
		{TableName: wfmodel.DependencyNodeRunStatusTableName, FieldName: "node_status", FieldType: evalcapi.FieldTypeInt},
	}
}

func (polDef *DependencyPolicyDef) Deserialize(rawPol json.RawMessage, scriptType ScriptType) error {
	if err := JsonOrYamlUnmarshal(scriptType, rawPol, polDef); err != nil {
		return fmt.Errorf("cannot unmarshal dependency policy: [%s]", err.Error())
	}

	if err := polDef.parseEventPriorityOrderString(); err != nil {
		return err
	}

	vars := wfmodel.NewVarsFromDepCtx(wfmodel.DependencyNodeRunStatus{})
	for ruleIdx := 0; ruleIdx < len(polDef.Rules); ruleIdx++ {
		usedFieldRefs := FieldRefs{}
		var err error
		polDef.Rules[ruleIdx].ParsedExpression, err = ParseRawGolangExpressionStringAndHarvestFieldRefs(polDef.Rules[ruleIdx].RawExpression, &usedFieldRefs)
		if err != nil {
			return fmt.Errorf("cannot parse rule expression '%s': %s", polDef.Rules[ruleIdx].RawExpression, err.Error())
		}

		for _, fr := range usedFieldRefs {
			fieldSubMap, ok := vars[fr.TableName]
			if !ok {
				return fmt.Errorf("cannot parse rule expression '%s': all fields must be prefixed with one of these : %s", polDef.Rules[ruleIdx].RawExpression, vars.Tables())
			}
			if _, ok := fieldSubMap[fr.FieldName]; !ok {
				return fmt.Errorf("cannot parse rule expression '%s': field %s.%s not found, available fields are %s", polDef.Rules[ruleIdx].RawExpression, fr.TableName, fr.FieldName, vars.Names())
			}
		}
	}
	return nil
}

func (polDef *DependencyPolicyDef) parseEventPriorityOrderString() error {
	idxDefMap := IdxDefMap{}
	rawIndexes := map[string]string{"order_by": fmt.Sprintf("non_unique(%s)", polDef.EventPriorityOrderString)}
	if err := idxDefMap.parseRawIndexDefMap(rawIndexes, NewFieldRefsFromNodeEvent()); err != nil {
		return fmt.Errorf("cannot parse event order string '%s': %s", polDef.EventPriorityOrderString, err.Error())
	}
	polDef.OrderIdxDef = *idxDefMap["order_by"]

	return nil
}

func (polDef *DependencyPolicyDef) evalRuleExpressionsAndCheckType() error {
	vars := wfmodel.NewVarsFromDepCtx(wfmodel.DependencyNodeRunStatus{})
	eCtx := eval.NewPlainEvalCtx(evalcapi.CapillariesEvalFunctions, evalcapi.CapillariesEvalConstants, vars)
	for ruleIdx, rule := range polDef.Rules {
		result, err := eCtx.Eval(rule.ParsedExpression)
		if err != nil {
			return fmt.Errorf("invalid rule %d expression '%s': %s", ruleIdx, rule.RawExpression, err.Error())
		}
		if err := CheckValueType(result, evalcapi.FieldTypeBool); err != nil {
			return fmt.Errorf("invalid rule %d expression '%s' type: %s", ruleIdx, rule.RawExpression, err.Error())
		}
	}
	return nil
}
