package storage

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval_capi"
)

type GuessedField struct {
	OriginalHeader string
	CapiName       string
	Type           eval_capi.TableFieldType
	Format         string
}

func guessCsvType(strVal string) (eval_capi.TableFieldType, string) {
	reDecimal2 := regexp.MustCompile(`^(\+|-|)[0-9]*\.[0-9][0-9]$`)              // Exactly two digits after decimal point
	reInt := regexp.MustCompile(`^(\+|-|)[0-9]+$`)                               // Just digits
	reFloat := regexp.MustCompile(`^(\+|-|)[0-9]*\.[0-9]+$`)                     // No scientific notation support
	reBool := regexp.MustCompile(`^(true|false|t|f|T|F|True|False|TRUE|FALSE)$`) // Whatever strconv.ParseBool supports
	reDt := []*regexp.Regexp{
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9][0-9][0-9][0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9](\+|-)[0-9][0-9][0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9][0-9](\+|-)[0-9][0-9][0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9][0-9][0-9][0-9][0-9](\+|-)[0-9][0-9][0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9] [0-9][0-9]:[0-9][0-9]:[0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9] [0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9] [0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9][0-9][0-9][0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9] [0-9][0-9]:[0-9][0-9]:[0-9][0-9](\+|-)[0-9][0-9][0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9] [0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9][0-9](\+|-)[0-9][0-9][0-9][0-9]$`),
		regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9] [0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9][0-9][0-9][0-9][0-9](\+|-)[0-9][0-9][0-9][0-9]$`)}
	fmtDt := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05.0000000",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.0000000-0700",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05.0000000",
		"2006-01-02 15:04:05-0700",
		"2006-01-02 15:04:05.000-0700",
		"2006-01-02 15:04:05.0000000-0700"}

	if reDecimal2.MatchString(strings.TrimSpace(strVal)) {
		return eval_capi.FieldTypeDecimal2, "%f" // Capillaries uses %f format specifier for decimal2 fields (because it uses sscanf(float) internally)
	}
	if reInt.MatchString(strings.TrimSpace(strVal)) {
		return eval_capi.FieldTypeInt, "%d"
	}
	if reFloat.MatchString(strings.TrimSpace(strVal)) {
		return eval_capi.FieldTypeFloat, "%f"
	}
	if reBool.MatchString(strings.ToLower(strings.TrimSpace(strVal))) {
		return eval_capi.FieldTypeBool, "" // Capillaries does not accept format specifier for bool fields
	}

	for i, re := range reDt {
		if re.MatchString(strings.TrimSpace(strVal)) {
			return eval_capi.FieldTypeDateTime, fmtDt[i]
		}
	}

	if len(strVal) == 0 {
		return eval_capi.FieldTypeString, ""
	}
	return eval_capi.FieldTypeString, "" // Capillaries does not accept format specifier for string fields
}

