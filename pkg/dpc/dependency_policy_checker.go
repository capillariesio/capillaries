package dpc

import (
	"fmt"
	"sort"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

func CheckDependencyPolicyAgainstNodeEventList(targetNodeDepPol *sc.DependencyPolicyDef, events wfmodel.DependencyNodeEvents) (sc.ReadyToRunNodeCmdType, int16, string, error) {
	var err error

	for eventIdx := 0; eventIdx < len(events); eventIdx++ {
		vars := wfmodel.NewVarsFromDepCtx(0, events[eventIdx])
		events[eventIdx].SortKey, err = sc.BuildKey(vars[wfmodel.DependencyNodeEventTableName], &targetNodeDepPol.OrderIdxDef)
		if err != nil {
			return sc.NodeNogo, 0, "", fmt.Errorf("cannot sort events: %s", err.Error())
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].SortKey < events[j].SortKey })

	for eventIdx := 0; eventIdx < len(events); eventIdx++ {
		vars := wfmodel.NewVarsFromDepCtx(0, events[eventIdx])
		eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, &vars)
		for ruleIdx, rule := range targetNodeDepPol.Rules {
			ruleMatched, err := eCtx.Eval(rule.ParsedExpression)
			if err != nil {
				return sc.NodeNogo, 0, "", fmt.Errorf("cannot check rule %d '%s' against event %s: %s", ruleIdx, rule.RawExpression, events[eventIdx].ToString(), err.Error())
			}
			ruleMatchedBool, ok := ruleMatched.(bool)
			if !ok {
				return sc.NodeNogo, 0, "", fmt.Errorf("failed checking rule %d '%s' against event %s: expected result type was bool, got %T", ruleIdx, rule.RawExpression, events[eventIdx].ToString(), ruleMatched)
			}
			if ruleMatchedBool {
				return rule.Cmd, events[eventIdx].RunId, fmt.Sprintf("matched rule %d(%s) '%s' against event %d %s. All events %s", ruleIdx, rule.Cmd, rule.RawExpression, eventIdx, events[eventIdx].ToString(), events.ToString()), nil
			}
		}
	}

	return sc.NodeNogo, 0, fmt.Sprintf("no rules matched against events %s", events.ToString()), nil
}
