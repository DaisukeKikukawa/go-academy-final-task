package opendata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DefaultCKANBaseURL は東京都オープンデータカタログの API ベース URL。
const DefaultCKANBaseURL = "https://catalog.data.metro.tokyo.lg.jp/api/3/action"

// CKANClient はカタログサイト(CKAN)の API を叩く。
type CKANClient struct {
	BaseURL string
	HTTP    *http.Client
}

// NewCKANClient は東京都カタログ向けのクライアントを返す。
func NewCKANClient(timeout time.Duration) *CKANClient {
	return &CKANClient{BaseURL: DefaultCKANBaseURL, HTTP: &http.Client{Timeout: timeout}}
}

// PackageInfo は package_show の結果のうち、更新検知に使う部分。
type PackageInfo struct {
	ID               string
	MetadataModified time.Time // データセットの最終更新(UTC)
	Resources        []PackageResource
}

// PackageResource はデータセットに含まれるファイル1つ。
type PackageResource struct {
	URL              string
	Format           string
	MetadataModified time.Time
}

// package_show のレスポンスの、必要な部分だけを受ける構造体。
type packageShowResponse struct {
	Success bool `json:"success"`
	Result  struct {
		ID               string `json:"id"`
		MetadataModified string `json:"metadata_modified"`
		Resources        []struct {
			URL              string `json:"url"`
			Format           string `json:"format"`
			MetadataModified string `json:"metadata_modified"`
		} `json:"resources"`
	} `json:"result"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// PackageShow はデータセット ID(例: t131091d0000000007)の情報を取る。
func (c *CKANClient) PackageShow(ctx context.Context, packageID string) (*PackageInfo, error) {
	u := fmt.Sprintf("%s/package_show?id=%s", c.BaseURL, url.QueryEscape(packageID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gomi-api-importer/0.1")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CKAN package_show: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CKAN package_show: HTTP %d", resp.StatusCode)
	}

	var body packageShowResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("CKAN package_show: JSON を読めません: %w", err)
	}
	if !body.Success {
		return nil, fmt.Errorf("CKAN package_show: %s", body.Error.Message)
	}

	modified, err := parseCKANTime(body.Result.MetadataModified)
	if err != nil {
		return nil, err
	}
	info := &PackageInfo{ID: body.Result.ID, MetadataModified: modified}
	for _, r := range body.Result.Resources {
		t, _ := parseCKANTime(r.MetadataModified) // リソース側は無くても致命的ではない
		info.Resources = append(info.Resources, PackageResource{URL: r.URL, Format: r.Format, MetadataModified: t})
	}
	return info, nil
}

// parseCKANTime は "2025-12-12T08:22:56.376076" のような、タイムゾーン無し(UTC)の文字列を読む。
func parseCKANTime(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02T15:04:05.999999", s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("CKAN の日時 %q を読めません: %w", s, err)
	}
	return t, nil
}
