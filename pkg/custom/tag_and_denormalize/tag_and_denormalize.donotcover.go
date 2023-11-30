package custom

import (
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/proc"
)

func (procDef *TagAndDenormalizeProcessorDef) Run(logger *l.Logger, pCtx *ctx.MessageProcessingContext, rsIn *proc.Rowset, flushVarsArray func(varsArray []*eval.VarValuesMap, varsArrayCount int) error) error {
	logger.PushF("custom.TagAndDenormalizeProcessorDef.Run")
	defer logger.PopF()

	return procDef.tagAndDenormalize(rsIn, flushVarsArray)
}
