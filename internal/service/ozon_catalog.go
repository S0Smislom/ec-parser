package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"wb-parser/internal/model"
	chromedputils "wb-parser/package/chromedp_utils"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type ozonCatalogService struct{}

func NewOzonCatalogService() *ozonCatalogService {
	return &ozonCatalogService{}
}

func (s *ozonCatalogService) Parse(ctx context.Context, url string, pages int, output string) {
	cctx, cancel := chromedputils.InitChromeDPContext(ctx)
	defer cancel()

	products, err := s.parseCatalog(cctx, url, pages)
	if err != nil {
		fmt.Println(err)
	}
	if err := s.writeResults(ctx, products, output); err != nil {
		fmt.Println(err)
	}
}

func (s *ozonCatalogService) writeResults(ctx context.Context, products []*model.ProductCard, output string) error {
	err := os.MkdirAll(output, os.ModePerm)
	if err != nil {
		return err
	}
	filepath := fmt.Sprintf("%s/ozon-products-%s.csv", output, time.Now().Format("2006-01-02_15-04-05"))
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	csvw := csv.NewWriter(f)

	if err := csvw.Write([]string{
		"title", "url", "price", "full_price", "rate", "reviews",
	}); err != nil {
		return err
	}

	for _, product := range products {
		row := []string{
			product.Title,
			product.Url,
			product.Price,
			product.FullPrice,
			product.Rate,
			product.Reviews,
		}
		if err := csvw.Write(row); err != nil {
			return err
		}
	}
	csvw.Flush()
	if err = csvw.Error(); err != nil {
		return err
	}
	return nil
}

func (s *ozonCatalogService) parseCatalog(ctx context.Context, url string, pages int) ([]*model.ProductCard, error) {
	// // Navigate
	// if err := chromedp.Run(ctx,
	// 	chromedp.Navigate(url),
	// ); err != nil {
	// 	return nil, err
	// }
	// Count total products to check total pages
	// total, err := s.parseTotalProducts(ctx)
	// if err != nil {
	// 	return nil, err
	// }
	// totalPages := pages

	products := []*model.ProductCard{}

	for i := 1; i <= pages; i++ {
		var pageUrl string

		if strings.Contains(url, "?") {
			pageUrl = fmt.Sprintf("%s&page=%d", url, i)
		} else {
			pageUrl = fmt.Sprintf("%s?page=%d", url, i)
		}
		if err := chromedp.Run(
			ctx,
			chromedp.Navigate(pageUrl),
		); err != nil {
			return nil, err
		}
		parsedProducts, err := s.parseProducts(ctx)
		if err != nil {
			continue
		}
		products = append(products, parsedProducts...)
	}
	return products, nil
}

func (s *ozonCatalogService) parseProducts(ctx context.Context) ([]*model.ProductCard, error) {
	var productNodes []*cdp.Node

	if err := chromedp.Run(
		ctx,
		chromedp.WaitVisible(".tile-root", chromedp.ByQueryAll),
		chromedp.Nodes(".tile-root", &productNodes, chromedp.ByQueryAll),
	); err != nil {
		return nil, err
	}

	for {
		l := len(productNodes)
		chromedp.Run(ctx,
			// chromedp.Click("#paginatorContent", chromedp.ByID),
			chromedp.ActionFunc(func(ctx context.Context) error {
				_, exp, err := runtime.Evaluate(`window.scrollTo(0,document.body.scrollHeight);`).Do(ctx)
				time.Sleep(500 * time.Millisecond)
				if err != nil {
					return err
				}
				if exp != nil {
					return exp
				}
				return nil
			}),
			chromedp.WaitVisible(".tile-root", chromedp.ByQueryAll),
			chromedp.Nodes(".tile-root", &productNodes, chromedp.ByQueryAll),
		)
		if l == len(productNodes) {
			break
		}
	}
	products := []*model.ProductCard{}
	for _, node := range productNodes {
		var title string
		var linkNodes []*cdp.Node
		var url string
		var fullPrice string
		// var price string
		var rate string
		// var reviews string

		if err := chromedp.Run(ctx,
			chromedputils.RunWithTimeOut(
				ctx,
				500*time.Millisecond,
				chromedp.Tasks{
					chromedp.Text(".tsBody500Medium", &title, chromedp.ByQueryAll, chromedp.FromNode(node)),
					chromedp.Text(".tsBodyMBold", &rate, chromedp.ByQueryAll, chromedp.FromNode(node)),
					// chromedp.Text(".tsHeadline500Medium", &price, chromedp.ByQueryAll, chromedp.FromNode(node)),
					chromedp.Text(".c3011-a0", &fullPrice, chromedp.ByQueryAll, chromedp.FromNode(node)),
					chromedp.Nodes(".tile-hover-target", &linkNodes, chromedp.ByQueryAll, chromedp.FromNode(node)),
				},
			),
		); err != nil {
			fmt.Println(err)
			continue
		}
		url = linkNodes[0].AttributeValue("href")

		product := &model.ProductCard{
			Url:       s.prepareURL(url),
			Title:     title,
			FullPrice: s.prepareFullPrice(fullPrice),
			Price:     s.preparePrice(fullPrice),
			Rate:      s.prepareRate(rate),
			Reviews:   s.prepareReviews(rate),
		}
		fmt.Println(product)
		products = append(products, product)
	}
	return products, nil

}

func (s *ozonCatalogService) prepareURL(url string) string {
	return fmt.Sprintf("%s%s", "https://www.ozon.ru", url)
}
func (s *ozonCatalogService) prepareReviews(reviews string) string {
	splited := strings.Split(reviews, " ")
	res := strings.Join(splited[1:], "")

	pattern := regexp.MustCompile(`\d+`)
	digets := []string{}
	for _, match := range pattern.FindAll([]byte(res), -1) {
		digets = append(digets, string(match))
	}
	return strings.Join(digets, "")
}

func (s *ozonCatalogService) prepareRate(rate string) string {
	return strings.Split(rate, " ")[0]
}

func (s *ozonCatalogService) preparePrice(price string) string {
	temp := strings.Split(price, "\n")[0]
	temp = strings.Split(temp, "₽")[0]
	splited := strings.Split(temp, " ")
	res := strings.Join(splited, "")
	res = strings.ReplaceAll(res, "\u2009", "")
	res = strings.ToValidUTF8(res, "")
	return res
}

func (s *ozonCatalogService) prepareFullPrice(price string) string {
	temp := strings.Split(price, "\n")
	if len(temp) < 2 {
		return s.preparePrice(price)
	}
	temp = strings.Split(temp[1], "₽")
	splited := strings.Split(temp[0], " ")
	res := strings.Join(splited, "")
	res = strings.ReplaceAll(res, "\u2009", "")
	res = strings.ToValidUTF8(res, "")
	return res

}
