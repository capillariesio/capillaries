package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
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

func dayOfWeekName(weekday time.Weekday) string {
	switch weekday {
	case time.Monday:
		return "Monday"
	case time.Tuesday:
		return "Tuesday"
	case time.Wednesday:
		return "Wednesday"
	case time.Thursday:
		return "Thursday"
	case time.Friday:
		return "Friday"
	case time.Saturday:
		return "Saturday"
	case time.Sunday:
		return "Sunday"
	default:
		return "Unknown day of week"
	}
}

func main() {
	fileInPath := flag.String("in_file", "", "Path to input data file to generate")
	fileOutPyPath := flag.String("out_file_py", "", "Path to Python output data file to generate")
	fileOutGoPath := flag.String("out_file_go", "", "Path to Go output data file to generate")
	totalItems := flag.Int("items", 1000, "Total number of order items to generate")
	totalSellers := flag.Int("sellers", 100, "Total number of sellers to generate")
	maxProductsPerSeller := flag.Int("products", 100, "Max number of products per seller to generate")
	flag.Parse()

	fIn, err := os.Create(*fileInPath)
	if err != nil {
		log.Fatalf("cannot create in file [%s]: %s", *fileInPath, err.Error())
	}
	defer fIn.Close()

	fOutPy, err := os.Create(*fileOutPyPath)
	if err != nil {
		log.Fatalf("cannot create out file [%s]: %s", *fileOutPyPath, err.Error())
	}
	defer fOutPy.Close()

	fOutGo, err := os.Create(*fileOutGoPath)
	if err != nil {
		log.Fatalf("cannot create out file [%s]: %s", *fileOutGoPath, err.Error())
	}
	defer fOutGo.Close()

	if _, err := fIn.WriteString("order_id,order_item_id,product_id,seller_id,shipping_limit_date,price,freight_value\n"); err != nil {
		log.Fatalf("cannot write file [%s] header line: [%s]", *fileInPath, err.Error())
	}

	if _, err := fOutPy.WriteString("order_id,order_item_id,shipping_limit_date,price,freight_value,value,taxed_value,taxed_value_divided_by_nine_float,taxed_value_divided_by_nine_decimal,shipping_limit_date_monday,shipping_limit_day_of_week,shipping_limit_is_weekend\n"); err != nil {
		log.Fatalf("cannot write file [%s] header line: [%s]", *fileOutPyPath, err.Error())
	}
	if _, err := fOutGo.WriteString("order_id,order_item_id,shipping_limit_date,price,freight_value,value,taxed_value,taxed_value_divided_by_nine_float,taxed_value_divided_by_nine_decimal\n"); err != nil {
		log.Fatalf("cannot write file [%s] header line: [%s]", *fileOutGoPath, err.Error())
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
		itemsInOrder := rnd.Intn(10) + 1
		orderTs := time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(rnd.Intn(100000000) * int(time.Second)))
		for orderItemId := 1; orderItemId <= itemsInOrder; orderItemId++ {
			sellerIdx := rnd.Intn(len(sellerIds))
			productIdx := rnd.Intn(len(sellerProducts[sellerIdx]))
			shippingLimitDate := orderTs.Add(time.Duration(rnd.Intn(1000000) * int(time.Second)))
			product := sellerProducts[sellerIdx][productIdx]
			fIn.WriteString(fmt.Sprintf("%s,%05d,%s,%s,%s,%s,%s\n",
				orderId,
				orderItemId,
				product.Id,
				sellerIds[sellerIdx],
				shippingLimitDate.Format("2006-01-02 15:04:05"),
				product.Price,
				product.FreightValue))

			// Mimic Python calculations for taxedValue, so use float, not decimal
			taxedValueFloatPython, _ := product.Price.Add(product.FreightValue).Float64()
			taxedValueFloatPython = taxedValueFloatPython * float64(1.1)
			taxedValueFloatFlooredInPython := math.Floor(taxedValueFloatPython*100) / 100 // Watch that floor() call in increase_by_ten_percent() implementation
			taxedValueByNineFloatPython := taxedValueFloatFlooredInPython / 3 / 3         // Apply divide_by_three() twice
			taxedValueByNineDecPython := decimal.NewFromFloat(taxedValueByNineFloatPython).Round(2)

			// Mimic python next_local_monday(), but keep in mind that golang uses US days: Sun 0, Mon 1, ... ,Sat 6, while Python does Mon 0 to Sun 6
			weekdayDelta := (7-int(shippingLimitDate.Weekday()))%7 + 1 // Mon -> 0, Sun->1, Sat->2...
			if weekdayDelta == 7 {
				weekdayDelta = 0
			}
			nextMon := shippingLimitDate.Add(time.Duration(24*weekdayDelta) * time.Hour)
			nextMon = time.Date(nextMon.Year(), nextMon.Month(), nextMon.Day(), 23, 59, 59, 999000000, time.UTC)

			// Simple Golang calculations, mimic script.json Golang formulas
			taxedValueDecGo := product.Price.Add(product.FreightValue).Mul(decimal.NewFromFloat(1.1)).Round(2) // r.value*decimal2(1.1)
			taxedValueByNineFloatGo, _ := taxedValueDecGo.Float64()                                            // float(r.value*decimal2(1.1))
			taxedValueByNineFloatGo = taxedValueByNineFloatGo / 9.0                                            // float(r.value*decimal2(1.1))/9.0
			taxedValueByNineDecGo := decimal.NewFromFloat(taxedValueByNineFloatGo).Round(2)                    // decimal2(float(r.value*decimal2(1.1)/9.0))

			fOutGo.WriteString(fmt.Sprintf("%s,%05d,%s,%s,%s,%s,%s,%f,%s\n",
				orderId,
				orderItemId,
				shippingLimitDate.Format("2006-01-02 15:04:05"),
				product.Price.StringFixed(2),
				product.FreightValue.StringFixed(2),
				product.Price.Add(product.FreightValue).StringFixed(2),
				taxedValueDecGo.StringFixed(2),
				taxedValueByNineFloatGo,
				taxedValueByNineDecGo.StringFixed(2)))
			fOutPy.WriteString(fmt.Sprintf("%s,%05d,%s,%s,%s,%s,%s,%f,%s,%s,%s,%t\n",
				orderId,
				orderItemId,
				shippingLimitDate.Format("2006-01-02 15:04:05"),
				product.Price.StringFixed(2),
				product.FreightValue.StringFixed(2),
				product.Price.Add(product.FreightValue).StringFixed(2),
				decimal.NewFromFloat(taxedValueFloatFlooredInPython).Round(2).StringFixed(2),
				taxedValueByNineFloatPython,
				taxedValueByNineDecPython.StringFixed(2),
				nextMon.Format("2006-01-02 15:04:05"),
				dayOfWeekName(shippingLimitDate.Weekday()),
				shippingLimitDate.Weekday() == time.Saturday || shippingLimitDate.Weekday() == time.Sunday))

			itemIdx++
			if itemIdx == *totalItems {
				break
			}
		}
	}
}
