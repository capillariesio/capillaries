package sc

import (
	"encoding/json"
	"fmt"
	"go/ast"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type ReadyToRunNodeCmdType string

const (
	NodeNone ReadyToRunNodeCmdType = "none"
	NodeGo   ReadyToRunNodeCmdType = "go"
	NodeWait ReadyToRunNodeCmdType = "wait"
	NodeNogo ReadyToRunNodeCmdType = "nogo"
)

type DependencyRule struct {
	Cmd              ReadyToRunNodeCmdType `json:"cmd"`
	RawExpression    string                `json:"expression"`
	ParsedExpression ast.Expr
}

// type EventPriorityOrderDirection string

// const (
// 	EventSortAsc     EventPriorityOrderDirection = "asc"
// 	EventSortDesc    EventPriorityOrderDirection = "desc"
// 	EventSortUnknown EventPriorityOrderDirection = "unknown"
// )

// type EventPriorityOrderField struct {
// 	FieldName string
// 	Direction EventPriorityOrderDirection
// }

type DependencyPolicyDef struct {
	EventPriorityOrderString string           `json:"event_priority_order"`
	IsDefault                bool             `json:"is_default"`
	Rules                    []DependencyRule `json:"rules"`
	OrderIdxDef              IdxDef
	//EventPriorityOrder       []EventPriorityOrderField
}

func NewFieldRefsFromNodeEvent() *FieldRefs {
	return &FieldRefs{
		{TableName: wfmodel.DependencyNodeEventTableName, FieldName: "run_id", FieldType: FieldTypeInt},
		{TableName: wfmodel.DependencyNodeEventTableName, FieldName: "run_is_current", FieldType: FieldTypeBool},
		{TableName: wfmodel.DependencyNodeEventTableName, FieldName: "run_start_ts", FieldType: FieldTypeDateTime},
		{TableName: wfmodel.DependencyNodeEventTableName, FieldName: "run_status", FieldType: FieldTypeInt},
		{TableName: wfmodel.DependencyNodeEventTableName, FieldName: "run_status_ts", FieldType: FieldTypeDateTime},
		{TableName: wfmodel.DependencyNodeEventTableName, FieldName: "node_is_started", FieldType: FieldTypeBool},
		{TableName: wfmodel.DependencyNodeEventTableName, FieldName: "node_start_ts", FieldType: FieldTypeDateTime},
		{TableName: wfmodel.DependencyNodeEventTableName, FieldName: "node_status", FieldType: FieldTypeInt},
		{TableName: wfmodel.DependencyNodeEventTableName, FieldName: "node_status_ts", FieldType: FieldTypeDateTime}}
}

func (polDef *DependencyPolicyDef) Deserialize(rawPol json.RawMessage) error {
	var err error
	if err = json.Unmarshal(rawPol, polDef); err != nil {
		return fmt.Errorf("cannot unmarshal dependency policy: [%s]", err.Error())
	}

	if err = polDef.parseEventPriorityOrderString(); err != nil {
		return err
	}

	vars := wfmodel.NewVarsFromDepCtx(0, wfmodel.DependencyNodeEvent{})
	for ruleIdx := 0; ruleIdx < len(polDef.Rules); ruleIdx++ {
		usedFieldRefs := FieldRefs{}
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

	// vars := NewVarsFromDepCtx(0, wfmodel.DependencyNodeEvent{})
	// fieldStrings := strings.Split(polDef.EventPriorityOrderString, ",")
	// polDef.EventPriorityOrder = make([]EventPriorityOrderField, len(fieldStrings))
	// for fieldIdx, fieldString := range fieldStrings {
	// 	fieldExp, err := parser.ParseExpr(fieldString)
	// 	if err != nil {
	// 		return fmt.Errorf("cannot parse event field descriptor '%s': %s", fieldString, err.Error())
	// 	}
	// 	switch typedExp := fieldExp.(type) {
	// 	case *ast.CallExpr:
	// 		identExp, _ := typedExp.Fun.(*ast.Ident)
	// 		polDef.EventPriorityOrder[fieldIdx] = EventPriorityOrderField{FieldName: identExp.Name, Direction: EventSortUnknown}
	// 		for _, modifierExp := range typedExp.Args {
	// 			switch modifierExpType := modifierExp.(type) {
	// 			case *ast.Ident:
	// 				switch modifierExpType.Name {
	// 				case string(EventSortAsc):
	// 					polDef.EventPriorityOrder[fieldIdx].Direction = EventSortAsc
	// 				case string(EventSortDesc):
	// 					polDef.EventPriorityOrder[fieldIdx].Direction = EventSortDesc
	// 				default:
	// 					return fmt.Errorf("invalid direction modifier '%s'", modifierExpType.Name)
	// 				}
	// 			default:
	// 				return fmt.Errorf("cannot parse direction modifier in event field descriptor '%s', expected asc or desc", fieldString)
	// 			}
	// 		}
	// 		if polDef.EventPriorityOrder[fieldIdx].Direction == EventSortUnknown {
	// 			return fmt.Errorf("missing direction modifier in '%s', required %s or %s", fieldString, EventSortAsc, EventSortDesc)
	// 		}
	// 	default:
	// 		return fmt.Errorf("cannot parse event field descriptor '%s', it must be event_field_name(asc|desc) where event_field_name is one of these: %s", fieldString, vars.NamesByTable(wfmodel.DependencyNodeEventTableName))
	// 	}
	// }

	// // Verify that order expression contains only event fields
	// eventFieldMap := vars[wfmodel.DependencyNodeEventTableName]
	// for _, fld := range polDef.EventPriorityOrder {
	// 	if _, ok := eventFieldMap[fld.FieldName]; !ok {
	// 		return fmt.Errorf("cannot parse event field descriptor '%s', available fields are %s", fld.FieldName, vars.NamesByTable(wfmodel.DependencyNodeEventTableName))
	// 	}
	// }

	return nil
}

func (polDef *DependencyPolicyDef) evalRuleExpressionsAndCheckType() error {
	vars := wfmodel.NewVarsFromDepCtx(0, wfmodel.DependencyNodeEvent{})
	eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, &vars)
	for ruleIdx, rule := range polDef.Rules {
		result, err := eCtx.Eval(rule.ParsedExpression)
		if err != nil {
			return fmt.Errorf("invalid rule %d expression '%s': %s", ruleIdx, rule.RawExpression, err.Error())
		}
		if err := CheckValueType(result, FieldTypeBool); err != nil {
			return fmt.Errorf("invalid rule %d expression '%s' type: %s", ruleIdx, rule.RawExpression, err.Error())
		}
	}
	return nil
}
