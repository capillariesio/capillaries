package proc

import (
	"github.com/kleineshertz/capillaries/pkg/ctx"
	"github.com/kleineshertz/capillaries/pkg/eval"
	"github.com/kleineshertz/capillaries/pkg/l"
)

type CustomProcessorRunner interface {
	Run(logger *l.Logger, pCtx *ctx.MessageProcessingContext, rsIn *Rowset, flushVarsArray func(varsArray []*eval.VarValuesMap, varsArrayCount int) error) error
}
