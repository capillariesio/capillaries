package storage

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/capillariesio/capillaries/pkg/sc"
	gp "github.com/fraugster/parquet-go"
	gp_parquet "github.com/fraugster/parquet-go/parquet"
	"github.com/shopspring/decimal"
)

type ParquetWriter struct {
	FileWriter *gp.FileWriter
	StoreMap   map[string]*gp.ColumnStore // TODO: consider using w.FileWriter.GetColumnByName() instead and abandon ParquetWriter
}

func NewParquetWriter(ioWriter io.Writer, codec sc.ParquetCodecType) (*ParquetWriter, error) {
	codecMap := map[sc.ParquetCodecType]gp_parquet.CompressionCodec{
		sc.ParquetCodecGzip:         gp_parquet.CompressionCodec_GZIP,
		sc.ParquetCodecSnappy:       gp_parquet.CompressionCodec_SNAPPY,
		sc.ParquetCodecUncompressed: gp_parquet.CompressionCodec_UNCOMPRESSED,
	}
	gpCodec, ok := codecMap[codec]
	if !ok {
		return nil, fmt.Errorf("unsupported parquet codec %s", codec)
	}
	return &ParquetWriter{
		StoreMap:   map[string]*gp.ColumnStore{},
		FileWriter: gp.NewFileWriter(ioWriter, gp.WithCompressionCodec(gpCodec), gp.WithCreator("capillaries")),
	}, nil
}

func (w *ParquetWriter) AddColumn(name string, fieldType sc.TableFieldType) error {
	if _, ok := w.StoreMap[name]; ok {
		return fmt.Errorf("cannot add duplicate column %s", name)
	}

	var s *gp.ColumnStore
	var err error
	switch fieldType {
	case sc.FieldTypeString:
		params := &gp.ColumnParameters{LogicalType: gp_parquet.NewLogicalType()}
		params.LogicalType.STRING = gp_parquet.NewStringType()
		params.ConvertedType = gp_parquet.ConvertedTypePtr(gp_parquet.ConvertedType_UTF8)
		s, err = gp.NewByteArrayStore(gp_parquet.Encoding_PLAIN, true, params)
	case sc.FieldTypeDateTime:
		params := &gp.ColumnParameters{LogicalType: gp_parquet.NewLogicalType()}
		params.LogicalType.TIMESTAMP = gp_parquet.NewTimestampType()
		params.LogicalType.TIMESTAMP.Unit = gp_parquet.NewTimeUnit()
		// Go and Parquet support nanoseconds. Unfortunately, Cassandra supports only milliseconds. Millis are our lingua franca.
		params.LogicalType.TIMESTAMP.Unit.MILLIS = gp_parquet.NewMilliSeconds()
		params.ConvertedType = gp_parquet.ConvertedTypePtr(gp_parquet.ConvertedType_TIMESTAMP_MILLIS)
		s, err = gp.NewInt64Store(gp_parquet.Encoding_PLAIN, true, params)
	case sc.FieldTypeInt:
		s, err = gp.NewInt64Store(gp_parquet.Encoding_PLAIN, true, &gp.ColumnParameters{})
	case sc.FieldTypeDecimal2:
		params := &gp.ColumnParameters{LogicalType: gp_parquet.NewLogicalType()}
		params.LogicalType.DECIMAL = gp_parquet.NewDecimalType()
		params.LogicalType.DECIMAL.Scale = 2
		params.LogicalType.DECIMAL.Precision = 2
		// This is to make fraugster/go-parquet happy so it writes this metadata,
		// see buildElement() implementation in schema.go
		params.Scale = &params.LogicalType.DECIMAL.Scale
		params.Precision = &params.LogicalType.DECIMAL.Precision
		params.ConvertedType = gp_parquet.ConvertedTypePtr(gp_parquet.ConvertedType_DECIMAL)
		s, err = gp.NewInt64Store(gp_parquet.Encoding_PLAIN, true, params)
	case sc.FieldTypeFloat:
		s, err = gp.NewDoubleStore(gp_parquet.Encoding_PLAIN, true, &gp.ColumnParameters{})
	case sc.FieldTypeBool:
		s, err = gp.NewBooleanStore(gp_parquet.Encoding_PLAIN, &gp.ColumnParameters{})
	default:
		return fmt.Errorf("cannot add %s column %s: unsupported field type", fieldType, name)
	}
	if err != nil {
		return fmt.Errorf("cannot create store for %s column %s: %s", fieldType, name, err.Error())
	}
	if err := w.FileWriter.AddColumn(name, gp.NewDataColumn(s, gp_parquet.FieldRepetitionType_OPTIONAL)); err != nil {
		return fmt.Errorf("cannot add %s column %s: %s", fieldType, name, err.Error())
	}
	w.StoreMap[name] = s
	return nil
}

