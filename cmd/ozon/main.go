package main

import (
	"context"
	"flag"
	"fmt"
	"time"
	"wb-parser/internal/service"
)

var (
	categoryUrl string
	pages       int
	output      string
)

func init() {
	flag.StringVar(&categoryUrl, "url", "https://www.ozon.ru/category/shvabry-14618/?text=%D1%88%D0%B2%D0%B0%D0%B1%D1%80%D0%B0", "Category url")
	flag.IntVar(&pages, "pages", 30, "Max Pages")
	flag.StringVar(&output, "output", "output", "Output path")
}

func main() {
	flag.Parse()

	s := service.NewOzonCatalogService()

	start := time.Now()
	s.Parse(context.TODO(), categoryUrl, pages, output)

	fmt.Println(time.Since(start).Seconds())

}
