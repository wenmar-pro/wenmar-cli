package exports

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

const defaultMaxInlineBytes = 262144 // 256 KiB; purely informational for the CLI.

// Client makes raw HTTP requests to the unified /exports API.
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	Token      string
	LocationID string
}

// NewClient builds a Client. If httpClient is nil, a 30s default is used.
func NewClient(baseURL, token, locationID string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		HTTPClient: httpClient,
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		Token:      token,
		LocationID: locationID,
	}
}

// Schema fetches GET /exports/schema.json and returns the parsed schema.
func (c *Client) Schema(ctx context.Context) (*SchemaResponse, error) {
	u := c.BaseURL + "/exports/schema.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read schema response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, wenmar.ParseErrorBodyWithRequest(body, resp.StatusCode, http.MethodGet, "/exports/schema.json")
	}

	var schema SchemaResponse
	if err := json.Unmarshal(body, &schema); err != nil {
		return nil, fmt.Errorf("invalid schema response: %w", err)
	}
	return &schema, nil
}

// Create posts to /exports.json and returns the parsed response.
func (c *Client) Create(ctx context.Context, r CreateRequest) (*CreateResponse, error) {
	payload, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	u := c.BaseURL + "/exports.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read export response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, wenmar.ParseErrorBodyWithRequest(body, resp.StatusCode, http.MethodPost, "/exports.json")
	}

	var created CreateResponse
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("invalid export response: %w", err)
	}
	return &created, nil
}

// Download fetches the export file at downloadURL. It follows the 202 polling
// contract, sleeping for retryAfter seconds (default 2) between requests.
// On 200 it returns the body, filename from Content-Disposition, and content type.
// On 410 it returns an *wenmar.APIError wrapping the server's error message.
// maxWait limits total polling time; zero means 5 minutes.
func (c *Client) Download(ctx context.Context, downloadURL string, maxWait time.Duration) (data []byte, filename, contentType string, err error) {
	if maxWait == 0 {
		maxWait = 5 * time.Minute
	}
	deadline := time.Now().Add(maxWait)

	for {
		if time.Now().After(deadline) {
			return nil, "", "", fmt.Errorf("export download timed out after %s", maxWait)
		}

		u := c.resolveURL(downloadURL)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, "", "", err
		}
		c.setHeaders(req)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, "", "", err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, "", "", fmt.Errorf("read download response: %w", err)
		}

		switch resp.StatusCode {
		case http.StatusOK:
			return body, filenameFromHeader(resp.Header.Get("Content-Disposition")), resp.Header.Get("Content-Type"), nil
		case http.StatusAccepted:
			var pending DownloadPendingResponse
			_ = json.Unmarshal(body, &pending)
			retry := pending.RetryAfter
			if retry <= 0 {
				retry = 2
			}
			select {
			case <-ctx.Done():
				return nil, "", "", ctx.Err()
			case <-time.After(time.Duration(retry) * time.Second):
				continue
			}
		case http.StatusGone:
			return nil, "", "", wenmar.ParseErrorBodyWithRequest(body, resp.StatusCode, http.MethodGet, downloadURL)
		default:
			return nil, "", "", wenmar.ParseErrorBodyWithRequest(body, resp.StatusCode, http.MethodGet, downloadURL)
		}
	}
}

// DownloadInline decodes base64 data from Create when status is complete and data is present.
func DownloadInline(resp *CreateResponse) ([]byte, error) {
	if resp.Data == "" {
		return nil, fmt.Errorf("inline export requested but no data returned")
	}
	return base64.StdEncoding.DecodeString(resp.Data)
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if c.LocationID != "" {
		req.Header.Set("X-Wenmar-Location", c.LocationID)
	}
	req.Header.Set("Accept", "application/json")
}

func (c *Client) resolveURL(downloadURL string) string {
	if strings.HasPrefix(downloadURL, "http://") || strings.HasPrefix(downloadURL, "https://") {
		return downloadURL
	}
	return c.BaseURL + downloadURL
}

func filenameFromHeader(cd string) string {
	for _, part := range strings.Split(cd, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "filename=") {
			return strings.Trim(part[len("filename="):], `"`)
		}
		if strings.HasPrefix(part, "filename*=UTF-8'") {
			s := part[len("filename*=UTF-8'")+1:]
			if idx := strings.Index(s, "'"); idx >= 0 {
				s = s[idx+1:]
			}
			unescaped, _ := url.QueryUnescape(s)
			return unescaped
		}
	}
	return ""
}
