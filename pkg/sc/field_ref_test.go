package sc

import (
	"go/parser"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendWithFilter(t *testing.T) {
	targetRefs := FieldRefs{FieldRef{TableName: "t0", FieldName: "f0"}}
	sourceRefs := FieldRefs{FieldRef{TableName: "t1", FieldName: "f1"}, FieldRef{TableName: "t2", FieldName: "f2"}}
	targetRefs.AppendWithFilter(sourceRefs, "t2")
	assert.Equal(t, 2, len(targetRefs))
	assert.True(t, targetRefs[0].FieldName == "f0" && targetRefs[1].FieldName == "f2" || targetRefs[0].FieldName == "f2" && targetRefs[1].FieldName == "f0")

	assert.True(t, targetRefs.HasFieldsWithTableAlias("t0"))
	assert.False(t, targetRefs.HasFieldsWithTableAlias("t1"))
	assert.True(t, targetRefs.HasFieldsWithTableAlias("t2"))

	targetRefs.Append(sourceRefs)
	assert.Equal(t, 3, len(targetRefs))
}

func TestEvalFieldRefExpression(t *testing.T) {
	fieldRefs := FieldRefs{
		{
			FieldType: FieldTypeInt,
			TableName: "r",
			FieldName: "fieldInt"},
		{
			FieldType: FieldTypeFloat,
			TableName: "r",
			FieldName: "fieldFloat"},
		{
			FieldType: FieldTypeDecimal2,
			TableName: "r",
			FieldName: "fieldDec"},
		{
			FieldType: FieldTypeString,
			TableName: "r",
			FieldName: "fieldStr"}}

	exp, err := parser.ParseExpr(`r.fieldInt/r.fieldFloat`)
	assert.Nil(t, err)
	err = evalExpressionWithFieldRefsAndCheckType(exp, fieldRefs, FieldTypeFloat)
	assert.Nil(t, err)

	exp, err = parser.ParseExpr(`r.fieldFloat/r.fieldDec`)
	assert.Nil(t, err)
	err = evalExpressionWithFieldRefsAndCheckType(exp, fieldRefs, FieldTypeFloat)
	assert.Nil(t, err)

	exp, err = parser.ParseExpr(`r.fieldInt/r.fieldDec`)
	assert.Nil(t, err)
	err = evalExpressionWithFieldRefsAndCheckType(exp, fieldRefs, FieldTypeInt)
	assert.Contains(t, err.Error(), "expected type int, but got decimal")

	exp, err = parser.ParseExpr(`int(r.fieldStr)/float(r.fieldStr)`)
	assert.Nil(t, err)
	err = evalExpressionWithFieldRefsAndCheckType(exp, fieldRefs, FieldTypeFloat)
	assert.Nil(t, err)
}
