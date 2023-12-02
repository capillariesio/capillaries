package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/storage"
	"github.com/shopspring/decimal"
)

type Product struct {
	Id           string
	Price        decimal.Decimal
	FreightValue decimal.Decimal
}

type Order struct {
	OrderId                  string
	CustomerId               string
	OrderStatus              string
	OrderPurchaseTs          time.Time
	OrderApprovedTs          time.Time
	OrderDeliveredCarrierTs  time.Time
	OrderDeliveredCustomerTs time.Time
	OrderEstimateDeliveryTs  time.Time
}

type OrderItem struct {
	OrderId           string
	OrderItemId       int64
	ProductId         string
	SellerId          string
	ShippingLimitDate time.Time
	Price             decimal.Decimal
	FreightValue      decimal.Decimal
}

type NoGroupItem struct {
	OrderId           string
	OrderItemId       int64
	ProductId         string
	SellerId          string
	ShippingLimitDate time.Time
	OrderItemValue    decimal.Decimal
}

type GroupItem struct {
	TotalOrderValue    decimal.Decimal
	OrderPurchaseTs    time.Time
	OrderId            string
	AvgOrderValue      decimal.Decimal
	MinOrderValue      decimal.Decimal
	MaxOrderValue      decimal.Decimal
	MinProductId       string
	MaxProductId       string
	ActualItemsInOrder int64
}

func randomId(rnd *rand.Rand) string {
	return fmt.Sprintf("%016x%016x", rnd.Int63(), rnd.Int63())
}

func timeStr(ts time.Time) string {
	if ts.Unix() != 0 {
		return ts.Format("2006-01-02 15:04:05")
	} else {
		return ""
	}
}

type ScriptParams struct {
	StartDateString                string `json:"start_date"`
	EndDateString                  string `json:"end_date"`
	DefaultShippingLimitDateString string `json:"default_shipping_limit_date"` // 1980-02-03T04:05:06.777+00:00
	DefaultOrderItemIdString       string `json:"default_order_item_id"`       // 999`
	DefaultOrderItemValueString    string `json:"default_order_item_value"`    // 999.00
	StartDate                      time.Time
	EndDate                        time.Time
	DefaultShippingLimitDate       time.Time
	DefaultOrderItemId             int64
	DefaultOrderItemValue          decimal.Decimal
}

