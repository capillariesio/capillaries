package eval

import (
	"fmt"
	"go/parser"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGeneralTimeFunctions(t *testing.T) {
	testTime := time.Date(2001, 1, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200))
	testTimeUtc := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	varValuesMap := VarValuesMap{"t": map[string]any{"test_time": testTime}}

	assertEqual(t, `time.Parse("2006-01-02T15:04:05.000-0700","2001-01-01T01:01:01.100-0200")`, testTime, varValuesMap)
	assertEvalError(t, `time.Parse("2006-01-02T15:04:05.000-0700","2001-01-01T01:01:01.100-0200","aaa")`, "cannot evaluate time.Parse(), requires 2 args, 3 supplied", varValuesMap)
	assertEvalError(t, `time.Parse("2006-01-02T15:04:05.000-0700",123)`, "cannot evaluate time.Parse(), invalid args [2006-01-02T15:04:05.000-0700 123]", varValuesMap)
	assertEvalError(t, `time.Parse("2006-01-02T15:04:05.000-0700","2001-01-01T01:01:01")`, `parsing time "2001-01-01T01:01:01" as "2006-01-02T15:04:05.000-0700": cannot parse "" as ".000"`, varValuesMap)
	assertEqual(t, `time.Parse("2006-01-02","2001-01-01")`, testTimeUtc, varValuesMap)

	assertEqual(t, `time.Format(time.Date(2001, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), "2006-01-02T15:04:05.000-0700")`, testTime.Format("2006-01-02T15:04:05.000-0700"), varValuesMap)
	assertEvalError(t, `time.Format(time.Date(2001, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)))`, "cannot evaluate time.Format(), requires 2 args, 1 supplied", varValuesMap)
	assertEvalError(t, `time.Format("some_bad_param", "2006-01-02T15:04:05.000-0700")`, "cannot evaluate time.Format(), invalid args [some_bad_param 2006-01-02T15:04:05.000-0700]", varValuesMap)

	assertEqual(t, `time.Date(2001, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200))`, testTime, varValuesMap)
	assertEvalError(t, `time.Date(2001, 354, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200))`, "cannot evaluate time.Date(), invalid args [2001 354 1 1 1 1 100000000 ]", varValuesMap)
	assertEvalError(t, `time.Date(2001, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200), "extraparam")`, "cannot evaluate time.Date(), requires 8 args, 9 supplied", varValuesMap)

	assertEvalError(t, `time.FixedZone("")`, "cannot evaluate time.FixedZone(), requires 2 args, 1 supplied", varValuesMap)
	assertEvalError(t, `time.FixedZone("", "some_bad_param")`, "cannot evaluate time.FixedZone(), invalid args [ some_bad_param]", varValuesMap)

	assertEqual(t, `time.DiffMilli(time.Date(2001, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, int64(0), varValuesMap)
	assertEqual(t, `time.DiffMilli(time.Date(2002, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, int64(31536000000), varValuesMap)
	assertEqual(t, `time.DiffMilli(time.Date(2000, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, int64(-31622400000), varValuesMap)
	assertEvalError(t, `time.DiffMilli(1)`, "cannot evaluate time.DiffMilli(), requires 2 args, 1 supplied", varValuesMap)
	assertEvalError(t, `time.DiffMilli("some_bad_param", t.test_time)`, "cannot evaluate time.DiffMilli(), invalid args [some_bad_param 2001-01-01 01:01:01.1 -0200 -0200]", varValuesMap)

	assertEqual(t, `time.Before(time.Date(2000, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, true, varValuesMap)
	assertEqual(t, `time.Before(time.Date(2002, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, false, varValuesMap)
	assertEvalError(t, `time.Before()`, "cannot evaluate time.Before(), requires 2 args, 0 supplied", varValuesMap)
	assertEvalError(t, `time.Before("some_bad_param", t.test_time)`, "cannot evaluate time.Before(), invalid args [some_bad_param 2001-01-01 01:01:01.1 -0200 -0200]", varValuesMap)

	assertEqual(t, `time.After(time.Date(2002, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, true, varValuesMap)
	assertEqual(t, `time.After(time.Date(2000, time.January, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200)), t.test_time)`, false, varValuesMap)
	assertEvalError(t, `time.After()`, "cannot evaluate time.After(), requires 2 args, 0 supplied", varValuesMap)
	assertEvalError(t, `time.After("some_bad_param", t.test_time)`, "cannot evaluate time.After(), invalid args [some_bad_param 2001-01-01 01:01:01.1 -0200 -0200]", varValuesMap)

	assertEqual(t, `time.Unix(t.test_time)`, testTime.Unix(), varValuesMap)
	assertEvalError(t, `time.Unix()`, "cannot evaluate time.Unix(), requires 1 args, 0 supplied", varValuesMap)
	assertEvalError(t, `time.Unix("some_bad_param")`, "cannot evaluate time.Unix(), invalid args [some_bad_param]", varValuesMap)

	assertEqual(t, `time.UnixMilli(t.test_time)`, testTime.UnixMilli(), varValuesMap)
	assertEvalError(t, `time.UnixMilli()`, "cannot evaluate time.UnixMilli(), requires 1 args, 0 supplied", varValuesMap)
	assertEvalError(t, `time.UnixMilli("some_bad_param")`, "cannot evaluate time.UnixMilli(), invalid args [some_bad_param]", varValuesMap)
}

func TestNow(t *testing.T) {

	exp, err1 := parser.ParseExpr(`time.Now()`)
	if err1 != nil {
		t.Error(fmt.Errorf("cannot parse Now(): %s", err1.Error()))
		return
	}
	eCtx := NewPlainEvalCtxWithVars(AggFuncDisabled, &VarValuesMap{})
	result, err2 := eCtx.Eval(exp)
	if err2 != nil {
		t.Error(fmt.Errorf("cannot eval Now(): %s", err2.Error()))
		return
	}

	resultTime, ok := result.(time.Time)
	assert.True(t, ok)
	assert.True(t, time.Since(resultTime).Milliseconds() < 500)
	assertEvalError(t, `time.Now(1)`, "cannot evaluate time.Now(), requires 0 args, 1 supplied", VarValuesMap{})
}
