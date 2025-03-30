package sc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/shopspring/decimal"
	"golang.org/x/text/runes"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const BeginningOfTimeMicro = int64(-62135596800000000) // time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC).UnixMicro()

func getNumericValueSign(v any, expectedType TableFieldType) (string, any, error) {
	var sign string
	var newVal any
	var ok bool

	switch expectedType {
	case FieldTypeInt:
		var n int64
		if n, ok = v.(int64); !ok {
			return "", nil, fmt.Errorf("cannot convert value %v to type %v", v, expectedType)
		}
		if n >= 0 {
			sign = "0" // "0" > "-"
			newVal = n
		} else {
			sign = "-"
			newVal = -n
		}

	case FieldTypeFloat:
		var f float64
		if f, ok = v.(float64); !ok {
			return "", nil, fmt.Errorf("cannot convert value %v to type %v", v, expectedType)
		}
		if f >= 0 {
			sign = "0" // "0" > "-"
			newVal = f
		} else {
			sign = "-"
			newVal = -f
		}

	case FieldTypeDecimal2:
		var d decimal.Decimal
		if d, ok = v.(decimal.Decimal); !ok {
			return "", nil, fmt.Errorf("cannot convert value %v to type %v", v, expectedType)
		}
		if d.Sign() >= 0 {
			sign = "0" // "0" > "-"
			newVal = d
		} else {
			sign = "-"
			newVal = d.Neg()
		}

	default:
		return "", nil, fmt.Errorf("cannot convert value %v to type %v, type not supported", v, expectedType)
	}
	return sign, newVal, nil
}

func BuildKey(fieldMap map[string]any, idxDef *IdxDef) (string, error) {
	var keyBuffer bytes.Buffer
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	flipReplacer := strings.NewReplacer("0", "9", "1", "8", "2", "7", "3", "6", "4", "5", "5", "4", "6", "3", "7", "2", "8", "1", "9", "0")

	for _, comp := range idxDef.Components {
		if _, ok := fieldMap[comp.FieldName]; !ok {
			return "", fmt.Errorf("cannot find value for field %v in %v while building key for index %v", comp.FieldName, fieldMap, idxDef)
		}

		var stringValue string

		switch comp.FieldType {
		case FieldTypeInt:
			sign, absVal, err := getNumericValueSign(fieldMap[comp.FieldName], FieldTypeInt)
			if err != nil {
				return "", err
			}
			stringValue = fmt.Sprintf("%s%018d", sign, absVal)
			// If this is a negative value, flip every digit
			if sign == "-" {
				stringValue = flipReplacer.Replace(stringValue)
			}

		case FieldTypeFloat:
			// We should support numbers as big as 10^32 and with 32 digits afetr decimal point
			sign, absVal, err := getNumericValueSign(fieldMap[comp.FieldName], FieldTypeFloat)
			if err != nil {
				return "", err
			}
			stringValue = strings.ReplaceAll(fmt.Sprintf("%s%66s", sign, fmt.Sprintf("%.32f", absVal)), " ", "0")
			// If this is a negative value, flip every digit
			if sign == "-" {
				stringValue = flipReplacer.Replace(stringValue)
			}

		case FieldTypeDecimal2:
			sign, absVal, err := getNumericValueSign(fieldMap[comp.FieldName], FieldTypeDecimal2)
			if err != nil {
				return "", err
			}
			decVal, ok := absVal.(decimal.Decimal)
			if !ok {
				return "", fmt.Errorf("unexpectedly cannot convert value %v to type decimal2", fieldMap[comp.FieldName])
			}
			floatVal, _ := decVal.Float64()
			stringValue = strings.ReplaceAll(fmt.Sprintf("%s%66s", sign, fmt.Sprintf("%.32f", floatVal)), " ", "0")
			// If this is a negative value, flip every digit
			if sign == "-" {
				stringValue = flipReplacer.Replace(stringValue)
			}

		case FieldTypeDateTime:
			// We support time differences up to microsecond. Not nanosecond! Cassandra supports only milliseconds. Millis are our lingua franca.
			t, ok := fieldMap[comp.FieldName].(time.Time)
			if !ok {
				return "", fmt.Errorf("cannot convert value %v to type datetime", fieldMap[comp.FieldName])
			}
			stringValue = fmt.Sprintf("%020d", t.UnixMicro()-BeginningOfTimeMicro)

		case FieldTypeString:
			s, ok := fieldMap[comp.FieldName].(string)
			if !ok {
				return "", fmt.Errorf("cannot convert value %v to type string", fieldMap[comp.FieldName])
			}
			// Normalize the string
			transformedString, _, _ := transform.String(t, s)
			// Take only first 64 (or whatever we have in StringLen) characters
			// use "%-64s" sprint format to pad with spaces on the right
			formatString := fmt.Sprintf("%s-%ds", "%", comp.StringLen)
			stringValue = fmt.Sprintf(formatString, transformedString)[:comp.StringLen]
			if comp.CaseSensitivity == IdxIgnoreCase {
				stringValue = strings.ToUpper(stringValue)
			}

		case FieldTypeBool:
			b, ok := fieldMap[comp.FieldName].(bool)
			if !ok {
				return "", fmt.Errorf("cannot convert value %v to type bool", fieldMap[comp.FieldName])
			}
			if b {
				stringValue = "T" // "F" < "T"
			} else {
				stringValue = "F"
			}

		default:
			return "", fmt.Errorf("cannot build key, unsupported field data type %s", comp.FieldType)
		}

		// Used by file creator top. Not used by actual indexes - Cassandra cannot do proper ORDER BY anyways
		if comp.SortOrder == IdxSortDesc {
			stringBytes := []byte(stringValue)
			for i, b := range stringBytes {
				stringBytes[i] = 0xFF - b
			}
			stringValue = hex.EncodeToString(stringBytes)
		}

		keyBuffer.WriteString(stringValue)
	}

	return keyBuffer.String(), nil
}