func shuffleAndSaveInOrders(inOrders []*Order, totalChunks int, basePath string, formats string) {
	rnd := rand.New(rand.NewSource((time.Now().Unix() << 32) + time.Now().UnixMilli()))

	for i := 0; i < len(inOrders); i++ {
		j := i
		for j == i {
			j = rnd.Intn(len(inOrders))
		}
		tmp := inOrders[i]
		inOrders[i] = inOrders[j]
		inOrders[j] = tmp
	}

	chunkSize := len(inOrders)
	if totalChunks > 1 {
		chunkSize = int(math.Ceil(float64(len(inOrders)) / float64(totalChunks)))
	}

	var fCsv *os.File
	var fParquet *os.File
	var parquetWriter *storage.ParquetWriter
	chunkLineCount := 0
	chunkIdx := 0
	var err error
	for itemIdx, item := range inOrders {
		if chunkLineCount == 0 {
			finalFilePath := basePath
			if totalChunks > 1 {
				finalFilePath = fmt.Sprintf("%s_%02d", basePath, chunkIdx)
			}

			if strings.Contains(formats, "csv") {
				fCsv, err = os.Create(finalFilePath + ".csv")
				if err != nil {
					log.Fatalf("cannot create in file [%s]: %s", finalFilePath, err.Error())
				}
				if _, err := fCsv.WriteString("order_id,customer_id,order_status,order_purchase_timestamp,order_approved_at,order_delivered_carrier_date,order_delivered_customer_date,order_estimated_delivery_date\n"); err != nil {
					log.Fatalf("cannot write file [%s] header line: [%s]", finalFilePath, err.Error())
				}
			}

			if strings.Contains(formats, "parquet") {
				fParquet, err = os.Create(finalFilePath + ".parquet")
				if err != nil {
					log.Fatalf("cannot create in file [%s]: %s", finalFilePath, err.Error())
				}
				parquetWriter, err = storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
				if err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("order_id", sc.FieldTypeString); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("customer_id", sc.FieldTypeString); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("order_status", sc.FieldTypeString); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("order_purchase_timestamp", sc.FieldTypeDateTime); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("order_approved_at", sc.FieldTypeDateTime); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("order_delivered_carrier_date", sc.FieldTypeDateTime); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("order_delivered_customer_date", sc.FieldTypeDateTime); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("order_estimated_delivery_date", sc.FieldTypeDateTime); err != nil {
					log.Fatalf(err.Error())
				}
				// Test only
				// if err := w.AddColumn("is_sent", sc.FieldTypeBool); err != nil {
				// 	log.Fatalf(err.Error())
				// }
			}
		}

		if strings.Contains(formats, "csv") {
			fCsv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s\n",
				item.OrderId,
				item.CustomerId,
				item.OrderStatus,
				timeStr(item.OrderPurchaseTs),
				timeStr(item.OrderApprovedTs),
				timeStr(item.OrderDeliveredCarrierTs),
				timeStr(item.OrderDeliveredCustomerTs),
				timeStr(item.OrderEstimateDeliveryTs)))
		}

		if strings.Contains(formats, "parquet") {
			if err := parquetWriter.FileWriter.AddData(map[string]any{
				"order_id":                      item.OrderId,
				"customer_id":                   item.CustomerId,
				"order_status":                  item.OrderStatus,
				"order_purchase_timestamp":      storage.ParquetWriterMilliTs(item.OrderPurchaseTs),
				"order_approved_at":             storage.ParquetWriterMilliTs(item.OrderApprovedTs),
				"order_delivered_carrier_date":  storage.ParquetWriterMilliTs(item.OrderDeliveredCarrierTs),
				"order_delivered_customer_date": storage.ParquetWriterMilliTs(item.OrderDeliveredCustomerTs),
				"order_estimated_delivery_date": storage.ParquetWriterMilliTs(item.OrderEstimateDeliveryTs),
				// "is_sent":                  !item.OrderDeliveredCarrierTs.Equal(sc.DefaultDateTime()),
			}); err != nil {
				log.Fatalf(err.Error())
			}
		}
		chunkLineCount++
		if chunkLineCount == chunkSize || itemIdx == len(inOrders)-1 {
			if strings.Contains(formats, "csv") {
				fCsv.Close()
			}
			if strings.Contains(formats, "parquet") {
				if fParquet != nil {
					if err := parquetWriter.Close(); err != nil {
						log.Fatalf("cannot complete parquet file [%s]: %s", basePath, err.Error())
					}
					fParquet.Close()
				}
			}
			chunkLineCount = 0
			chunkIdx++
		}
	}
}

