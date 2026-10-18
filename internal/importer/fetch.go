package importer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// HTTPFetcher は URL から HTTP GET で本文を取る。
func HTTPFetcher(timeout time.Duration) Fetcher {
	client := &http.Client{Timeout: timeout}
	return func(ctx context.Context, url string) (io.ReadCloser, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "gomi-api-importer/0.1")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return resp.Body, nil
	}
}

// FileFetcher は URL を無視して、指定したローカルファイルを返す(開発・テスト用)。
func FileFetcher(path string) Fetcher {
	return func(ctx context.Context, _ string) (io.ReadCloser, error) {
		return os.Open(path)
	}
}
