package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

type Product struct {
	Id           string
	Price        decimal.Decimal
	FreightValue decimal.Decimal
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

func main() {
	cfgRoot := "/tmp/capitest_cfg/lookup"
	inRoot := "/tmp/capitest_in/lookup"
	outRoot := "/tmp/capitest_out/lookup"

	// Read script params file to get cutoff dates and other params
	fileScriptParams := cfgRoot + "/script_params_one_run.json"
	fScriptParamsFile, err := os.Open(fileScriptParams)
	if err != nil {
		log.Fatalf("cannot open file [%s]: %s", fileScriptParams, err.Error())
	}
	defer fScriptParamsFile.Close()

	scriptParamsBytes, _ := ioutil.ReadAll(fScriptParamsFile)

	var sp ScriptParams
	if err := json.Unmarshal(scriptParamsBytes, &sp); err != nil {
		log.Fatalf("cannot unmarshal parameters [%s]: %s", fileScriptParams, err.Error())
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

	defaultProductId := ""
	defaultSellerId := ""

	fileInOrdersPath := flag.String("in_orders", inRoot+"/raw_orders", "Path to input file to generate: orders")
	fileInItemsPath := flag.String("in_items", inRoot+"/raw_items", "Path to input file to generate: order items")
	fileOutNoGroupInnerPath := flag.String("out_no_group_inner", outRoot+"/raw_no_group_inner", "Path to output file to generate: orders inner joined with items, no grouping")
	fileOutNoGroupLeftOuterPath := flag.String("out_no_group_left_outer", outRoot+"/raw_no_group_outer", "Path to output file to generate: orders left outer joined with items, no grouping")
	fileOutGroupInnerPath := flag.String("out_group_inner", outRoot+"/raw_grouped_inner", "Path to output file to generate: orders inner joined with items, grouped")
	fileOutGroupLeftOuterPath := flag.String("out_group_left_outer", outRoot+"/raw_grouped_outer", "Path to output file to generate: orders left outer joined with items, grouped")
	totalItems := flag.Int("items", 1000, "Total number of order items to generate")
	totalSellers := flag.Int("sellers", 20, "Total number of sellers to generate")
	maxProductsPerSeller := flag.Int("products", 10, "Max number of products per seller to generate")
	flag.Parse()

	// In files
	fInOrders, err := os.Create(*fileInOrdersPath)
	if err != nil {
		log.Fatalf("cannot create in file [%s]: %s", *fileInOrdersPath, err.Error())
	}
	defer fInOrders.Close()

	fInItems, err := os.Create(*fileInItemsPath)
	if err != nil {
		log.Fatalf("cannot create in file [%s]: %s", *fileInItemsPath, err.Error())
	}
	defer fInItems.Close()

	if _, err := fInOrders.WriteString("order_id,customer_id,order_status,order_purchase_timestamp,order_approved_at,order_delivered_carrier_date,order_delivered_customer_date,order_estimated_delivery_date\n"); err != nil {
		log.Fatalf("cannot write file [%s] header line: [%s]", *fileInOrdersPath, err.Error())
	}

	if _, err := fInItems.WriteString("order_id,order_item_id,product_id,seller_id,shipping_limit_date,price,freight_value\n"); err != nil {
		log.Fatalf("cannot write file [%s] header line: [%s]", *fileInItemsPath, err.Error())
	}

	// Out files
	fOutNoGroupInner, err := os.Create(*fileOutNoGroupInnerPath)
	if err != nil {
		log.Fatalf("cannot create out file [%s]: %s", *fileOutNoGroupInnerPath, err.Error())
	}
	defer fOutNoGroupInner.Close()
	fOutNoGroupLeftOuter, err := os.Create(*fileOutNoGroupLeftOuterPath)
	if err != nil {
		log.Fatalf("cannot create out file [%s]: %s", *fileOutNoGroupLeftOuterPath, err.Error())
	}
	defer fOutNoGroupLeftOuter.Close()
	fOutGroupInner, err := os.Create(*fileOutGroupInnerPath)
	if err != nil {
		log.Fatalf("cannot create out file [%s]: %s", *fileOutGroupInnerPath, err.Error())
	}
	defer fOutGroupInner.Close()
	fOutGroupLeftOuter, err := os.Create(*fileOutGroupLeftOuterPath)
	if err != nil {
		log.Fatalf("cannot create out file [%s]: %s", *fileOutGroupLeftOuterPath, err.Error())
	}
	defer fOutGroupLeftOuter.Close()

	if _, err := fOutNoGroupInner.WriteString("order_id,order_item_id,product_id,seller_id,shipping_limit_date,value\n"); err != nil {
		log.Fatalf("cannot write file [%s] header line: [%s]", *fileOutNoGroupInnerPath, err.Error())
	}
	if _, err := fOutNoGroupLeftOuter.WriteString("order_id,order_item_id,product_id,seller_id,shipping_limit_date,value\n"); err != nil {
		log.Fatalf("cannot write file [%s] header line: [%s]", *fileOutNoGroupLeftOuterPath, err.Error())
	}
	if _, err := fOutGroupInner.WriteString("total_value,order_purchase_timestamp,order_id,avg_value,min_value,max_value,min_product_id,max_product_id,item_count\n"); err != nil {
		log.Fatalf("cannot write file [%s] header line: [%s]", *fileOutGroupInnerPath, err.Error())
	}
	if _, err := fOutGroupLeftOuter.WriteString("total_value,order_purchase_timestamp,order_id,avg_value,min_value,max_value,min_product_id,max_product_id,item_count\n"); err != nil {
		log.Fatalf("cannot write file [%s] header line: [%s]", *fileOutGroupLeftOuterPath, err.Error())
	}

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
		orderId := randomId(rnd)
		projectedItemsInOrder := rnd.Intn(3) // There may be 0 items in order
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
		fInOrders.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s\n",
			orderId,
			customerId,
			orderStatus,
			timeStr(orderPurchaseTs),
			timeStr(orderApprovedTs),
			timeStr(orderDeliveredCarrierTs),
			timeStr(orderDeliveredCustomerTs),
			timeStr(orderEstimateDeliveryTs)))

		totalOrderValue := decimal.NewFromInt(0)
		minOrderValue := decimal.NewFromInt(0)
		maxOrderValue := decimal.NewFromInt(0)
		minProductId := ""
		maxProductId := ""

		actualItemsInOrder := 0

		for orderItemId := 1; orderItemId <= projectedItemsInOrder; orderItemId++ {
			sellerIdx := rnd.Intn(len(sellerIds))
			productIdx := rnd.Intn(len(sellerProducts[sellerIdx]))
			shippingLimitDate := orderEstimateDeliveryTs.Add(time.Duration(rnd.Intn(100000) * int(time.Second)))
			product := sellerProducts[sellerIdx][productIdx]
			fInItems.WriteString(fmt.Sprintf("%s,%05d,%s,%s,%s,%s,%s\n",
				orderId,
				orderItemId,
				product.Id,
				sellerIds[sellerIdx],
				timeStr(shippingLimitDate),
				product.Price,
				product.FreightValue))

			if !(sp.StartDate.After(orderPurchaseTs) || sp.EndDate.Before(orderPurchaseTs)) {
				orderItemValue := product.Price.Add(product.FreightValue)
				fOutNoGroupInner.WriteString(fmt.Sprintf("%s,%05d,%s,%s,%s,%s\n",
					orderId,
					orderItemId,
					product.Id,
					sellerIds[sellerIdx],
					timeStr(shippingLimitDate),
					orderItemValue.StringFixed(2)))

				fOutNoGroupLeftOuter.WriteString(fmt.Sprintf("%s,%05d,%s,%s,%s,%s\n",
					orderId,
					orderItemId,
					product.Id,
					sellerIds[sellerIdx],
					timeStr(shippingLimitDate),
					orderItemValue.StringFixed(2)))

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
			fOutNoGroupLeftOuter.WriteString(fmt.Sprintf("%s,%05d,%s,%s,%s,%s\n",
				orderId,
				sp.DefaultOrderItemId,
				defaultProductId,
				defaultSellerId,
				timeStr(sp.DefaultShippingLimitDate),
				sp.DefaultOrderItemValue.StringFixed(2)))
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
			fOutGroupInner.WriteString(fmt.Sprintf("%10s,%s,%s,%s,%s,%s,%s,%s,%d\n",
				totalOrderValue.StringFixed(2),
				timeStr(orderPurchaseTs),
				orderId,
				avgOrderValue.StringFixed(2),
				minOrderValue.StringFixed(2),
				maxOrderValue.StringFixed(2),
				minProductId,
				maxProductId,
				actualItemsInOrder))
		}

		// Outer
		fOutGroupLeftOuter.WriteString(fmt.Sprintf("%10s,%s,%s,%s,%s,%s,%s,%s,%d\n",
			totalOrderValue.StringFixed(2),
			timeStr(orderPurchaseTs),
			orderId,
			avgOrderValue.StringFixed(2),
			minOrderValue.StringFixed(2),
			maxOrderValue.StringFixed(2),
			minProductId,
			maxProductId,
			actualItemsInOrder))
	}
}