func (w *ParquetWriter) Close() error {
	if w.FileWriter != nil {
		if err := w.FileWriter.FlushRowGroup(); err != nil {
			return fmt.Errorf("cannot flush row group: %s", err.Error())
		}

		if err := w.FileWriter.Close(); err != nil {
			return fmt.Errorf("cannot close writer: %s", err.Error())
		}
	}
	return nil
}
func ParquetWriterMilliTs(t time.Time) interface{} {
	if t.Equal(sc.DefaultDateTime()) {
		return nil
	} else {
		// Go and Parquet support nanoseconds. Unfortunately, Cassandra supports only milliseconds. Millis are our lingua franca.
		return t.UnixMilli()
	}
}

func ParquetWriterDecimal2(dec decimal.Decimal) interface{} {
	return dec.Mul(decimal.NewFromInt(100)).IntPart()
}

func isType(se *gp_parquet.SchemaElement, t gp_parquet.Type) bool {
	return se.Type != nil && *se.Type == t
}

func isLogicalOrConvertedString(se *gp_parquet.SchemaElement) bool {
	return se.LogicalType != nil && se.LogicalType.STRING != nil ||
		se.ConvertedType != nil && *se.ConvertedType == gp_parquet.ConvertedType_UTF8
}

func isLogicalOrConvertedDecimal(se *gp_parquet.SchemaElement) bool {
	return se.LogicalType != nil && se.LogicalType.DECIMAL != nil ||
		se.ConvertedType != nil && *se.ConvertedType == gp_parquet.ConvertedType_DECIMAL
}

func isLogicalOrConvertedDateTime(se *gp_parquet.SchemaElement) bool {
	return se.LogicalType != nil && se.LogicalType.TIMESTAMP != nil ||
		se.ConvertedType != nil && (*se.ConvertedType == gp_parquet.ConvertedType_TIMESTAMP_MILLIS || *se.ConvertedType == gp_parquet.ConvertedType_TIMESTAMP_MICROS)
}

func isParquetString(se *gp_parquet.SchemaElement) bool {
	return isLogicalOrConvertedString(se) && isType(se, gp_parquet.Type_BYTE_ARRAY)
}

func isParquetIntDecimal2(se *gp_parquet.SchemaElement) bool {
	return isLogicalOrConvertedDecimal(se) &&
		(isType(se, gp_parquet.Type_INT32) || isType(se, gp_parquet.Type_INT64)) &&
		se.Scale != nil && *se.Scale > -20 && *se.Scale < 20 &&
		se.Precision != nil && *se.Precision >= 0 && *se.Precision < 18
}

func isParquetFixedLengthByteArrayDecimal2(se *gp_parquet.SchemaElement) bool {
	return isLogicalOrConvertedDecimal(se) &&
		isType(se, gp_parquet.Type_FIXED_LEN_BYTE_ARRAY) &&
		se.Scale != nil && *se.Scale > -20 && *se.Scale < 20 &&
		se.Precision != nil && *se.Precision >= 0 && *se.Precision <= 38
}

func isParquetDateTime(se *gp_parquet.SchemaElement) bool {
	return isLogicalOrConvertedDateTime(se) &&
		(isType(se, gp_parquet.Type_INT32) || isType(se, gp_parquet.Type_INT64))
}

func isParquetInt96Date(se *gp_parquet.SchemaElement) bool {
	return isType(se, gp_parquet.Type_INT96)
}

