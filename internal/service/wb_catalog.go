package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"wb-parser/internal/model"
	chromedputils "wb-parser/package/chromedp_utils"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	itemPerPage = 100
)

type wbCatalogService struct{}

func NewWBCatalogService() *wbCatalogService {
	return &wbCatalogService{}
}

func (s *wbCatalogService) Parse(ctx context.Context, wbCatalogUrl string, pages int, output string) {
	cctx, cancel := chromedputils.InitChromeDPContext(ctx)
	defer cancel()
	products, err := s.parsewbCatalog(cctx, wbCatalogUrl, pages)
	if err != nil {
		fmt.Println(err)
	}
	if err := s.writeResults(ctx, products, output); err != nil {
		fmt.Println(err)
	}

}

func (s *wbCatalogService) parsewbCatalog(ctx context.Context, wbCatalogUrl string, pages int) ([]*model.ProductCard, error) {
	// // Navigate
	// if err := chromedp.Run(ctx,
	// 	chromedp.Navigate(wbCatalogUrl),
	// ); err != nil {
	// 	return nil, err
	// }
	// // Count total products to check total pages
	// total, err := s.parseTotalProducts(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	// fmt.Println("Total:", total, "Pages:", total/itemPerPage)
	// totalPages := total / itemPerPage
	totalPages := pages

	products := []*model.ProductCard{}

	for i := 1; i < totalPages; i++ {

		var pageUrl string
		if i == 1 {
			pageUrl = wbCatalogUrl
		} else if strings.Contains(wbCatalogUrl, "?") {
			pageUrl = fmt.Sprintf("%s&page=%d", wbCatalogUrl, i)
		} else {
			pageUrl = fmt.Sprintf("%s?page=%d", wbCatalogUrl, i)
		}
		fmt.Println(pageUrl)
		// Navigate
		if err := chromedp.Run(ctx,
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

func (s *wbCatalogService) parseProducts(ctx context.Context) ([]*model.ProductCard, error) {
	var productNodes []*cdp.Node

	if err := chromedp.Run(
		ctx,
		chromedp.WaitVisible(".product-card", chromedp.ByQueryAll),
		chromedp.Nodes(".product-card", &productNodes, chromedp.ByQueryAll),
	); err != nil {
		return nil, err
	}

	for {
		l := len(productNodes)
		chromedp.Run(ctx,
			chromedp.Click("#body-layout", chromedp.ByID),
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
			chromedp.WaitVisible(".product-card", chromedp.ByQueryAll),
			chromedp.Nodes(".product-card", &productNodes, chromedp.ByQueryAll),
		)
		if l == len(productNodes) {
			break
		}
	}
	fmt.Println(len(productNodes))
	products := []*model.ProductCard{}

	for _, node := range productNodes {
		var productTitle string
		// var url string
		var linkNodes []*cdp.Node
		var url string
		var fullPrice string
		var rate string
		var reviews string

		if err := chromedp.Run(ctx,
			chromedputils.RunWithTimeOut(ctx, 500*time.Millisecond, chromedp.Tasks{

				chromedp.Text(".product-card__name", &productTitle, chromedp.ByQueryAll, chromedp.FromNode(node)),
				chromedp.Text(".price__wrap", &fullPrice, chromedp.ByQueryAll, chromedp.FromNode(node)),
				chromedp.Nodes(".product-card__link", &linkNodes, chromedp.ByQueryAll, chromedp.FromNode(node)),
				chromedp.Text(".address-rate-mini", &rate, chromedp.ByQueryAll, chromedp.FromNode(node)),
				chromedp.Text(".product-card__count", &reviews, chromedp.ByQueryAll, chromedp.FromNode(node)),
			}),
		); err != nil {

			continue
		}
		url = linkNodes[0].AttributeValue("href")

		// fmt.Println(productTitle, url, fullPrice, rate, reviews)

		products = append(products, &model.ProductCard{
			Url:       url,
			Title:     s.prepareTitle(productTitle),
			FullPrice: s.prepareFullPrice(fullPrice),
			Price:     s.preparePrice(fullPrice),
			Rate:      rate,
			Reviews:   s.prepareReviews(reviews),
		})
	}
	return products, nil
}

func (s *wbCatalogService) prepareTitle(title string) string {
	title = strings.ReplaceAll(title, "/", "")
	title = strings.TrimSpace(title)
	return title
}

func (s *wbCatalogService) prepareReviews(reviews string) string {
	splited := strings.Split(reviews, " ")
	res := strings.Join(splited[:len(splited)-1], "")
	res = strings.ReplaceAll(res, "\xa0", "")
	res = strings.ToValidUTF8(res, "")
	if res == "Нет" {
		return ""
	}
	return res
}

func (s *wbCatalogService) preparePriceStr(str string) string {
	splited := strings.Split(str, " ")
	res := strings.Join(splited, "")
	res = strings.ReplaceAll(res, "\xa0", "")
	res = strings.ToValidUTF8(res, "")
	return res
}

func (s *wbCatalogService) preparePrice(fullPrice string) string {
	res := strings.Split(fullPrice, "\n")[0]
	splited := strings.Split(res, "₽")
	return s.preparePriceStr(splited[0])
}

func (s *wbCatalogService) prepareFullPrice(fullPrice string) string {
	res := strings.Split(fullPrice, "\n")[0]
	splited := strings.Split(res, "₽")
	if len(splited) < 3 {
		return s.preparePrice(fullPrice)
	}
	return s.preparePriceStr(splited[1])
}

func (s *wbCatalogService) parseTotalProducts(ctx context.Context) (int, error) {
	var totalGoods []*cdp.Node

	if err := chromedp.Run(ctx,
		chromedputils.RunWithTimeOut(ctx, 500*time.Millisecond, chromedp.Tasks{
			chromedp.WaitVisible(".goods-count", chromedp.ByQueryAll),
			chromedp.Nodes(".goods-count", &totalGoods, chromedp.ByQueryAll),
		}),
	); err != nil {
		chromedp.Run(ctx,
			chromedp.WaitVisible(".searching-results__count", chromedp.ByQueryAll),
			chromedp.Nodes(".searching-results__count", &totalGoods, chromedp.ByQueryAll),
		)
	}
	if len(totalGoods) == 0 {
		return 0, errors.New("no goods")
	}
	var total string
	chromedp.Run(ctx,
		chromedp.Text(totalGoods[0].FullXPath()+"/span", &total, chromedp.BySearch),
	)
	return s.convertTotalProducts(total)
}

func (s *wbCatalogService) convertTotalProducts(total string) (int, error) {
	sTotal := strings.Split(total, " ")
	jTotal := strings.Join(sTotal, "")
	return strconv.Atoi(jTotal)
}

func (s *wbCatalogService) writeResults(ctx context.Context, products []*model.ProductCard, output string) error {
	err := os.MkdirAll(output, os.ModePerm)
	if err != nil {
		return err
	}
	filepath := fmt.Sprintf("%s/wb-products-%s.csv", output, time.Now().Format("2006-01-02_15-04-05"))
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
