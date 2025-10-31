package dpc

import (
	"fmt"
	"sort"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

func CheckDependencyPolicyAgainstNodeEventList(logger *l.CapiLogger, fullBatchId string, targetNodeDepPol *sc.DependencyPolicyDef, events wfmodel.DependencyNodeEvents) (sc.ReadyToRunNodeCmdType, int16, int, error) {
	var err error

	for eventIdx := 0; eventIdx < len(events); eventIdx++ {
		vars := wfmodel.NewVarsFromDepCtx(events[eventIdx])
		events[eventIdx].SortKey, err = sc.BuildKey(vars[wfmodel.DependencyNodeEventTableName], &targetNodeDepPol.OrderIdxDef)
		if err != nil {
			return sc.NodeNogo, 0, -1, fmt.Errorf("unexpectedly, cannot build key to sort events: %s", err.Error())
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].SortKey < events[j].SortKey })

	for eventIdx := 0; eventIdx < len(events); eventIdx++ {
		vars := wfmodel.NewVarsFromDepCtx(events[eventIdx])
		eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, &vars)
		for ruleIdx, rule := range targetNodeDepPol.Rules {
			ruleMatched, err := eCtx.Eval(rule.ParsedExpression)
			if err != nil {
				return sc.NodeNogo, 0, -1, fmt.Errorf("cannot check rule %d '%s' against event %s for batch %s, eval failed: %s", ruleIdx, rule.RawExpression, events[eventIdx].ToString(), fullBatchId, err.Error())
			}
			ruleMatchedBool, ok := ruleMatched.(bool)
			if !ok {
				return sc.NodeNogo, 0, -1, fmt.Errorf("cannot check rule %d '%s' against event %s for batch %s: expected result type was bool, got %T", ruleIdx, rule.RawExpression, events[eventIdx].ToString(), fullBatchId, ruleMatched)
			}
			if ruleMatchedBool {
				if logger != nil {
					logger.Debug("matched rule %d(%s) '%s' against event %d(%s) for batch %s, all events %s", ruleIdx, rule.Cmd, rule.RawExpression, eventIdx, events[eventIdx].ToString(), fullBatchId, events.ToString())
				}
				return rule.Cmd, events[eventIdx].RunId, ruleIdx, nil
			}
		}
	}

	// Extremely rare case, saw it in Oct 2025, log says:
	// checked all dependency nodes for 04_loan_payment_summaries, commands are [nogo go], run ids are [0 1], finalCmd is nogo
	// some dependency nodes for fannie_mae_quicktest_local_fs_multi/4/04_loan_payment_summaries/9 are in bad state, or runs executing dependency nodes were stopped/invalidated, will not run this node; for details, check rules in dependency_policies and previous runs history

	// dependencyNodeCmds[nodeIdx], dependencyRunIds[nodeIdx], checkerLogMsg, err = dpc.CheckDependencyPolicyAgainstNodeEventList() call
	// returned nogo, 0, "...", nil because of this return we had here:
	// return sc.NodeNogo, 0, fmt.Sprintf("no rules matched against events %s", events.ToString()), nil

	// We are checking a batch for 04_loan_payment_summaries. It depends on 01_read_payments (run 1) and 02_loan_ids (run 2).
	// returned dependencyRunIds[0] is zero, which is nonsense. dependencyRunIds[1] is 1 which means 01_read_payments was good.
	// Another dependency is 02_loan_ids, which was supposed to have run 2.
	// So, something is off with the nodeEventListMap["02_loan_ids"].
	// This rule was supposed to kick in, but it did not (run 2, node 02_loan_ids)
	// {"cmd": "go",   "expression": "e.run_is_current == false && e.run_final_status == wfmodel.RunComplete && e.node_status == wfmodel.NodeBatchSuccess"	},
	// Could there be a case that e.run_final_status == wfmodel.RunComplete but e.node_status == wfmodel.NodeBatchStart  because of Cassandra's eventual consistency?
	// Like, we wrote to Cassandra nodeStatus = Complete, then, runStatus = RunComplete, but when we read them back, we see only runStatus = RunComplete and nodeStatus = CompleteNodeBatchStart?
	// If this is the case, we just wait for Cassandra to settle.

	// len(nodeEventListMap[depNodeName]) check in the caller guarantees we have at leas one event in the list. All events in the list have the same runId.
	if logger != nil {
		logger.Warn("assuming wait for batch %s, no rules matched against events %s", fullBatchId, events.ToString())
	}

	// In two words: we do not know what is going on exactly, we assume that if we wait, the db will come to some coherent state
	return sc.NodeWait, events[0].RunId, -1, nil
}
