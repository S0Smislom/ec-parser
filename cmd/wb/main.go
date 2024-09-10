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
	flag.StringVar(&categoryUrl, "url", "https://www.wildberries.ru/catalog/dom/hranenie-veshchey/korobki-korzinki-keysy", "Category url")
	flag.IntVar(&pages, "pages", 30, "Max Pages")
	flag.StringVar(&output, "output", "output", "Output path")
}

func main() {
	flag.Parse()

	s := service.NewWBCatalogService()

	start := time.Now()
	s.Parse(context.TODO(), categoryUrl, pages, output)

	fmt.Println(time.Since(start).Seconds())

}
