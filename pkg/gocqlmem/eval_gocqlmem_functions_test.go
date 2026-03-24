package gocqlmem

import (
	"fmt"
	"go/ast"
	"go/parser"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gopkg.in/inf.v0"
)

type testValue struct {
	valIn    any
	errorOut string
	valOut   any
}

type testPairValue struct {
	valIn1   any
	valIn2   any
	errorOut string
	valOut   any
}

func TestCast(t *testing.T) {
	var err error
	var exp ast.Expr
	var eCtx *eval.EvalCtx
	var val any

	exp, _ = parser.ParseExpr(`cast()`)
	eCtx = eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, nil)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate cast(), requires 2 args, 0 supplied", err.Error())

	vars := eval.VarValuesMap{
		"": {
			"f1": nil,
		},
	}

	exp, _ = parser.ParseExpr(`cast(f1,DATATYPE)`)
	eCtx = eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, vars)

	testValues := []testPairValue{

		// bad
		{"hello", "baddatatype", "cannot convert cast() arg baddatatype to DataType", 0},
		{int(1), DataTypeUuid, "cannot cast int 1 to uuid", 0},
		{float64(1.1), DataTypeUuid, "cannot cast float 1.1 to uuid", 0},
		{true, DataTypeUuid, "cannot cast bool true to uuid", 0},
		{decimal.NewFromFloat(1.1), DataTypeUuid, "cannot cast decimal 1.1 to uuid", 0},
		{"hello", DataTypeUuid, "cannot cast string hello to uuid", 0},

		// to int

		{nil, DataTypeTinyint, "cannot cast <nil> to tinyint, unsupported source type", 0},
		{"1", DataTypeTinyint, "cannot cast string 1 to tinyint, unsupported source type", 0},
		{int(101), DataTypeTinyint, "", int64(101)},
		{int8(12), DataTypeTinyint, "", int64(12)},
		{int16(103), DataTypeTinyint, "", int64(103)},
		{int32(104), DataTypeTinyint, "", int64(104)},
		{int64(105), DataTypeTinyint, "", int64(105)},
		{float32(106.1), DataTypeTinyint, "", int64(106)},
		{float64(107.1), DataTypeTinyint, "", int64(107)},
		{decimal.NewFromFloat(108.1), DataTypeTinyint, "", int64(108)},

		{nil, DataTypeInt, "cannot cast <nil> to int, unsupported source type", 0},
		{"1", DataTypeInt, "cannot cast string 1 to int, unsupported source type", 0},
		{int(201), DataTypeInt, "", int64(201)},
		{int8(22), DataTypeInt, "", int64(22)},
		{int16(203), DataTypeInt, "", int64(203)},
		{int32(204), DataTypeInt, "", int64(204)},
		{int64(205), DataTypeInt, "", int64(205)},
		{float32(206.1), DataTypeInt, "", int64(206)},
		{float64(207.1), DataTypeInt, "", int64(207)},
		{decimal.NewFromFloat(208.1), DataTypeInt, "", int64(208)},

		{nil, DataTypeSmallint, "cannot cast <nil> to smallint, unsupported source type", 0},
		{"1", DataTypeSmallint, "cannot cast string 1 to smallint, unsupported source type", 0},
		{int(301), DataTypeSmallint, "", int64(301)},
		{int8(32), DataTypeSmallint, "", int64(32)},
		{int16(303), DataTypeSmallint, "", int64(303)},
		{int32(304), DataTypeSmallint, "", int64(304)},
		{int64(305), DataTypeSmallint, "", int64(305)},
		{float32(306.1), DataTypeSmallint, "", int64(306)},
		{float64(307.1), DataTypeSmallint, "", int64(307)},
		{decimal.NewFromFloat(308.1), DataTypeSmallint, "", int64(308)},

		{nil, DataTypeBigint, "cannot cast <nil> to bigint, unsupported source type", 0},
		{"1", DataTypeBigint, "cannot cast string 1 to bigint, unsupported source type", 0},
		{int(401), DataTypeBigint, "", int64(401)},
		{int8(42), DataTypeBigint, "", int64(42)},
		{int16(403), DataTypeBigint, "", int64(403)},
		{int32(404), DataTypeBigint, "", int64(404)},
		{int64(405), DataTypeBigint, "", int64(405)},
		{float32(406.1), DataTypeBigint, "", int64(406)},
		{float64(407.1), DataTypeBigint, "", int64(407)},
		{decimal.NewFromFloat(408.1), DataTypeBigint, "", int64(408)},

		{nil, DataTypeVarint, "cannot cast <nil> to varint, unsupported source type", 0},
		{"1", DataTypeVarint, "cannot cast string 1 to varint, unsupported source type", 0},
		{int(501), DataTypeVarint, "", int64(501)},
		{int8(52), DataTypeVarint, "", int64(52)},
		{int16(503), DataTypeVarint, "", int64(503)},
		{int32(504), DataTypeVarint, "", int64(504)},
		{int64(505), DataTypeVarint, "", int64(505)},
		{float32(506.1), DataTypeVarint, "", int64(506)},
		{float64(507.1), DataTypeVarint, "", int64(507)},
		{decimal.NewFromFloat(508.1), DataTypeVarint, "", int64(508)},

		// to float

		{nil, DataTypeFloat, "cannot cast <nil> to float, unsupported source type", 0},
		{"1", DataTypeFloat, "cannot cast string 1 to float, unsupported source type", 0},
		{int(101), DataTypeFloat, "", float64(101)},
		{int8(12), DataTypeFloat, "", float64(12)},
		{int16(103), DataTypeFloat, "", float64(103)},
		{int32(104), DataTypeFloat, "", float64(104)},
		{int64(105), DataTypeFloat, "", float64(105)},
		{float32(106.1), DataTypeFloat, "", float64(106.0999984741211)},
		{float64(107.2), DataTypeFloat, "", float64(107.2)},
		{decimal.NewFromFloat(108.3), DataTypeFloat, "", float64(108.3)},

		{nil, DataTypeDouble, "cannot cast <nil> to double, unsupported source type", 0},
		{"1", DataTypeDouble, "cannot cast string 1 to double, unsupported source type", 0},
		{int(101), DataTypeDouble, "", float64(101)},
		{int8(12), DataTypeDouble, "", float64(12)},
		{int16(103), DataTypeDouble, "", float64(103)},
		{int32(104), DataTypeDouble, "", float64(104)},
		{int64(105), DataTypeDouble, "", float64(105)},
		{float32(106.4), DataTypeDouble, "", float64(106.4000015258789)},
		{float64(107.5), DataTypeDouble, "", float64(107.5)},
		{decimal.NewFromFloat(108.6), DataTypeDouble, "", float64(108.6)},

		// to decimal

		{nil, DataTypeDecimal, "cannot cast <nil> to decimal, unsupported source type", 0},
		{"1", DataTypeDecimal, "cannot cast string 1 to decimal, unsupported source type", 0},
		{int(101), DataTypeDecimal, "", *inf.NewDec(101, 0)},
		{int8(12), DataTypeDecimal, "", *inf.NewDec(12, 0)},
		{int16(103), DataTypeDecimal, "", *inf.NewDec(103, 0)},
		{int32(104), DataTypeDecimal, "", *inf.NewDec(104, 0)},
		{int64(105), DataTypeDecimal, "", *inf.NewDec(105, 0)},
		{float32(106.1), DataTypeDecimal, "", decimal.NewFromFloat(106.0999984741211)},
		{float64(107.2), DataTypeDecimal, "", decimal.NewFromFloat(107.2)},
		{decimal.NewFromFloat(108.3), DataTypeDecimal, "", decimal.NewFromFloat(108.3)},

		// to text, varchar

		{nil, DataTypeText, "cannot cast <nil> to text, unsupported source type", 0},
		{"1", DataTypeText, "", "1"},
		{int(101), DataTypeText, "", "101"},
		{int8(12), DataTypeText, "", "12"},
		{int16(103), DataTypeText, "", "103"},
		{int32(104), DataTypeText, "", "104"},
		{int64(105), DataTypeText, "", "105"},
		{float32(106.1), DataTypeText, "", "106.0999984741211"},
		{float64(107.2), DataTypeText, "", "107.2"},
		{decimal.NewFromFloat(108.3), DataTypeText, "", "108.3"},
		{true, DataTypeText, "", "TRUE"},
		{false, DataTypeText, "", "FALSE"},

		{nil, DataTypeVarchar, "cannot cast <nil> to varchar, unsupported source type", 0},
		{"2", DataTypeVarchar, "", "2"},
		{int(201), DataTypeVarchar, "", "201"},
		{int8(22), DataTypeVarchar, "", "22"},
		{int16(203), DataTypeVarchar, "", "203"},
		{int32(204), DataTypeVarchar, "", "204"},
		{int64(205), DataTypeVarchar, "", "205"},
		{float32(206.1), DataTypeVarchar, "", "206.10000610351562"},
		{float64(207.2), DataTypeVarchar, "", "207.2"},
		{decimal.NewFromFloat(208.3), DataTypeVarchar, "", "208.3"},
		{true, DataTypeVarchar, "", "TRUE"},
		{false, DataTypeVarchar, "", "FALSE"},
	}

	for i := range len(testValues) {
		vars[""]["f1"] = testValues[i].valIn1
		vars[""]["DATATYPE"] = testValues[i].valIn2
		val, err = eCtx.Eval(exp)
		if testValues[i].errorOut == "" {
			assert.Nil(t, err)
			assert.Equal(t, testValues[i].valOut, val)
		} else {
			assert.Contains(t, testValues[i].errorOut, err.Error())
		}
	}
}