func isParquetInt32Date(se *gp_parquet.SchemaElement) bool {
	return se.Type != nil && *se.Type == gp_parquet.Type_INT32 &&
		se.LogicalType != nil && se.LogicalType.DATE != nil
}

func isParquetInt(se *gp_parquet.SchemaElement) bool {
	return (se.LogicalType == nil || se.LogicalType != nil && se.LogicalType.INTEGER != nil) &&
		se.Type != nil && (*se.Type == gp_parquet.Type_INT32 || *se.Type == gp_parquet.Type_INT64)
}

func isParquetFloat(se *gp_parquet.SchemaElement) bool {
	return se.LogicalType == nil &&
		se.Type != nil && (*se.Type == gp_parquet.Type_FLOAT || *se.Type == gp_parquet.Type_DOUBLE)
}

func isParquetBool(se *gp_parquet.SchemaElement) bool {
	return se.LogicalType == nil &&
		se.Type != nil && *se.Type == gp_parquet.Type_BOOLEAN
}

func ParquetGuessCapiType(se *gp_parquet.SchemaElement) (sc.TableFieldType, error) {
	if isParquetString(se) {
		return sc.FieldTypeString, nil
	} else if isParquetIntDecimal2(se) || isParquetFixedLengthByteArrayDecimal2(se) {
		return sc.FieldTypeDecimal2, nil
	} else if isParquetDateTime(se) || isParquetInt96Date(se) || isParquetInt32Date(se) {
		return sc.FieldTypeDateTime, nil
	} else if isParquetInt(se) {
		return sc.FieldTypeInt, nil
	} else if isParquetFloat(se) {
		return sc.FieldTypeFloat, nil
	} else if isParquetBool(se) {
		return sc.FieldTypeBool, nil
	} else {
		return sc.FieldTypeUnknown, fmt.Errorf("parquet schema element not supported: %v", se)
	}
}

func ParquetReadString(val interface{}, se *gp_parquet.SchemaElement) (string, error) {
	if !isParquetString(se) {
		return sc.DefaultString, fmt.Errorf("cannot read parquet string, schema %v", se)
	}
	typedVal, ok := val.([]byte)
	if !ok {
		return sc.DefaultString, fmt.Errorf("cannot read parquet string, schema %v, not a byte array (%T)", se, val)
	}
	return string(typedVal), nil
}

func ParquetReadDateTime(val interface{}, se *gp_parquet.SchemaElement) (time.Time, error) {
	if !isParquetDateTime(se) && !isParquetInt96Date(se) && !isParquetInt32Date(se) {
		return sc.DefaultDateTime(), fmt.Errorf("cannot read parquet datetime, schema %v", se)
	}
	// Important: all time constructor below createdatetime objects with Local TZ.
	// This is not good because our time.Format("2006-01-02") will use this TZ and produce a datetime for a local TZ, causing confusion.
	// Only UTC times should be used internally.
	switch typedVal := val.(type) {
	case int32:
		if isParquetInt32Date(se) {
			// It's a number of days from UNIX epoch
			return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(typedVal)).In(time.UTC), nil
		} else {
			switch *se.ConvertedType {
			case gp_parquet.ConvertedType_TIMESTAMP_MILLIS:
				return time.UnixMilli(int64(typedVal)).In(time.UTC), nil
			case gp_parquet.ConvertedType_TIMESTAMP_MICROS:
				return time.UnixMicro(int64(typedVal)).In(time.UTC), nil
			default:
				return sc.DefaultDateTime(), fmt.Errorf("cannot read parquet datetime from int32, unsupported converted type, schema %v", se)
			}
		}
	case int64:
		switch *se.ConvertedType {
		case gp_parquet.ConvertedType_TIMESTAMP_MILLIS:
			return time.UnixMilli(typedVal).In(time.UTC), nil
		case gp_parquet.ConvertedType_TIMESTAMP_MICROS:
			return time.UnixMicro(typedVal).In(time.UTC), nil
		default:
			return sc.DefaultDateTime(), fmt.Errorf("cannot read parquet datetime from int64, unsupported converted type, schema %v", se)
		}
	case [12]byte:
		// Deprecated parquet int96 timestamp
		return gp.Int96ToTime(typedVal).In(time.UTC), nil
	default:
		return sc.DefaultDateTime(), fmt.Errorf("cannot read parquet datetime from %T, schema %v", se, typedVal)
	}
}

