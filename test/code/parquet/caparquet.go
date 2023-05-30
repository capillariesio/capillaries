package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	gp "github.com/fraugster/parquet-go"
	"github.com/fraugster/parquet-go/parquet"
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

func diff(leftPath string, rightPath string, isCheckCodec bool) error {
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

	if schemaLeft.String() != schemaRight.String() {
		return fmt.Errorf("schemas do not match:\n\n%s\n\n%s", schemaLeft.String(), schemaRight.String())
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
			return fmt.Errorf("cannot get left row %d: %s", rowIdx, err.Error())
		} else if errRight != nil {
			return fmt.Errorf("cannot get right row %d: %s", rowIdx, err.Error())
		}

		// Compare two maps
		if !reflect.DeepEqual(dLeft, dRight) {
			return fmt.Errorf("mismatch:\n%v\n%v", dLeft, dRight)
		}
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
		for _, k := range fields {
			if sb.Len() > 0 {
				sb.WriteString(",")
			}

			// HERE !!!

			// se, _ := schemaElementMap[k]
			// switch se.
			sb.WriteString(fmt.Sprintf("%v", d[k]))
		}
		fmt.Printf("%s\n", strings.Join(fields, ","))
		fmt.Printf("%s\n", sb.String())
		//fmt.Printf("%v\n", d)
	}
	return nil
}

func main() {
	if len(os.Args) <= 1 {
		usage(nil)
	}

	switch os.Args[1] {
	case CmdDiff:
		diffCmd := flag.NewFlagSet(CmdDiff, flag.ExitOnError)
		leftPath := ""
		if len(os.Args) >= 3 {
			leftPath = os.Args[2]
		}
		rightPath := ""
		if len(os.Args) >= 4 {
			rightPath = os.Args[3]
		}
		checkCodec := diffCmd.Bool("check_codec", false, "Check if the same compression codec is used")
		if err := diffCmd.Parse(os.Args[2:]); err != nil || leftPath == "" || rightPath == "" {
			usage(diffCmd)
		}

		if err := diff(leftPath, rightPath, *checkCodec); err != nil {
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