func TestToken(t *testing.T) {
	var err error
	var exp ast.Expr
	var eCtx *eval.EvalCtx
	var val any

	exp, _ = parser.ParseExpr(`token()`)
	eCtx = eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, nil)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate token(), requires 1 args, 0 supplied", err.Error())

	vars := eval.VarValuesMap{
		"": {
			"f1": nil,
		},
	}

	exp, _ = parser.ParseExpr(`token(f1)`)
	eCtx = eval.NewPlainEvalCtx(GocqlmemEvalFunctions, nil, vars)

	testValues := []testValue{
		{nil, "cannot token <nil>, unsupported source type <nil>", 0},
		{"hello", "", int64(-3758069500696749310)},
		{int(1), "", int64(19144387141682250)},
		{int8(2), "", int64(-2447670524089286488)},
		{int16(3), "", int64(6574508035858270988)},
		{int32(4), "", int64(-5469109305088493887)},
		{int64(5), "", int64(1140754268591781659)},
		{float32(1.1), "", int64(0x7165b2a5cd92e2a)}, // float64(float32(1.1)) !=  float64(1.1)
		{float64(1.2), "", int64(4067943965976384332)},
		{decimal.NewFromFloat(1.3), "", int64(5143825980438560564)},
		{true, "", int64(0x4403b7fb05c44a)},
	}

	for i := range len(testValues) {
		vars[""]["f1"] = testValues[i].valIn
		val, err = eCtx.Eval(exp)
		if testValues[i].errorOut == "" {
			assert.Nil(t, err)
			assert.Equal(t, testValues[i].valOut, val, fmt.Sprintf("value in: %v", testValues[i].valIn))
		} else {
			assert.Contains(t, testValues[i].errorOut, err.Error())
		}
	}
}