func CsvGuessFields(filePath string, csvHeaderLineIdx int, csvFirstDataLineIdx int, csvSeparator string) ([]*GuessedField, error) {
	var guessedFields []*GuessedField
	f, err := os.Open(filePath)
	if err != nil {
		return guessedFields, fmt.Errorf("cannot open csv file %s: %s", filePath, err.Error())
	}
	if f == nil {
		return guessedFields, fmt.Errorf("cannot open csv file %s: unknown error", filePath)
	}
	defer f.Close()

	fileReader := bufio.NewReader(f)
	r := csv.NewReader(fileReader)
	r.Comma = rune(csvSeparator[0])
	// To avoid bare \" error: https://stackoverflow.com/questions/31326659/golang-csv-error-bare-in-non-quoted-field
	r.LazyQuotes = true

	lineIdx := 0
	reNonAlphanum := regexp.MustCompile("[^a-zA-Z0-9_]")

	// If we find something more generic than on previous steps, make data type more generic (eventually, string)
	generalizeMap := map[eval_capi.TableFieldType]map[eval_capi.TableFieldType]eval_capi.TableFieldType{
		eval_capi.FieldTypeUnknown: {
			eval_capi.FieldTypeUnknown:  eval_capi.FieldTypeUnknown,
			eval_capi.FieldTypeString:   eval_capi.FieldTypeString,
			eval_capi.FieldTypeBool:     eval_capi.FieldTypeBool,
			eval_capi.FieldTypeInt:      eval_capi.FieldTypeInt,
			eval_capi.FieldTypeFloat:    eval_capi.FieldTypeFloat,
			eval_capi.FieldTypeDecimal2: eval_capi.FieldTypeDecimal2,
			eval_capi.FieldTypeDateTime: eval_capi.FieldTypeDateTime},
		eval_capi.FieldTypeBool: {
			eval_capi.FieldTypeUnknown:  eval_capi.FieldTypeBool,
			eval_capi.FieldTypeString:   eval_capi.FieldTypeString,
			eval_capi.FieldTypeBool:     eval_capi.FieldTypeBool,
			eval_capi.FieldTypeInt:      eval_capi.FieldTypeString,
			eval_capi.FieldTypeFloat:    eval_capi.FieldTypeString,
			eval_capi.FieldTypeDecimal2: eval_capi.FieldTypeString,
			eval_capi.FieldTypeDateTime: eval_capi.FieldTypeString},
		eval_capi.FieldTypeString: {
			eval_capi.FieldTypeUnknown:  eval_capi.FieldTypeString,
			eval_capi.FieldTypeString:   eval_capi.FieldTypeString,
			eval_capi.FieldTypeBool:     eval_capi.FieldTypeString,
			eval_capi.FieldTypeInt:      eval_capi.FieldTypeString,
			eval_capi.FieldTypeFloat:    eval_capi.FieldTypeString,
			eval_capi.FieldTypeDecimal2: eval_capi.FieldTypeString,
			eval_capi.FieldTypeDateTime: eval_capi.FieldTypeString},
		eval_capi.FieldTypeInt: {
			eval_capi.FieldTypeUnknown:  eval_capi.FieldTypeInt,
			eval_capi.FieldTypeString:   eval_capi.FieldTypeString,
			eval_capi.FieldTypeBool:     eval_capi.FieldTypeString,
			eval_capi.FieldTypeInt:      eval_capi.FieldTypeInt,
			eval_capi.FieldTypeFloat:    eval_capi.FieldTypeFloat,
			eval_capi.FieldTypeDecimal2: eval_capi.FieldTypeDecimal2,
			eval_capi.FieldTypeDateTime: eval_capi.FieldTypeString},
		eval_capi.FieldTypeFloat: {
			eval_capi.FieldTypeUnknown:  eval_capi.FieldTypeFloat,
			eval_capi.FieldTypeString:   eval_capi.FieldTypeString,
			eval_capi.FieldTypeBool:     eval_capi.FieldTypeString,
			eval_capi.FieldTypeInt:      eval_capi.FieldTypeFloat,
			eval_capi.FieldTypeFloat:    eval_capi.FieldTypeFloat,
			eval_capi.FieldTypeDecimal2: eval_capi.FieldTypeFloat,
			eval_capi.FieldTypeDateTime: eval_capi.FieldTypeString},
		eval_capi.FieldTypeDecimal2: {
			eval_capi.FieldTypeUnknown:  eval_capi.FieldTypeDecimal2,
			eval_capi.FieldTypeString:   eval_capi.FieldTypeString,
			eval_capi.FieldTypeBool:     eval_capi.FieldTypeString,
			eval_capi.FieldTypeInt:      eval_capi.FieldTypeDecimal2,
			eval_capi.FieldTypeFloat:    eval_capi.FieldTypeFloat,
			eval_capi.FieldTypeDecimal2: eval_capi.FieldTypeDecimal2,
			eval_capi.FieldTypeDateTime: eval_capi.FieldTypeString},
		eval_capi.FieldTypeDateTime: {
			eval_capi.FieldTypeUnknown:  eval_capi.FieldTypeDateTime,
			eval_capi.FieldTypeString:   eval_capi.FieldTypeString,
			eval_capi.FieldTypeBool:     eval_capi.FieldTypeString,
			eval_capi.FieldTypeInt:      eval_capi.FieldTypeString,
			eval_capi.FieldTypeFloat:    eval_capi.FieldTypeString,
			eval_capi.FieldTypeDecimal2: eval_capi.FieldTypeString,
			eval_capi.FieldTypeDateTime: eval_capi.FieldTypeDateTime},
	}

	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return guessedFields, fmt.Errorf("cannot read file [%s]: [%s]", filePath, err.Error())
		}

		if csvHeaderLineIdx >= 0 && csvHeaderLineIdx == lineIdx {
			// It's a header line
			guessedFields = make([]*GuessedField, len(line))
			for i, hdr := range line {
				guessedFields[i] = &GuessedField{
					OriginalHeader: hdr,
					CapiName:       "col_" + reNonAlphanum.ReplaceAllString(hdr, "_"),
					Type:           eval_capi.FieldTypeUnknown,
					Format:         ""}
			}
		} else if lineIdx < csvFirstDataLineIdx {
			// Still some header stuff
			continue
		} else {
			// It's a data line

			if guessedFields == nil {
				// No headers provided, time to improvise
				guessedFields = make([]*GuessedField, len(line))
				for i := range line {
					guessedFields[i] = &GuessedField{
						OriginalHeader: "",
						CapiName:       "col_" + fmt.Sprintf("%03d", i),
						Type:           eval_capi.FieldTypeUnknown,
						Format:         ""}
				}
			}

			for i, strVal := range line {
				guessedType, guessedFmt := guessCsvType(strVal)
				overrideType := generalizeMap[guessedFields[i].Type][guessedType]
				guessedFields[i].Type = overrideType
				guessedFields[i].Format = guessedFmt
			}
		}

		lineIdx++

		if lineIdx >= 100 {
			break
		}
	}

	for i, gf := range guessedFields {
		if gf.Type == eval_capi.FieldTypeUnknown {
			return guessedFields, fmt.Errorf("cannot detect type of column %d(%s)", i, gf.OriginalHeader)
		}
	}

	return guessedFields, nil
}
