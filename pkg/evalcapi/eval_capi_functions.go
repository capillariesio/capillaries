package evalcapi

import "github.com/capillariesio/capillaries/pkg/eval"

var CapillariesEvalFunctions = map[string]eval.EvalFunction{
	"math.Sqrt":          callMathSqrt,
	"math.Round":         callMathRound,
	"len":                callLen,
	"string":             callString,
	"float":              callFloat,
	"int":                callInt,
	"decimal2":           callDecimal2,
	"int.iif":            callIntIif,
	"float.iif":          callFloatIif,
	"decimal2.iif":       callDecimal2Iif,
	"string.iif":         callStringIif,
	"time.iif":           callTimeIif,
	"time.Parse":         callTimeParse,
	"time.Format":        callTimeFormat,
	"time.Date":          callTimeDate,
	"time.Now":           callTimeNow,
	"time.Unix":          callTimeUnix,
	"time.UnixMilli":     callTimeUnixMilli,
	"time.DiffMilli":     callTimeDiffMilli,
	"time.Before":        callTimeBefore,
	"time.After":         callTimeAfter,
	"time.FixedZone":     callTimeFixedZone,
	"re.MatchString":     callReMatchString,
	"strings.ReplaceAll": callStringsReplaceAll,
	"fmt.Sprintf":        callFmtSprintf,
}
