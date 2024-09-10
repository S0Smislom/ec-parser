package chromedputils

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

func RunWithTimeOut(ctx context.Context, timeout time.Duration, tasks chromedp.Tasks) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return tasks.Do(timeoutContext)
	}
}
