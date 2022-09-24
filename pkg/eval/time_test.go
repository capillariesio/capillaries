package eval

import (
	"fmt"
	"go/parser"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeFunctions(t *testing.T) {
	testTime := time.Date(2001, 1, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200))
	varValuesMap := VarValuesMap{
		"t": map[string]interface{}{
			"test_time": testTime}}
	assertEqual(t, `time.Parse("2006-01-02T15:04:05.000-0700","2001-01-01T01:01:01.100-0200")`, testTime, varValuesMap)
	assertEvalError(t, `time.Parse("2006-01-02T15:04:05.000-0700","2001-01-01T01:01:01.100-0200","aaa")`, "cannot evaluate time.Parse(), requires 2 args, 3 supplied", varValuesMap)
	assertEvalError(t, `time.Parse("2006-01-02T15:04:05.000-0700",123)`, "cannot evaluate time.Parse(), invalid args [2006-01-02T15:04:05.000-0700 123]", varValuesMap)
	assertEvalError(t, `time.Parse("2006-01-02T15:04:05.000-0700","2001-01-01T01:01:01")`, `parsing time "2001-01-01T01:01:01" as "2006-01-02T15:04:05.000-0700": cannot parse "" as ".000"`, varValuesMap)

	assertEqual(t, `time.Date(2001, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200))`, testTime, varValuesMap)
	assertEvalError(t, `time.Date(2001, 354, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200))`, "cannot evaluate time.Date(), invalid args [2001 354 1 1 1 1 100000000 ]", varValuesMap)
	assertEvalError(t, `time.Date(2001, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200), "extraparam")`, "cannot evaluate time.Date(), requires 8 args, 9 supplied", varValuesMap)

	assertEqual(t, `time.DiffMilli(time.Date(2001, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, int64(0), varValuesMap)
	assertEqual(t, `time.DiffMilli(time.Date(2002, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, int64(31536000000), varValuesMap)
	assertEqual(t, `time.DiffMilli(time.Date(2000, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, int64(-31622400000), varValuesMap)

	assertEqual(t, `time.Unix(t.test_time)`, testTime.Unix(), varValuesMap)

	assertEqual(t, `time.UnixMilli(t.test_time)`, testTime.UnixMilli(), varValuesMap)
}

func TestNow(t *testing.T) {

	exp, err1 := parser.ParseExpr(`time.Now()`)
	if err1 != nil {
		t.Error(fmt.Errorf("cannot parse Now(): %s", err1.Error()))
		return
	}
	eCtx := NewPlainEvalCtxWithVars(false, &VarValuesMap{})
	result, err2 := eCtx.Eval(exp)
	if err2 != nil {
		t.Error(fmt.Errorf("cannot eval Now(): %s", err2.Error()))
		return
	}

	resultTime, _ := result.(time.Time)

	assert.True(t, time.Since(resultTime).Milliseconds() < 500)
}
