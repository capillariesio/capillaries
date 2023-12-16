package storage

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/capillariesio/capillaries/pkg/sc"
)

type GuessedField struct {
	OriginalHeader string
	CapiName       string
	Type           sc.TableFieldType
	Format         string
}

func guessCsvType(strVal string) (sc.TableFieldType, string) {
	reDecimal2 := regexp.MustCompile(`^(\+|-|)[0-9]*\.[0-9](|[0-9])$`)
	reInt := regexp.MustCompile(`^(\+|-|)[0-9]+$`)
	reFloat := regexp.MustCompile(`^(\+|-|)[0-9]*\.[0-9]+$`)
	reBool := regexp.MustCompile(`^(true|false|t|f)$`)
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
		return sc.FieldTypeDecimal2, "%s"
	}
	if reInt.MatchString(strings.TrimSpace(strVal)) {
		return sc.FieldTypeInt, "%d"
	}
	if reFloat.MatchString(strings.TrimSpace(strVal)) {
		return sc.FieldTypeFloat, "%f"
	}
	if reBool.MatchString(strings.ToLower(strings.TrimSpace(strVal))) {
		return sc.FieldTypeBool, "%t"
	}

	for i, re := range reDt {
		if re.MatchString(strings.TrimSpace(strVal)) {
			return sc.FieldTypeDateTime, fmtDt[i]
		}
	}

	if len(strVal) == 0 {
		return sc.FieldTypeUnknown, ""
	}
	return sc.FieldTypeString, "%s"
}

func CsvGuessFields(filePath string, csvHeaderLineIdx int, csvFirstDataLineIdx int, csvSeparator string) ([]*GuessedField, error) {
	var guessedFields []*GuessedField
	f, err := os.Open(filePath)
	if err != nil {
		return guessedFields, fmt.Errorf("cannot open file %s: %s", filePath, err.Error())
	}
	defer f.Close()

	fileReader := bufio.NewReader(f)
	r := csv.NewReader(fileReader)
	r.Comma = rune(csvSeparator[0])
	// To avoid bare \" error: https://stackoverflow.com/questions/31326659/golang-csv-error-bare-in-non-quoted-field
	r.LazyQuotes = true

	lineIdx := 0
	reNonAlphanum := regexp.MustCompile("[^a-zA-Z0-9_]")

	generalizeMap := map[sc.TableFieldType]map[sc.TableFieldType]sc.TableFieldType{
		sc.FieldTypeUnknown: {
			sc.FieldTypeUnknown:  sc.FieldTypeUnknown,
			sc.FieldTypeString:   sc.FieldTypeString,
			sc.FieldTypeBool:     sc.FieldTypeBool,
			sc.FieldTypeInt:      sc.FieldTypeInt,
			sc.FieldTypeFloat:    sc.FieldTypeFloat,
			sc.FieldTypeDecimal2: sc.FieldTypeDecimal2,
			sc.FieldTypeDateTime: sc.FieldTypeDateTime},
		sc.FieldTypeBool: {
			sc.FieldTypeUnknown:  sc.FieldTypeBool,
			sc.FieldTypeString:   sc.FieldTypeString,
			sc.FieldTypeBool:     sc.FieldTypeBool,
			sc.FieldTypeInt:      sc.FieldTypeString,
			sc.FieldTypeFloat:    sc.FieldTypeString,
			sc.FieldTypeDecimal2: sc.FieldTypeString,
			sc.FieldTypeDateTime: sc.FieldTypeString},
		sc.FieldTypeString: {
			sc.FieldTypeUnknown:  sc.FieldTypeString,
			sc.FieldTypeString:   sc.FieldTypeString,
			sc.FieldTypeBool:     sc.FieldTypeString,
			sc.FieldTypeInt:      sc.FieldTypeString,
			sc.FieldTypeFloat:    sc.FieldTypeString,
			sc.FieldTypeDecimal2: sc.FieldTypeString,
			sc.FieldTypeDateTime: sc.FieldTypeString},
		sc.FieldTypeInt: {
			sc.FieldTypeUnknown:  sc.FieldTypeInt,
			sc.FieldTypeString:   sc.FieldTypeString,
			sc.FieldTypeBool:     sc.FieldTypeString,
			sc.FieldTypeInt:      sc.FieldTypeInt,
			sc.FieldTypeFloat:    sc.FieldTypeFloat,
			sc.FieldTypeDecimal2: sc.FieldTypeDecimal2,
			sc.FieldTypeDateTime: sc.FieldTypeString},
		sc.FieldTypeFloat: {
			sc.FieldTypeUnknown:  sc.FieldTypeFloat,
			sc.FieldTypeString:   sc.FieldTypeString,
			sc.FieldTypeBool:     sc.FieldTypeString,
			sc.FieldTypeInt:      sc.FieldTypeFloat,
			sc.FieldTypeFloat:    sc.FieldTypeFloat,
			sc.FieldTypeDecimal2: sc.FieldTypeFloat,
			sc.FieldTypeDateTime: sc.FieldTypeString},
		sc.FieldTypeDecimal2: {
			sc.FieldTypeUnknown:  sc.FieldTypeDecimal2,
			sc.FieldTypeString:   sc.FieldTypeString,
			sc.FieldTypeBool:     sc.FieldTypeString,
			sc.FieldTypeInt:      sc.FieldTypeDecimal2,
			sc.FieldTypeFloat:    sc.FieldTypeFloat,
			sc.FieldTypeDecimal2: sc.FieldTypeDecimal2,
			sc.FieldTypeDateTime: sc.FieldTypeString},
		sc.FieldTypeDateTime: {
			sc.FieldTypeUnknown:  sc.FieldTypeDateTime,
			sc.FieldTypeString:   sc.FieldTypeString,
			sc.FieldTypeBool:     sc.FieldTypeString,
			sc.FieldTypeInt:      sc.FieldTypeString,
			sc.FieldTypeFloat:    sc.FieldTypeString,
			sc.FieldTypeDecimal2: sc.FieldTypeString,
			sc.FieldTypeDateTime: sc.FieldTypeDateTime},
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
					Type:           sc.FieldTypeUnknown,
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
						Type:           sc.FieldTypeUnknown,
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
		if gf.Type == sc.FieldTypeUnknown {
			return guessedFields, fmt.Errorf("cannot detect type of column %d(%s)", i, gf.OriginalHeader)
		}
	}

	return guessedFields, nil
}
