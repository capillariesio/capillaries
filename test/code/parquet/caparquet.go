package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/storage"
	gp "github.com/fraugster/parquet-go"
	"github.com/fraugster/parquet-go/parquet"
	//"github.com/pkg/profile"
)

const (
	CmdDiff string = "diff"
	CmdCat  string = "cat"
)

func usage(flagset *flag.FlagSet) {
	fmt.Printf("Capillaries parquet tool\nUsage: caparquet <command> <command parameters>\nCommands:\n")
	fmt.Printf("  %s %s %s\n",
		CmdDiff, "<left file>", "<right file>")
	if flagset != nil {
		fmt.Printf("\n%s parameters:\n", flagset.Name())
		flagset.PrintDefaults()
	}
	os.Exit(0)
}

func diff(leftPath string, rightPath string, isIdenticalSchemas bool) error {
	fLeft, err := os.Open(leftPath)
	if err != nil {
		return err
	}
	defer fLeft.Close()

	readerLeft, err := gp.NewFileReader(fLeft)
	if err != nil {
		return err
	}
	schemaLeft := readerLeft.GetSchemaDefinition()

	fRight, err := os.Open(rightPath)
	if err != nil {
		return err
	}
	defer fRight.Close()

	readerRight, err := gp.NewFileReader(fRight)
	if err != nil {
		return err
	}
	schemaRight := readerRight.GetSchemaDefinition()

	if isIdenticalSchemas && schemaLeft.String() != schemaRight.String() {
		return fmt.Errorf("schemas do not match:\n\n%s\n\n%s", schemaLeft.String(), schemaRight.String())
	}

	leftSchemaElementMap := map[string]*parquet.SchemaElement{}
	leftFields := make([]string, len(schemaLeft.RootColumn.Children))
	for i, column := range schemaLeft.RootColumn.Children {
		leftFields[i] = column.SchemaElement.Name
		leftSchemaElementMap[column.SchemaElement.Name] = column.SchemaElement
	}

	rightSchemaElementMap := map[string]*parquet.SchemaElement{}
	rightFields := make([]string, len(schemaRight.RootColumn.Children))
	for i, column := range schemaRight.RootColumn.Children {
		rightFields[i] = column.SchemaElement.Name
		rightSchemaElementMap[column.SchemaElement.Name] = column.SchemaElement
	}

	if strings.Join(leftFields, ",") != strings.Join(rightFields, ",") {
		return fmt.Errorf("column sets do not match:\n\n%s\n\n%s", strings.Join(leftFields, ","), strings.Join(rightFields, ","))
	}

	leftTypes := make([]sc.TableFieldType, len(leftFields))
	rightTypes := make([]sc.TableFieldType, len(rightFields))
	for i, fieldName := range leftFields {
		seLeft, _ := leftSchemaElementMap[fieldName]
		seRight, _ := rightSchemaElementMap[fieldName]
		var err error
		leftTypes[i], err = storage.ParquetGuessCapiType(seLeft)
		if err != nil {
			return fmt.Errorf("cannot read left field %s: %s", fieldName, err.Error())
		}
		rightTypes[i], err = storage.ParquetGuessCapiType(seRight)
		if err != nil {
			return fmt.Errorf("cannot read right field %s: %s", fieldName, err.Error())
		}
		if leftTypes[i] != rightTypes[i] {
			return fmt.Errorf("type mismatch, column %s:  %s vs %s", fieldName, leftTypes[i], rightTypes[i])
		}
	}

	rowIdx := 0
	for {
		dLeft, errLeft := readerLeft.NextRow()
		dRight, errRight := readerRight.NextRow()

		if errLeft == io.EOF && errRight == io.EOF {
			break
		} else if errLeft == io.EOF && errRight == nil {
			return fmt.Errorf("left file is shorter, has only %d rows", rowIdx)
		} else if errLeft == nil && errRight == io.EOF {
			return fmt.Errorf("right file is shorter, has only %d rows", rowIdx)
		} else if errLeft != nil {
			return fmt.Errorf("cannot get left row %d: %s", rowIdx, errLeft.Error())
		} else if errRight != nil {
			return fmt.Errorf("cannot get right row %d: %s", rowIdx, errRight.Error())
		}

		if isIdenticalSchemas {
			if !reflect.DeepEqual(dLeft, dRight) {
				return fmt.Errorf("mismatch:\n%v\n%v", dLeft, dRight)
			}
		} else {
			for colIdx, fieldName := range leftFields {
				seLeft, _ := leftSchemaElementMap[fieldName]
				seRight, _ := rightSchemaElementMap[fieldName]
				leftVolatile, leftPresent := dLeft[fieldName]
				rightVolatile, rightPresent := dRight[fieldName]
				if !leftPresent && !rightPresent {
					// Both nil, good
					continue
				} else if leftPresent && !rightPresent {
					return fmt.Errorf("mismatch row %d, column %s: left not nil, right nil", rowIdx, fieldName)
				} else if !leftPresent && rightPresent {
					return fmt.Errorf("mismatch row %d, column %s: left nil, right not nil", rowIdx, fieldName)
				}
				switch leftTypes[colIdx] {
				case sc.FieldTypeString:
					l, err := storage.ParquetReadString(leftVolatile, seLeft)
					if err != nil {
						return fmt.Errorf("cannot read string row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					r, err := storage.ParquetReadString(rightVolatile, seRight)
					if err != nil {
						return fmt.Errorf("cannot read string row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					if l != r {
						return fmt.Errorf("%07d-%s: %s vs %s", rowIdx, fieldName, l, r)
					}
				case sc.FieldTypeInt:
					l, err := storage.ParquetReadInt(leftVolatile, seLeft)
					if err != nil {
						return fmt.Errorf("cannot read int row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					r, err := storage.ParquetReadInt(rightVolatile, seRight)
					if err != nil {
						return fmt.Errorf("cannot read int row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					if l != r {
						return fmt.Errorf("%07d-%s: %d vs %d", rowIdx, fieldName, l, r)
					}
				case sc.FieldTypeFloat:
					l, err := storage.ParquetReadFloat(leftVolatile, seLeft)
					if err != nil {
						return fmt.Errorf("cannot read float row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					r, err := storage.ParquetReadFloat(rightVolatile, seRight)
					if err != nil {
						return fmt.Errorf("cannot read float row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					if l != r {
						return fmt.Errorf("%07d-%s: %f vs %f", rowIdx, fieldName, l, r)
					}
				case sc.FieldTypeBool:
					l, err := storage.ParquetReadBool(leftVolatile, seLeft)
					if err != nil {
						return fmt.Errorf("cannot read bool row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					r, err := storage.ParquetReadBool(rightVolatile, seRight)
					if err != nil {
						return fmt.Errorf("cannot read bool row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					if l != r {
						return fmt.Errorf("%07d-%s: %t vs %t", rowIdx, fieldName, l, r)
					}
				case sc.FieldTypeDateTime:
					l, err := storage.ParquetReadDateTime(leftVolatile, seLeft)
					if err != nil {
						return fmt.Errorf("cannot read DateTime row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					r, err := storage.ParquetReadDateTime(rightVolatile, seRight)
					if err != nil {
						return fmt.Errorf("cannot read DateTime row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					if !l.Equal(r) {
						return fmt.Errorf("%07d-%s: %s vs %s", rowIdx, fieldName, l.Format("2006-01-02T15:04:05.000000"), r.Format("2006-01-02T15:04:05.000000"))
					}
				case sc.FieldTypeDecimal2:
					l, err := storage.ParquetReadDecimal2(leftVolatile, seLeft)
					if err != nil {
						return fmt.Errorf("cannot read decimal2 row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					r, err := storage.ParquetReadDecimal2(rightVolatile, seRight)
					if err != nil {
						return fmt.Errorf("cannot read decimal2 row %d, column %s: %s", rowIdx, fieldName, err.Error())
					}
					if !l.Equal(r) {
						return fmt.Errorf("%07d-%s: %s vs %s", rowIdx, fieldName, l.String(), r.String())
					}
				}
			}
		}
		rowIdx++
	}
	return nil
}

func cat(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader, err := gp.NewFileReader(f)
	if err != nil {
		return err
	}
	schema := reader.GetSchemaDefinition()

	schemaElementMap := map[string]*parquet.SchemaElement{}
	fields := make([]string, len(schema.RootColumn.Children))
	for i, column := range schema.RootColumn.Children {
		fields[i] = column.SchemaElement.Name
		schemaElementMap[column.SchemaElement.Name] = column.SchemaElement
	}

	types := make([]sc.TableFieldType, len(fields))
	for i, fieldName := range fields {
		se, _ := schemaElementMap[fieldName]
		var err error
		types[i], err = storage.ParquetGuessCapiType(se)
		if err != nil {
			return fmt.Errorf("cannot read field %s: %s", fieldName, err.Error())
		}
	}

	fmt.Printf("%s\n", strings.Join(fields, ","))

	rowIdx := 0
	for {
		d, err := reader.NextRow()

		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("cannot get row %d: %s", rowIdx, err.Error())
		}

		var sb strings.Builder
		for colIdx, fieldName := range fields {
			if colIdx > 0 {
				sb.WriteString(",")
			}
			se, _ := schemaElementMap[fieldName]
			volatile, present := d[fieldName]
			if !present {
				// Both nil, good
				continue
			}
			switch types[colIdx] {
			case sc.FieldTypeString:
				typedVal, err := storage.ParquetReadString(volatile, se)
				if err != nil {
					return fmt.Errorf("cannot read string row %d, column %s: %s", rowIdx, fieldName, err.Error())
				}
				sb.WriteString(typedVal)

			case sc.FieldTypeInt:
				typedVal, err := storage.ParquetReadInt(volatile, se)
				if err != nil {
					return fmt.Errorf("cannot read int row %d, column %s: %s", rowIdx, fieldName, err.Error())
				}
				sb.WriteString(fmt.Sprintf("%d", typedVal))

			case sc.FieldTypeFloat:
				typedVal, err := storage.ParquetReadFloat(volatile, se)
				if err != nil {
					return fmt.Errorf("cannot read float row %d, column %s: %s", rowIdx, fieldName, err.Error())
				}
				sb.WriteString(fmt.Sprintf("%f", typedVal))

			case sc.FieldTypeBool:
				typedVal, err := storage.ParquetReadBool(volatile, se)
				if err != nil {
					return fmt.Errorf("cannot read bool row %d, column %s: %s", rowIdx, fieldName, err.Error())
				}
				sb.WriteString(fmt.Sprintf("%t", typedVal))

			case sc.FieldTypeDateTime:
				typedVal, err := storage.ParquetReadDateTime(volatile, se)
				if err != nil {
					return fmt.Errorf("cannot read DateTime row %d, column %s: %s", rowIdx, fieldName, err.Error())
				}
				sb.WriteString(typedVal.Format("2006-01-02T15:04:05.000000Z"))

			case sc.FieldTypeDecimal2:
				typedVal, err := storage.ParquetReadDecimal2(volatile, se)
				if err != nil {
					return fmt.Errorf("cannot read decimal2 row %d, column %s: %s", rowIdx, fieldName, err.Error())
				}
				if *se.Precision > 0 {
					sb.WriteString(typedVal.StringFixed(*se.Precision))
				} else {
					sb.WriteString(typedVal.String())
				}
			default:
				sb.WriteString("NaN")
			}
		}
		fmt.Printf("%s\n", sb.String())
		rowIdx++
	}
	return nil
}

func main() {
	//defer profile.Start().Stop()
	if len(os.Args) <= 1 {
		usage(nil)
	}

	switch os.Args[1] {
	case CmdDiff:
		diffCmd := flag.NewFlagSet(CmdDiff, flag.ExitOnError)
		leftPath := ""
		rightPath := ""
		if len(os.Args) >= 4 {
			leftPath = os.Args[len(os.Args)-2]
			rightPath = os.Args[len(os.Args)-1]
		}
		isIdenticalSchemas := diffCmd.Bool("identical_schemas", false, "Check if schemas are identical")
		if err := diffCmd.Parse(os.Args[2:]); err != nil || leftPath == "" || rightPath == "" {
			usage(diffCmd)
		}

		if err := diff(leftPath, rightPath, *isIdenticalSchemas); err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}
	case CmdCat:
		catCmd := flag.NewFlagSet(CmdDiff, flag.ExitOnError)
		path := ""
		if len(os.Args) >= 3 {
			path = os.Args[2]
		}
		if err := catCmd.Parse(os.Args[2:]); err != nil || path == "" {
			usage(catCmd)
		}

		if err := cat(path); err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}
	}
}
