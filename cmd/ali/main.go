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
	flag.StringVar(&categoryUrl, "url", "https://aliexpress.ru/category/22/electronic-components-supplies?spm=a2g2w.home.0.0.75df5586E01UcQ&source=nav_category", "Category url")
	flag.IntVar(&pages, "pages", 30, "Max Pages")
	flag.StringVar(&output, "output", "output", "Output path")
}

func main() {
	flag.Parse()

	s := service.NewAliCatalogService()

	start := time.Now()
	s.Parse(context.TODO(), categoryUrl, pages, output)

	fmt.Println(time.Since(start).Seconds())

}