func shuffleAndSaveInOrderItems(inOrderItems []*OrderItem, totalChunks int, basePath string, formats string) {
	rnd := rand.New(rand.NewSource((time.Now().Unix() << 32) + time.Now().UnixMilli()))
	for i := 0; i < len(inOrderItems); i++ {
		j := i
		for j == i {
			j = rnd.Intn(len(inOrderItems))
		}
		tmp := inOrderItems[i]
		inOrderItems[i] = inOrderItems[j]
		inOrderItems[j] = tmp
	}

	chunkSize := len(inOrderItems)
	if totalChunks > 1 {
		chunkSize = int(math.Ceil(float64(len(inOrderItems)) / float64(totalChunks)))
	}

	var fCsv *os.File
	var fParquet *os.File
	var parquetWriter *storage.ParquetWriter
	chunkLineCount := 0
	chunkIdx := 0
	var err error
	for itemIdx, item := range inOrderItems {
		if chunkLineCount == 0 {
			finalFilePath := basePath
			if totalChunks > 1 {
				finalFilePath = fmt.Sprintf("%s_%02d", basePath, chunkIdx)
			}
			if strings.Contains(formats, "csv") {
				fCsv, err = os.Create(finalFilePath + ".csv")
				if err != nil {
					log.Fatalf("cannot create in file [%s]: %s", finalFilePath, err.Error())
				}
				if _, err := fCsv.WriteString("order_id,order_item_id,product_id,seller_id,shipping_limit_date,price,freight_value\n"); err != nil {
					log.Fatalf("cannot write file [%s] header line: [%s]", finalFilePath, err.Error())
				}
			}
			if strings.Contains(formats, "parquet") {
				fParquet, err = os.Create(finalFilePath + ".parquet")
				if err != nil {
					log.Fatalf("cannot create in file [%s]: %s", finalFilePath, err.Error())
				}
				parquetWriter, err = storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
				if err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("order_id", sc.FieldTypeString); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("order_item_id", sc.FieldTypeInt); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("product_id", sc.FieldTypeString); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("seller_id", sc.FieldTypeString); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("shipping_limit_date", sc.FieldTypeDateTime); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("price", sc.FieldTypeDecimal2); err != nil {
					log.Fatalf(err.Error())
				}
				if err := parquetWriter.AddColumn("freight_value", sc.FieldTypeDecimal2); err != nil {
					log.Fatalf(err.Error())
				}
			}
		}
		if strings.Contains(formats, "csv") {
			fCsv.WriteString(fmt.Sprintf("%s,%05d,%s,%s,%s,%s,%s\n",
				item.OrderId,
				item.OrderItemId,
				item.ProductId,
				item.SellerId,
				timeStr(item.ShippingLimitDate),
				item.Price,
				item.FreightValue))
		}
		if strings.Contains(formats, "parquet") {
			if err := parquetWriter.FileWriter.AddData(map[string]any{
				"order_id":            item.OrderId,
				"order_item_id":       item.OrderItemId,
				"product_id":          item.ProductId,
				"seller_id":           item.SellerId,
				"shipping_limit_date": storage.ParquetWriterMilliTs(item.ShippingLimitDate),
				"price":               storage.ParquetWriterDecimal2(item.Price),
				"freight_value":       storage.ParquetWriterDecimal2(item.FreightValue),
			}); err != nil {
				log.Fatalf(err.Error())
			}
		}

		chunkLineCount++
		if chunkLineCount == chunkSize || itemIdx == len(inOrderItems)-1 {
			if strings.Contains(formats, "csv") {
				fCsv.Close()
			}
			if strings.Contains(formats, "parquet") {
				if fParquet != nil {
					if err := parquetWriter.Close(); err != nil {
						log.Fatalf("cannot complete parquet file [%s]: %s", basePath, err.Error())
					}
					fParquet.Close()
				}
			}
			chunkLineCount = 0
			chunkIdx++
		}
	}
}

func sortAndSaveNoGroup(items []*NoGroupItem, fileBase string, formats string) {
	// "order": "order_id(asc),order_item_id(asc)"
	sort.Slice(items, func(i, j int) bool {
		if items[i].OrderId != items[j].OrderId {
			return items[i].OrderId < items[j].OrderId
		} else {
			return items[i].OrderItemId < items[j].OrderItemId
		}
	})
	if strings.Contains(formats, "csv") {
		csvFilePath := fileBase + ".csv"
		fCsv, err := os.Create(csvFilePath)
		if err != nil {
			log.Fatalf("cannot create file '%s': %s", csvFilePath, err.Error())
		}
		defer fCsv.Close()

		if _, err := fCsv.WriteString("order_id,order_item_id,product_id,seller_id,shipping_limit_date,value\n"); err != nil {
			log.Fatalf("cannot write file '%s' header line: %s", csvFilePath, err.Error())
		}

		for i := 0; i < len(items); i++ {
			if _, err := fCsv.WriteString(fmt.Sprintf("%s,%05d,%s,%s,%s,%s\n",
				items[i].OrderId,
				items[i].OrderItemId,
				items[i].ProductId,
				items[i].SellerId,
				timeStr(items[i].ShippingLimitDate),
				items[i].OrderItemValue.StringFixed(2))); err != nil {
				log.Fatalf("cannot write file '%s': %s", csvFilePath, err.Error())
			}

		}
	}

	if strings.Contains(formats, "parquet") {
		parquetFilePath := fileBase + ".parquet"
		fParquet, err := os.Create(parquetFilePath)
		if err != nil {
			log.Fatalf("cannot create file '%s': %s", parquetFilePath, err.Error())
		}

		w, err := storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
		if err != nil {
			log.Fatalf(err.Error())
		}

		if err := w.AddColumn("order_id", sc.FieldTypeString); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("order_item_id", sc.FieldTypeInt); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("product_id", sc.FieldTypeString); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("seller_id", sc.FieldTypeString); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("shipping_limit_date", sc.FieldTypeDateTime); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("value", sc.FieldTypeDecimal2); err != nil {
			log.Fatalf(err.Error())
		}

		for _, item := range items {
			if err := w.FileWriter.AddData(map[string]any{
				"order_id":            item.OrderId,
				"order_item_id":       item.OrderItemId,
				"product_id":          item.ProductId,
				"seller_id":           item.SellerId,
				"shipping_limit_date": storage.ParquetWriterMilliTs(item.ShippingLimitDate),
				"value":               storage.ParquetWriterDecimal2(item.OrderItemValue),
			}); err != nil {
				log.Fatalf(err.Error())
			}
		}

		if err := w.Close(); err != nil {
			log.Fatalf("cannot close parquet writer '%s': %s", parquetFilePath, err.Error())
		}

		if err := fParquet.Close(); err != nil {
			log.Fatalf("cannot close file '%s': %s", parquetFilePath, err.Error())
		}
	}
}

