package chromedputils

import (
	"context"

	"github.com/chromedp/chromedp"
)

func InitChromeDPContext(ctx context.Context) (context.Context, context.CancelFunc) {
	initialCtx, _ := chromedp.NewExecAllocator(ctx, chromedp.Flag("headless", false))
	cctx, cancel := chromedp.NewContext(initialCtx) // chromedp.WithDebugf(log.Printf),
	return cctx, cancel
}
