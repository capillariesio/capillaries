package proc

import (
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/l"
)

type CustomProcessorRunner interface {
	Run(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, rsIn *Rowset, flushVarsArray func(logger *l.CapiLogger, varsArray []*eval.VarValuesMap, varsArrayCount int) error) error
}