func sortAndSaveGroup(items []*GroupItem, fileBase string, formats string) {
	// "order": "total_value(desc),order_purchase_timestamp(desc),order_id(desc)"
	sort.Slice(items, func(i, j int) bool {
		if !items[i].TotalOrderValue.Equal(items[j].TotalOrderValue) {
			return items[i].TotalOrderValue.GreaterThan(items[j].TotalOrderValue)
		} else if !items[i].OrderPurchaseTs.Equal(items[j].OrderPurchaseTs) {
			return items[i].OrderPurchaseTs.After(items[j].OrderPurchaseTs)
		} else {
			return items[i].OrderId > items[j].OrderId
		}
	})

	if strings.Contains(formats, "csv") {
		csvFilePath := fileBase + ".csv"
		fCsv, err := os.Create(csvFilePath)
		if err != nil {
			log.Fatalf("cannot create out file '%s': %s", csvFilePath, err.Error())
		}
		defer fCsv.Close()

		if _, err := fCsv.WriteString("total_value,order_purchase_timestamp,order_id,avg_value,min_value,max_value,min_product_id,max_product_id,item_count\n"); err != nil {
			log.Fatalf("cannot write file '%s' header line: %s", csvFilePath, err.Error())
		}

		for i := 0; i < len(items); i++ {
			if _, err := fCsv.WriteString(fmt.Sprintf("%10s,%s,%s,%s,%s,%s,%s,%s,%d\n",
				items[i].TotalOrderValue.StringFixed(2),
				timeStr(items[i].OrderPurchaseTs),
				items[i].OrderId,
				items[i].AvgOrderValue.StringFixed(2),
				items[i].MinOrderValue.StringFixed(2),
				items[i].MaxOrderValue.StringFixed(2),
				items[i].MinProductId,
				items[i].MaxProductId,
				items[i].ActualItemsInOrder)); err != nil {
				log.Fatalf("cannot write file '%s': %s", csvFilePath, err.Error())
			}
		}
	}

	if strings.Contains(formats, "parquet") {
		parquetFilePath := fileBase + ".parquet"
		fParquet, err := os.Create(parquetFilePath)
		if err != nil {
			log.Fatalf("cannot create file '%s': %s", parquetFilePath, err.Error())
		}

		w, err := storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
		if err != nil {
			log.Fatalf(err.Error())
		}

		if err := w.AddColumn("total_value", sc.FieldTypeDecimal2); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("order_purchase_timestamp", sc.FieldTypeDateTime); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("order_id", sc.FieldTypeString); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("avg_value", sc.FieldTypeDecimal2); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("min_value", sc.FieldTypeDecimal2); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("max_value", sc.FieldTypeDecimal2); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("min_product_id", sc.FieldTypeString); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("max_product_id", sc.FieldTypeString); err != nil {
			log.Fatalf(err.Error())
		}
		if err := w.AddColumn("item_count", sc.FieldTypeInt); err != nil {
			log.Fatalf(err.Error())
		}

		for _, item := range items {
			if err := w.FileWriter.AddData(map[string]any{
				"total_value":              storage.ParquetWriterDecimal2(item.TotalOrderValue),
				"order_purchase_timestamp": storage.ParquetWriterMilliTs(item.OrderPurchaseTs),
				"order_id":                 item.OrderId,
				"avg_value":                storage.ParquetWriterDecimal2(item.AvgOrderValue),
				"min_value":                storage.ParquetWriterDecimal2(item.MinOrderValue),
				"max_value":                storage.ParquetWriterDecimal2(item.MaxOrderValue),
				"min_product_id":           item.MinProductId,
				"max_product_id":           item.MaxProductId,
				"item_count":               item.ActualItemsInOrder,
			}); err != nil {
				log.Fatalf(err.Error())
			}
		}

		if err := w.Close(); err != nil {
			log.Fatalf("cannot close parquet writer '%s': %s", parquetFilePath, err.Error())
		}

		if err := fParquet.Close(); err != nil {
			log.Fatalf("cannot close file '%s': %s", parquetFilePath, err.Error())
		}
	}
}

