package sc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertIdxComp(t *testing.T, fName string, fType TableFieldType, caseSens IdxCaseSensitivity, sortOrder IdxSortOrder, strLen int64, compDef *IdxComponentDef) {
	assert.Equal(t, fName, compDef.FieldName)
	assert.Equal(t, fType, compDef.FieldType)
	assert.Equal(t, caseSens, compDef.CaseSensitivity)
	assert.Equal(t, sortOrder, compDef.SortOrder)
	assert.Equal(t, strLen, compDef.StringLen)
}

func TestIndexDefParser(t *testing.T) {
	fieldRefs := FieldRefs{
		FieldRef{"t1", "f_int", FieldTypeInt},
		FieldRef{"t1", "f_float", FieldTypeFloat},
		FieldRef{"t1", "f_bool", FieldTypeBool},
		FieldRef{"t1", "f_str", FieldTypeString},
		FieldRef{"t1", "f_time", FieldTypeDateTime},
		FieldRef{"t1", "f_dec", FieldTypeDecimal2},
	}
	rawIdxDefMap := map[string]string{
		"idx_all_default": "non_unique(f_int(),f_float(),f_bool(),f_str(),f_time(),f_dec())",
		"idx_all_desc":    "unique(f_int(desc),f_float(desc),f_bool(desc),f_str(desc,ignore_case,128),f_time(desc),f_dec(desc))",
		"idx_all_asc":     "unique(f_int(asc),f_float(asc),f_bool(asc),f_str(asc,case_sensitive,15),f_time(asc),f_dec(asc))",
		"idx_no_mods":     "unique(f_int,f_float,f_bool,f_str,f_time,f_dec)",
	}
	idxDefMap := IdxDefMap{}
	idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)

	extractedFieldRefs := idxDefMap["idx_all_default"].getComponentFieldRefs("t2")
	for i := 0; i < len(extractedFieldRefs); i++ {
		extractedFieldRef := &extractedFieldRefs[i]
		assert.Equal(t, "t2", extractedFieldRef.TableName)
		foundFieldRef, _ := fieldRefs.FindByFieldName(extractedFieldRef.FieldName)
		assert.Equal(t, extractedFieldRef.FieldType, foundFieldRef.FieldType)
		assert.Equal(t, "t1", foundFieldRef.TableName)
	}

	assert.Equal(t, IdxNonUnique, idxDefMap["idx_all_default"].Uniqueness)
	assertIdxComp(t, "f_int", FieldTypeInt, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_default"].Components[0])
	assertIdxComp(t, "f_float", FieldTypeFloat, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_default"].Components[1])
	assertIdxComp(t, "f_bool", FieldTypeBool, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_default"].Components[2])
	assertIdxComp(t, "f_str", FieldTypeString, IdxCaseSensitive, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_default"].Components[3])
	assertIdxComp(t, "f_time", FieldTypeDateTime, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_default"].Components[4])
	assertIdxComp(t, "f_dec", FieldTypeDecimal2, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_default"].Components[5])

	assert.Equal(t, IdxUnique, idxDefMap["idx_all_desc"].Uniqueness)
	assertIdxComp(t, "f_int", FieldTypeInt, IdxCaseSensitivityUnknown, IdxSortDesc, DefaultStringComponentLen, &idxDefMap["idx_all_desc"].Components[0])
	assertIdxComp(t, "f_float", FieldTypeFloat, IdxCaseSensitivityUnknown, IdxSortDesc, DefaultStringComponentLen, &idxDefMap["idx_all_desc"].Components[1])
	assertIdxComp(t, "f_bool", FieldTypeBool, IdxCaseSensitivityUnknown, IdxSortDesc, DefaultStringComponentLen, &idxDefMap["idx_all_desc"].Components[2])
	assertIdxComp(t, "f_str", FieldTypeString, IdxIgnoreCase, IdxSortDesc, 128, &idxDefMap["idx_all_desc"].Components[3])
	assertIdxComp(t, "f_time", FieldTypeDateTime, IdxCaseSensitivityUnknown, IdxSortDesc, DefaultStringComponentLen, &idxDefMap["idx_all_desc"].Components[4])
	assertIdxComp(t, "f_dec", FieldTypeDecimal2, IdxCaseSensitivityUnknown, IdxSortDesc, DefaultStringComponentLen, &idxDefMap["idx_all_desc"].Components[5])

	assert.Equal(t, IdxUnique, idxDefMap["idx_all_asc"].Uniqueness)
	assertIdxComp(t, "f_int", FieldTypeInt, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_asc"].Components[0])
	assertIdxComp(t, "f_float", FieldTypeFloat, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_asc"].Components[1])
	assertIdxComp(t, "f_bool", FieldTypeBool, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_asc"].Components[2])
	assertIdxComp(t, "f_str", FieldTypeString, IdxCaseSensitive, IdxSortAsc, MinStringComponentLen, &idxDefMap["idx_all_asc"].Components[3])
	assertIdxComp(t, "f_time", FieldTypeDateTime, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_asc"].Components[4])
	assertIdxComp(t, "f_dec", FieldTypeDecimal2, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_all_asc"].Components[5])

	assert.Equal(t, IdxUnique, idxDefMap["idx_no_mods"].Uniqueness)
	assertIdxComp(t, "f_int", FieldTypeInt, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_no_mods"].Components[0])
	assertIdxComp(t, "f_float", FieldTypeFloat, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_no_mods"].Components[1])
	assertIdxComp(t, "f_bool", FieldTypeBool, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_no_mods"].Components[2])
	assertIdxComp(t, "f_str", FieldTypeString, IdxCaseSensitive, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_no_mods"].Components[3])
	assertIdxComp(t, "f_time", FieldTypeDateTime, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_no_mods"].Components[4])
	assertIdxComp(t, "f_dec", FieldTypeDecimal2, IdxCaseSensitivityUnknown, IdxSortAsc, DefaultStringComponentLen, &idxDefMap["idx_no_mods"].Components[5])

}

func TestIndexDefParserBad(t *testing.T) {
	fieldRefs := FieldRefs{
		FieldRef{"t1", "f_int", FieldTypeInt},
		FieldRef{"t1", "f_float", FieldTypeFloat},
		FieldRef{"t1", "f_bool", FieldTypeBool},
		FieldRef{"t1", "f_str", FieldTypeString},
		FieldRef{"t1", "f_time", FieldTypeDateTime},
		FieldRef{"t1", "f_dec", FieldTypeDecimal2},
	}
	rawIdxDefMap := map[string]string{"idx_bad_unique": "somename(f_int,f_float,f_bool,f_str,f_time,f_dec)"}
	idxDefMap := IdxDefMap{}
	err := idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)
	assert.Equal(t, "cannot parse index def [somename(f_int,f_float,f_bool,f_str,f_time,f_dec)]: expected top level unique()) or non_unique() definition, found somename", err.Error())

	rawIdxDefMap = map[string]string{"idx_bad_field": "unique(somefield1,somefield2(),f_bool,f_str,f_time,f_dec)"}
	idxDefMap = IdxDefMap{}
	err = idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)
	assert.Equal(t, "cannot parse order definitions: [index unique(somefield1,somefield2(),f_bool,f_str,f_time,f_dec): [cannot parse order component ident expression, field somefield1 unknown]; index unique(somefield1,somefield2(),f_bool,f_str,f_time,f_dec): [cannot parse order component func expression, field somefield2 unknown]]", err.Error())

	rawIdxDefMap = map[string]string{"idx_bad_modifier": "unique(f_int(somemodifier),f_float(case_sensitive),f_bool,f_str,f_time,f_dec)"}
	idxDefMap = IdxDefMap{}
	err = idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)
	assert.Equal(t, "cannot parse order definitions: [index unique(f_int(somemodifier),f_float(case_sensitive),f_bool,f_str,f_time,f_dec): [unknown modifier somemodifier for field f_int, expected ignore_case,case_sensitive,asc,desc]; index unique(f_int(somemodifier),f_float(case_sensitive),f_bool,f_str,f_time,f_dec): [index component for field f_float of type float cannot have case sensitivity modifier case_sensitive, remove it from index component definition]]", err.Error())

	rawIdxDefMap = map[string]string{"idx_bad_comp_string_len": "unique(f_int,f_float,f_bool(128),f_str(32000),f_time,f_dec)"}
	idxDefMap = IdxDefMap{}
	err = idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)
	assert.Equal(t, "cannot parse order definitions: [index unique(f_int,f_float,f_bool(128),f_str(32000),f_time,f_dec): [invalid expression &{29 INT 128} in f_bool, component length modifier is valid only for string fields, but f_bool has type bool]; index unique(f_int,f_float,f_bool(128),f_str(32000),f_time,f_dec): [invalid expression &{40 INT 32000} in f_str, component length modifier for string fields cannot exceed 1024]]", err.Error())

	rawIdxDefMap = map[string]string{"idx_bad_comp_string_len_float": "unique(f_int,f_float,f_bool,f_str(5.2),f_time,f_dec)"}
	idxDefMap = IdxDefMap{}
	err = idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)
	assert.Equal(t, "cannot parse order definitions: [index unique(f_int,f_float,f_bool,f_str(5.2),f_time,f_dec): [invalid expression &{35 FLOAT 5.2} in f_str, expected an integer for string component length]]", err.Error())

	rawIdxDefMap = map[string]string{"idx_bad_modifier_func": "unique(f_int,f_float,f_bool,f_str(badmofifierfunc()),f_time,f_dec)"}
	idxDefMap = IdxDefMap{}
	err = idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)
	assert.Equal(t, "cannot parse order definitions: [index unique(f_int,f_float,f_bool,f_str(badmofifierfunc()),f_time,f_dec): [invalid expression &{badmofifierfunc 50 [] 0 51}, expected a modifier for field f_str: expected ignore_case,case_sensitive,asc,desc or an integer]]", err.Error())

	rawIdxDefMap = map[string]string{"idx_bad_field_expr": "unique(123)"}
	idxDefMap = IdxDefMap{}
	err = idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)
	assert.Equal(t, "cannot parse order definitions: [index unique(123): [invalid expression in index component definition, expected 'field([modifiers])' or 'field' where 'field' is one of the fields of the table created by this node]]", err.Error())

	rawIdxDefMap = map[string]string{"idx_bad_syntax": "unique("}
	idxDefMap = IdxDefMap{}
	err = idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)
	assert.Equal(t, "cannot parse order def 'unique(': 1:8: expected ')', found 'EOF'", err.Error())

	rawIdxDefMap = map[string]string{"idx_bad_no_call": "unique"}
	idxDefMap = IdxDefMap{}
	err = idxDefMap.parseRawIndexDefMap(rawIdxDefMap, &fieldRefs)
	assert.Equal(t, "cannot parse index def [unique]: expected top level unique()) or non_unique() definition, found unknown expression", err.Error())
}