func TestDatetimeFunctions(t *testing.T) {
	var err error
	var exp ast.Expr
	var eCtx *eval.EvalCtx
	var val any

	eCtx = eval.NewPlainEvalCtx(GocqlmemEvalFunctions, nil, nil)

	// current_timestamp

	exp, _ = parser.ParseExpr(`current_timestamp(1)`)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate current_timestamp(), requires 0 args, 1 supplied", err.Error())

	exp, _ = parser.ParseExpr(`current_timestamp()`)
	now := time.Now().UnixMilli()
	val, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	assert.Equal(t, now/100, val.(int64)/100) // Give it some slack

	// current_date

	exp, _ = parser.ParseExpr(`current_date(1)`)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate current_date(), requires 0 args, 1 supplied", err.Error())

	exp, _ = parser.ParseExpr(`current_date()`)
	now = time.Now().UnixMilli()
	val, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	assert.Equal(t, int64(time.Since(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)).Hours())/24, val)

	// current_time

	exp, _ = parser.ParseExpr(`current_time(1)`)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate current_time(), requires 0 args, 1 supplied", err.Error())

	exp, _ = parser.ParseExpr(`current_time()`)
	now = time.Now().UnixMilli()
	val, err = eCtx.Eval(exp)
	ti := time.Now()
	curTime := int64(((ti.Hour()*60+ti.Minute())*60+ti.Second())*1000000000 + ti.Nanosecond())
	assert.Nil(t, err)
	assert.Equal(t, curTime/10000000, val.(int64)/10000000) // Give it some slack

	// toTimestamp

	timeToTest := time.Unix(1436832817, 476000000).UTC()
	sometimeuuid := gocql.UUIDFromTime(timeToTest)
	eCtx.SetVars(eval.VarValuesMap{"": {"sometimeuuid": sometimeuuid}})
	exp, _ = parser.ParseExpr(`totimestamp(sometimeuuid)`)
	val, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	assert.Equal(t, time.Date(2015, 7, 14, 0, 13, 37, 476000000, time.UTC), val)
	assert.Equal(t, time.Unix(1436832817, 476000000).UTC(), val)

	eCtx.SetVars(eval.VarValuesMap{"": {"somedate": int64(5)}})
	exp, _ = parser.ParseExpr(`totimestamp(somedate`)
	val, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	assert.Equal(t, time.Date(1970, 1, 6, 0, 0, 0, 0, time.UTC), val)
}