func main() {
	defaultProductId := ""
	defaultSellerId := ""

	scriptParamsPath := flag.String("script_params_path", "../../data/cfg/lookup/script_params_one_run.json", "Path to lookup script parameters")
	formats := flag.String("formats", "csv", "Comma-separated list of file formats to produce, for exanple: csv,parquet")
	inRoot := flag.String("in_root", "/tmp/capi_in/lookup_quicktest", "Root dir for in files")
	outRoot := flag.String("out_root", "/tmp/capi_out/lookup_quicktest", "Root dir for out files")
	totalItems := flag.Int("items", 1000, "Total number of order items to generate")
	totalSellers := flag.Int("sellers", 20, "Total number of sellers to generate")
	maxProductsPerSeller := flag.Int("products", 10, "Max number of products per seller to generate")
	splitOrders := flag.Int("split_orders", 1, "Number of in order files to generate")
	splitOrderItems := flag.Int("split_items", 1, "Number of in order item files to generate")
	flag.Parse()

	fileInOrdersPath := *inRoot + "/olist_orders_dataset"
	fileInItemsPath := *inRoot + "/olist_order_items_dataset"
	fileOutNoGroupInnerPath := *outRoot + "/order_item_date_inner_baseline"
	fileOutNoGroupLeftOuterPath := *outRoot + "/order_item_date_left_outer_baseline"
	fileOutGroupInnerPath := *outRoot + "/order_date_value_grouped_inner_baseline"
	fileOutGroupLeftOuterPath := *outRoot + "/order_date_value_grouped_left_outer_baseline"

	// Read script params file to get cutoff dates and other params
	fScriptParamsFile, err := os.Open(*scriptParamsPath)
	if err != nil {
		log.Fatalf("cannot open file [%s]: %s", *scriptParamsPath, err.Error())
	}
	defer fScriptParamsFile.Close()

	scriptParamsBytes, _ := ioutil.ReadAll(fScriptParamsFile)

	var sp ScriptParams
	if err := json.Unmarshal(scriptParamsBytes, &sp); err != nil {
		log.Fatalf("cannot unmarshal parameters [%s]: %s", *scriptParamsPath, err.Error())
	}

	sp.StartDate, err = time.Parse("2006-01-02 15:04:05", sp.StartDateString)
	if err != nil {
		log.Fatalf("cannot unmarshal start date [%s]: %s", sp.StartDateString, err.Error())
	}
	sp.EndDate, err = time.Parse("2006-01-02 15:04:05", sp.EndDateString)
	if err != nil {
		log.Fatalf("cannot unmarshal end date [%s]: %s", sp.EndDateString, err.Error())
	}
	sp.DefaultShippingLimitDate, err = time.Parse("2006-01-02T15:04:05.000+00:00", sp.DefaultShippingLimitDateString)
	if err != nil {
		log.Fatalf("cannot unmarshal default shipping limit date [%s]: %s", sp.DefaultShippingLimitDateString, err.Error())
	}
	sp.DefaultOrderItemId, err = strconv.ParseInt(sp.DefaultOrderItemIdString, 10, 64)
	if err != nil {
		log.Fatalf("cannot unmarshal default order item id [%s]: %s", sp.DefaultOrderItemIdString, err.Error())
	}
	sp.DefaultOrderItemValue, err = decimal.NewFromString(sp.DefaultOrderItemValueString)
	if err != nil {
		log.Fatalf("cannot unmarshal default order item value [%s]: %s", sp.DefaultOrderItemValueString, err.Error())
	}

	// Data in
	inOrders := make([]*Order, 0)
	inOrderItems := make([]*OrderItem, 0)

	// Data out
	noGroupInnerItems := make([]*NoGroupItem, 0)
	noGroupOuterItems := make([]*NoGroupItem, 0)
	groupInnerItems := make([]*GroupItem, 0)
	groupOuterItems := make([]*GroupItem, 0)

	seed := (time.Now().Unix() << 32) + time.Now().UnixMilli()
	fmt.Println("Seed:", seed)
	rnd := rand.New(rand.NewSource(seed))

	sellerIds := make([]string, *totalSellers)
	sellerProducts := make([][]*Product, *totalSellers)
	for sellerIdx := 0; sellerIdx < len(sellerIds); sellerIdx++ {
		sellerIds[sellerIdx] = randomId(rnd)
		productsPerSeller := rnd.Intn(*maxProductsPerSeller) + 1
		sellerProducts[sellerIdx] = make([]*Product, productsPerSeller)
		for productIdx := 0; productIdx < len(sellerProducts[sellerIdx]); productIdx++ {
			sellerProducts[sellerIdx][productIdx] = &Product{Id: randomId(rnd),
				Price:        decimal.NewFromFloat32(float32(500+rnd.Intn(29500)) / 100), // Min $5, max $299.99
				FreightValue: decimal.NewFromFloat32(float32(200+rnd.Intn(4800)) / 100)}  // Min $2, max $49.99
		}
	}

	itemIdx := 0
	for itemIdx < *totalItems {
		// Generate random order data
		orderId := randomId(rnd)
		projectedItemsInOrder := rnd.Intn(5) // There may be [0,4] items in order
		orderStatus := "invoiced"
		orderPurchaseTs := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(rnd.Intn(100000000) * int(time.Second)))
		orderEstimateDeliveryTs := orderPurchaseTs.Add(time.Duration(rnd.Intn(100000) * int(time.Second)))
		var orderApprovedTs time.Time
		var orderDeliveredCarrierTs time.Time
		var orderDeliveredCustomerTs time.Time

		if rnd.Intn(100) < 98 {
			orderStatus = "approved"
			orderApprovedTs = orderPurchaseTs.Add(time.Duration(rnd.Intn(10000) * int(time.Second)))
			if rnd.Intn(100) < 98 {
				orderStatus = "shipped"
				orderDeliveredCarrierTs = orderApprovedTs.Add(time.Duration(rnd.Intn(10000) * int(time.Second)))
				if rnd.Intn(100) < 98 {
					orderStatus = "delivered"
					orderDeliveredCustomerTs = orderDeliveredCarrierTs.Add(time.Duration(rnd.Intn(100000) * int(time.Second)))
				}
			}
		}
		if rnd.Intn(100) < 2 {
			orderStatus = "canceled"
		}
		if rnd.Intn(100) < 2 {
			orderStatus = "unavailable"
		}
		customerId := randomId(rnd)

		// Create order from data
		inOrders = append(inOrders, &Order{
			orderId,
			customerId,
			orderStatus,
			orderPurchaseTs,
			orderApprovedTs,
			orderDeliveredCarrierTs,
			orderDeliveredCustomerTs,
			orderEstimateDeliveryTs,
		})

		// Generate order items
		totalOrderValue := decimal.NewFromInt(0)
		minOrderValue := decimal.NewFromInt(0)
		maxOrderValue := decimal.NewFromInt(0)
		minProductId := ""
		maxProductId := ""

		actualItemsInOrder := 0

		for orderItemId := 1; orderItemId <= projectedItemsInOrder; orderItemId++ {
			// Generate random item data
			sellerIdx := rnd.Intn(len(sellerIds))
			productIdx := rnd.Intn(len(sellerProducts[sellerIdx]))
			shippingLimitDate := orderEstimateDeliveryTs.Add(time.Duration(rnd.Intn(100000) * int(time.Second)))
			product := sellerProducts[sellerIdx][productIdx]

			// Create item from data
			inOrderItems = append(inOrderItems, &OrderItem{
				orderId,
				int64(orderItemId),
				product.Id,
				sellerIds[sellerIdx],
				shippingLimitDate,
				product.Price,
				product.FreightValue,
			})

			// Compute baseline outputs
			if !(sp.StartDate.After(orderPurchaseTs) || sp.EndDate.Before(orderPurchaseTs)) {
				orderItemValue := product.Price.Add(product.FreightValue)
				noGroupInnerItems = append(noGroupInnerItems, &NoGroupItem{
					orderId,
					int64(orderItemId),
					product.Id,
					sellerIds[sellerIdx],
					shippingLimitDate,
					orderItemValue,
				})
				noGroupOuterItems = append(noGroupOuterItems, &NoGroupItem{
					orderId,
					int64(orderItemId),
					product.Id,
					sellerIds[sellerIdx],
					shippingLimitDate,
					orderItemValue,
				})
				totalOrderValue = totalOrderValue.Add(orderItemValue)
				if orderItemValue.LessThan(minOrderValue) || orderItemId == 1 {
					minOrderValue = orderItemValue
				}
				if orderItemValue.GreaterThan(maxOrderValue) || orderItemId == 1 {
					maxOrderValue = orderItemValue
				}
				if product.Id < minProductId || orderItemId == 1 {
					minProductId = product.Id
				}
				if product.Id > maxProductId || orderItemId == 1 {
					maxProductId = product.Id
				}
			}

			actualItemsInOrder++

			itemIdx++
			if itemIdx == *totalItems {
				break
			}
		}

		if sp.StartDate.After(orderPurchaseTs) || sp.EndDate.Before(orderPurchaseTs) {
			continue
		}

		if actualItemsInOrder == 0 {
			// Blank joined items for outer
			noGroupOuterItems = append(noGroupOuterItems, &NoGroupItem{
				orderId,
				sp.DefaultOrderItemId,
				defaultProductId,
				defaultSellerId,
				sp.DefaultShippingLimitDate,
				sp.DefaultOrderItemValue,
			})
		}

		// Grouped
		avgOrderValue := decimal.NewFromInt(0)
		if actualItemsInOrder == 0 {
			minOrderValue = decimal.NewFromInt(0)
			maxOrderValue = decimal.NewFromInt(0)
			minProductId = ""
			maxProductId = ""
		} else {
			avgOrderValue = totalOrderValue.Div(decimal.NewFromInt(int64(actualItemsInOrder))).Round(2)
		}

		// Inner
		if actualItemsInOrder > 0 {
			groupInnerItems = append(groupInnerItems, &GroupItem{
				totalOrderValue,
				orderPurchaseTs,
				orderId,
				avgOrderValue,
				minOrderValue,
				maxOrderValue,
				minProductId,
				maxProductId,
				int64(actualItemsInOrder),
			})
		}

		// Outer
		groupOuterItems = append(groupOuterItems, &GroupItem{
			totalOrderValue,
			orderPurchaseTs,
			orderId,
			avgOrderValue,
			minOrderValue,
			maxOrderValue,
			minProductId,
			maxProductId,
			int64(actualItemsInOrder),
		})
	}

	shuffleAndSaveInOrders(inOrders, *splitOrders, fileInOrdersPath, *formats)
	shuffleAndSaveInOrderItems(inOrderItems, *splitOrderItems, fileInItemsPath, *formats)

	sortAndSaveNoGroup(noGroupInnerItems, fileOutNoGroupInnerPath, *formats)
	sortAndSaveNoGroup(noGroupOuterItems, fileOutNoGroupLeftOuterPath, *formats)
	sortAndSaveGroup(groupInnerItems, fileOutGroupInnerPath, *formats)
	sortAndSaveGroup(groupOuterItems, fileOutGroupLeftOuterPath, *formats)
}
