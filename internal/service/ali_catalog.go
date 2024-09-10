package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"
	"wb-parser/internal/model"
	chromedputils "wb-parser/package/chromedp_utils"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type aliCatalgService struct{}

func NewAliCatalogService() *aliCatalgService {
	return &aliCatalgService{}
}

func (s *aliCatalgService) Parse(ctx context.Context, url string, pages int, output string) {
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

func (s *aliCatalgService) parseCatalog(ctx context.Context, url string, pages int) ([]*model.ProductCard, error) {
	products := []*model.ProductCard{}
	for i := 1; i < pages; i++ {
		pageUrl := s.generatePageUrl(url, i)

		// Navigate
		if err := chromedp.Run(ctx, chromedp.Navigate(pageUrl)); err != nil {
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

func (s *aliCatalgService) parseProducts(ctx context.Context) ([]*model.ProductCard, error) {
	var productNodes []*cdp.Node
	productCardClass := ".product-snippet_ProductSnippet__content__1mogfw"
	if err := chromedp.Run(
		ctx,
		chromedp.WaitVisible(productCardClass, chromedp.ByQueryAll),
		chromedp.Nodes(productCardClass, &productNodes, chromedp.ByQueryAll),
	); err != nil {
		return nil, err
	}

	for {
		l := len(productNodes)
		chromedp.Run(ctx,
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
			chromedp.WaitVisible(productCardClass, chromedp.ByQueryAll),
			chromedp.Nodes(productCardClass, &productNodes, chromedp.ByQueryAll),
		)
		if l == len(productNodes) {
			break
		}
	}

	fmt.Println(len(productNodes))
	products := []*model.ProductCard{}

	// TODO parse products
	for _, node := range productNodes {
		var productTitle string
		var linkNodes []*cdp.Node
		var url string
		var price string
		var rate string
		var reviews string // кол-во покупок

		if err := chromedp.Run(ctx,
			chromedputils.RunWithTimeOut(ctx, 500*time.Millisecond, chromedp.Tasks{
				chromedp.Text(".product-snippet_ProductSnippet__name__1mogfw", &productTitle, chromedp.ByQueryAll, chromedp.FromNode(node)),
				chromedp.Text(".snow-price_SnowPrice__mainM__uw8t09", &price, chromedp.ByQueryAll, chromedp.FromNode(node)),
				chromedp.Nodes(".product-snippet_ProductSnippet__galleryBlock__1mogfw", &linkNodes, chromedp.ByQueryAll, chromedp.FromNode(node)),
				// chromedp.Text("", &rate, chromedp.ByQueryAll, chromedp.FromNode(node)),
				// chromedp.Text("", &reviews, chromedp.ByQueryAll)
			}),
		); err != nil {
			continue
		}
		chromedp.Run(ctx,
			chromedputils.RunWithTimeOut(ctx, 100*time.Millisecond, chromedp.Tasks{
				chromedp.Text(".product-snippet_ProductSnippet__score__1mogfw", &rate, chromedp.ByQueryAll, chromedp.FromNode(node)),
			}),
		)
		chromedp.Run(ctx,
			chromedputils.RunWithTimeOut(ctx, 100*time.Millisecond, chromedp.Tasks{
				chromedp.Text(".product-snippet_ProductSnippet__sold__1mogfw", &reviews, chromedp.ByQueryAll, chromedp.FromNode(node)),
			}),
		)
		url = linkNodes[0].AttributeValue("href")

		fmt.Println(productTitle, url, price)
		products = append(products, &model.ProductCard{
			Url:       url,
			Title:     productTitle,
			FullPrice: price,
			Price:     price,
			Rate:      rate,
			Reviews:   reviews,
		})
	}

	return products, nil
}

func (s *aliCatalgService) writeResults(ctx context.Context, products []*model.ProductCard, output string) error {
	err := os.MkdirAll(output, os.ModePerm)
	if err != nil {
		return err
	}
	filepath := fmt.Sprintf("%s/ali-products-%s.csv", output, time.Now().Format("2006-01-02_15-04-05"))
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

func (s *aliCatalgService) generatePageUrl(url string, page int) string {
	if strings.Contains(url, "?") {
		return fmt.Sprintf("%s&page=%d", url, page)
	} else {
		return fmt.Sprintf("%s?page=%d", url, page)
	}
}