func TestMathFunctions(t *testing.T) {
	var err error
	var exp ast.Expr
	var eCtx *eval.EvalCtx
	var val any

	vars := eval.VarValuesMap{
		"": {
			"f1": nil,
		},
	}

	eCtx = eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, vars)

	// Abs

	exp, _ = parser.ParseExpr(`abs()`)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate abs(), requires 1 args, 0 supplied", err.Error())

	exp, _ = parser.ParseExpr(`abs(f1)`)

	testValues := []testValue{
		{nil, "cannot abs <nil>, unsupported source type <nil>", 0},
		{int(-1), "", int64(1)},
		{int8(1), "", int64(1)},
		{int16(1), "", int64(1)},
		{int32(1), "", int64(1)},
		{int64(1), "", int64(1)},
		{float32(-1.1), "", float64(1.100000023841858)},
		{float64(1.1), "", float64(1.1)},
		{decimal.NewFromFloat(-1.1), "", decimal.NewFromFloat(1.1)},
	}

	for i := range len(testValues) {
		vars[""]["f1"] = testValues[i].valIn
		val, err = eCtx.Eval(exp)
		if testValues[i].errorOut == "" {
			assert.Nil(t, err)
			assert.Equal(t, testValues[i].valOut, val)
		} else {
			assert.Contains(t, testValues[i].errorOut, err.Error())
		}
	}

	// Exp

	exp, _ = parser.ParseExpr(`exp()`)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate exp(), requires 1 args, 0 supplied", err.Error())

	exp, _ = parser.ParseExpr(`exp(f1)`)

	testValues = []testValue{
		{nil, "cannot exp <nil>, unsupported source type <nil>", 0},
		{int(-1), "", float64(0.36787944117144233)},
		{int8(-1), "", float64(0.36787944117144233)},
		{int16(-1), "", float64(0.36787944117144233)},
		{int32(-1), "", float64(0.36787944117144233)},
		{int64(-1), "", float64(0.36787944117144233)},
		{float32(-1.1), "", float64(0.33287107576181457)},
		{float64(-1.1), "", float64(0.3328710836980795)},
		{decimal.NewFromFloat(-1.1), "", decimal.NewFromFloat(0.33287108369807955329)},
	}

	for i := range len(testValues) {
		vars[""]["f1"] = testValues[i].valIn
		val, err = eCtx.Eval(exp)
		if testValues[i].errorOut == "" {
			assert.Nil(t, err)
			assert.Equal(t, fmt.Sprintf("%v", testValues[i].valOut), fmt.Sprintf("%v", val))
		} else {
			assert.Contains(t, testValues[i].errorOut, err.Error())
		}
	}

	// Log

	exp, _ = parser.ParseExpr(`log()`)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate log(), requires 1 args, 0 supplied", err.Error())

	exp, _ = parser.ParseExpr(`log(f1)`)

	testValues = []testValue{
		{nil, "cannot log <nil>, unsupported source type <nil>", 0},
		{int(1), "", float64(0)},
		{int8(1), "", float64(0)},
		{int16(1), "", float64(0)},
		{int32(1), "", float64(0)},
		{int64(1), "", float64(0)},
		{float32(1.1), "", float64(0.0953102014787409)},
		{float64(1.1), "", float64(0.09531017980432493)},
		{decimal.NewFromFloat(1.1), "", decimal.NewFromFloat(0.0953101798)},
	}

	for i := range len(testValues) {
		vars[""]["f1"] = testValues[i].valIn
		val, err = eCtx.Eval(exp)
		if testValues[i].errorOut == "" {
			assert.Nil(t, err)
			assert.Equal(t, testValues[i].valOut, val)
		} else {
			assert.Contains(t, testValues[i].errorOut, err.Error())
		}
	}

	// Log10

	exp, _ = parser.ParseExpr(`log10()`)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate log10(), requires 1 args, 0 supplied", err.Error())

	exp, _ = parser.ParseExpr(`log10(f1)`)

	testValues = []testValue{
		{nil, "cannot log10 <nil>, unsupported source type <nil>", 0},
		{int(1), "", float64(0)},
		{int8(1), "", float64(0)},
		{int16(1), "", float64(0)},
		{int32(1), "", float64(0)},
		{int64(1), "", float64(0)},
		{float32(1.1), "", float64(0.041392694571304324)},
		{float64(1.1), "", float64(0.04139268515822507)},
		{decimal.NewFromFloat(1.1), "", decimal.NewFromFloat(0.04139268515822507)},
	}

	for i := range len(testValues) {
		vars[""]["f1"] = testValues[i].valIn
		val, err = eCtx.Eval(exp)
		if testValues[i].errorOut == "" {
			assert.Nil(t, err)
			assert.Equal(t, testValues[i].valOut, val)
		} else {
			assert.Contains(t, testValues[i].errorOut, err.Error())
		}
	}

	// Round

	exp, _ = parser.ParseExpr(`round()`)
	val, err = eCtx.Eval(exp)
	assert.Contains(t, "cannot evaluate round(), requires 1 args, 0 supplied", err.Error())

	exp, _ = parser.ParseExpr(`round(f1)`)

	testValues = []testValue{
		{nil, "cannot round <nil>, unsupported source type <nil>", 0},
		{int(1), "", int64(1)},
		{int8(1), "", int64(1)},
		{int16(1), "", int64(1)},
		{int32(1), "", int64(1)},
		{int64(1), "", int64(1)},
		{float32(-1.5), "", float64(-2)},
		{float64(-1.5), "", float64(-2)},
		{decimal.NewFromFloat(-1.5), "", decimal.NewFromFloat(-2.0)},
	}

	for i := range len(testValues) {
		vars[""]["f1"] = testValues[i].valIn
		val, err = eCtx.Eval(exp)
		if testValues[i].errorOut == "" {
			assert.Nil(t, err)
			assert.Equal(t, testValues[i].valOut, val)
		} else {
			assert.Contains(t, testValues[i].errorOut, err.Error())
		}
	}
}