func ParquetReadInt(val interface{}, se *gp_parquet.SchemaElement) (int64, error) {
	if !isParquetInt(se) {
		return sc.DefaultInt, fmt.Errorf("cannot read parquet int, schema %v", se)
	}
	switch typedVal := val.(type) {
	case int32:
		return int64(typedVal), nil
	case int64:
		return typedVal, nil
	case int16:
		return int64(typedVal), nil
	case int8:
		return int64(typedVal), nil
	case uint32:
		return int64(typedVal), nil
	case uint64:
		return int64(typedVal), nil
	case uint16:
		return int64(typedVal), nil
	case uint8:
		return int64(typedVal), nil
	default:
		return sc.DefaultInt, fmt.Errorf("cannot read parquet int from %T, schema %v", se, typedVal)
	}
}

func ParquetReadDecimal2(val interface{}, se *gp_parquet.SchemaElement) (decimal.Decimal, error) {
	if !isParquetIntDecimal2(se) && !isParquetFixedLengthByteArrayDecimal2(se) {
		return sc.DefaultDecimal2(), fmt.Errorf("cannot read parquet decimal2, schema %v", se)
	}
	switch typedVal := val.(type) {
	case int32:
		return decimal.New(int64(typedVal), -*se.Scale), nil
	case int64:
		return decimal.New(typedVal, -*se.Scale), nil
	case []byte:
		if len(typedVal) == 0 {
			return sc.DefaultDecimal2(), fmt.Errorf("cannot read parquet decimal2 from byte array of zero length, schema %v", se)
		}
		var uintVal uint64
		if len(typedVal) < 8 {
			// Pad with zeroes or ones
			padByte := byte(0)
			if (typedVal[0] & 0x10) != 0 {
				padByte = 0xFF
			}
			paddedVal := make([]byte, 8)
			firstActualByteIdx := 8 - len(typedVal)
			for i := 0; i < 8; i++ {
				if i < firstActualByteIdx {
					paddedVal[i] = padByte
				} else {
					paddedVal[i] = typedVal[i-firstActualByteIdx]
				}
			}
			uintVal = binary.BigEndian.Uint64(paddedVal)
		} else {
			// TODO: handle first len-8 bytes, they should be either all be 0xFF (negative number) or all 0x00 (positive),
			// otherwise, we are losing data. Also, pay attention to the first bit of the resulting uint64 -
			// we either lose that bit to the int64 sign (throw then), or we have to adjust the sign
			uintVal = binary.BigEndian.Uint64(typedVal[len(typedVal)-8:])
		}
		return decimal.New(int64(uintVal), -*se.Scale), nil
	default:
		return sc.DefaultDecimal2(), fmt.Errorf("cannot read parquet decimal2 from %T, schema %v", se, typedVal)
	}
}

func ParquetReadFloat(val interface{}, se *gp_parquet.SchemaElement) (float64, error) {
	if !isParquetFloat(se) {
		return sc.DefaultFloat, fmt.Errorf("cannot read parquet float, schema %v", se)
	}
	switch typedVal := val.(type) {
	case float32:
		return float64(typedVal), nil
	case float64:
		return typedVal, nil
	default:
		return sc.DefaultFloat, fmt.Errorf("cannot read parquet float from %T, schema %v", se, typedVal)
	}
}

func ParquetReadBool(val interface{}, se *gp_parquet.SchemaElement) (bool, error) {
	if !isParquetBool(se) {
		return sc.DefaultBool, fmt.Errorf("cannot read parquet float, schema %v", se)
	}
	switch typedVal := val.(type) {
	case bool:
		return typedVal, nil
	default:
		return sc.DefaultBool, fmt.Errorf("cannot read parquet bool from %T, schema %v", se, typedVal)
	}
}
