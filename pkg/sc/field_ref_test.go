package sc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendWithFilter(t *testing.T) {
	targetRefs := FieldRefs{FieldRef{TableName: "t0", FieldName: "f0"}}
	sourceRefs := FieldRefs{FieldRef{TableName: "t1", FieldName: "f1"}, FieldRef{TableName: "t2", FieldName: "f2"}}
	targetRefs.AppendWithFilter(sourceRefs, "t2")
	assert.Equal(t, 2, len(targetRefs))
	assert.True(t, "f0" == targetRefs[0].FieldName && "f2" == targetRefs[1].FieldName || "f2" == targetRefs[0].FieldName && "f0" == targetRefs[1].FieldName)

	assert.True(t, targetRefs.HasFieldsWithTableAlias("t0"))
	assert.False(t, targetRefs.HasFieldsWithTableAlias("t1"))
	assert.True(t, targetRefs.HasFieldsWithTableAlias("t2"))

	targetRefs.Append(sourceRefs)
	assert.Equal(t, 3, len(targetRefs))
}
