package mediamtx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a typed HTTP client for the mediamtx v3 API.
type Client struct {
	base    string
	apiUser string
	apiPass string
	apiKey  string
	http    *http.Client
}

func NewClient(apiAddress, apiUser, apiPass, apiKey string) *Client {
	return &Client{
		base:    apiAddress,
		apiUser: apiUser,
		apiPass: apiPass,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Ping checks connectivity by fetching the global config.
func (c *Client) Ping() error {
	_, err := c.GetGlobalConfig()
	return err
}

// GetGlobalConfig fetches the mediamtx global configuration.
func (c *Client) GetGlobalConfig() (*GlobalConfig, error) {
	var cfg GlobalConfig
	if err := c.get("/v3/config/global/get", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ListPaths returns all currently active paths.
func (c *Client) ListPaths(page, itemsPerPage int) (*PagedResponse[PathItem], error) {
	var resp PagedResponse[PathItem]
	endpoint := fmt.Sprintf("/v3/paths/list?page=%d&itemsPerPage=%d", page, itemsPerPage)
	if err := c.get(endpoint, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAllPaths pages through all paths automatically.
func (c *Client) ListAllPaths() ([]PathItem, error) {
	const perPage = 100
	var all []PathItem
	for page := 0; ; page++ {
		resp, err := c.ListPaths(page, perPage)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Items...)
		if len(all) >= resp.ItemCount {
			break
		}
	}
	return all, nil
}

// ListConfigPaths returns all configured (not necessarily active) paths.
func (c *Client) ListConfigPaths(page, itemsPerPage int) (*PagedResponse[PathConfig], error) {
	var resp PagedResponse[PathConfig]
	endpoint := fmt.Sprintf("/v3/config/paths/list?page=%d&itemsPerPage=%d", page, itemsPerPage)
	if err := c.get(endpoint, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListAllConfigPaths() ([]PathConfig, error) {
	const perPage = 100
	var all []PathConfig
	for page := 0; ; page++ {
		resp, err := c.ListConfigPaths(page, perPage)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Items...)
		if len(all) >= resp.ItemCount {
			break
		}
	}
	return all, nil
}

func (c *Client) GetConfigPath(name string) (*PathConfig, error) {
	var p PathConfig
	if err := c.get("/v3/config/paths/get/"+url.PathEscape(name), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) AddConfigPath(name string, cfg PathConfig) error {
	return c.post("/v3/config/paths/add/"+url.PathEscape(name), cfg, nil)
}

func (c *Client) PatchConfigPath(name string, cfg PathConfig) error {
	return c.patch("/v3/config/paths/patch/"+url.PathEscape(name), cfg, nil)
}

func (c *Client) DeleteConfigPath(name string) error {
	return c.delete("/v3/config/paths/delete/" + url.PathEscape(name))
}

// ListRTSPConns returns active RTSP connections.
func (c *Client) ListRTSPConns() (*PagedResponse[RTSPConn], error) {
	var resp PagedResponse[RTSPConn]
	if err := c.get("/v3/rtspconns/list", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListWebRTCSessions returns active WebRTC sessions.
func (c *Client) ListWebRTCSessions() (*PagedResponse[WebRTCSession], error) {
	var resp PagedResponse[WebRTCSession]
	if err := c.get("/v3/webrtcsessions/list", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- HTTP helpers ---

func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) post(path string, body, out any) error {
	return c.sendJSON(http.MethodPost, path, body, out)
}

func (c *Client) patch(path string, body, out any) error {
	return c.sendJSON(http.MethodPatch, path, body, out)
}

func (c *Client) delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.base+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) sendJSON(method, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	if c.apiUser != "" {
		req.SetBasicAuth(c.apiUser, c.apiPass)
	} else if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("mediamtx api %s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("mediamtx api %s %s: status %d: %s", req.Method, req.URL.Path, resp.StatusCode, body)
	}

	if out != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("mediamtx api decode: %w", err)
		}
	}
	return nil
}
