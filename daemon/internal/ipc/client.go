package ipc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/paths"
	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

type Client struct {
	hc       *http.Client
	endpoint string
}

func NewClient() *Client {
	sock := paths.RPCSocket()
	tr := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", sock, 2*time.Second)
		},
	}
	return &Client{
		hc:       &http.Client{Transport: tr, Timeout: 0},
		endpoint: "http://unix",
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, *http.Response, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, rd)
	if err != nil {
		return nil, nil, err
	}
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if resp.Body == nil {
		return nil, resp, nil
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}
	if resp.StatusCode >= 400 {
		var e ErrorResponse
		if json.Unmarshal(out, &e) == nil && e.Error != "" {
			return out, resp, fmt.Errorf("%s", e.Error)
		}
		return out, resp, fmt.Errorf("daemon: %s", resp.Status)
	}
	return out, resp, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, _, err := c.do(ctx, http.MethodGet, PathPing, nil)
	return err
}

func (c *Client) List(ctx context.Context) ([]process.View, error) {
	b, _, err := c.do(ctx, http.MethodGet, PathList, nil)
	if err != nil {
		return nil, err
	}
	var r ListResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return r.Procs, nil
}

func (c *Client) Specs(ctx context.Context) ([]*process.Spec, error) {
	b, _, err := c.do(ctx, http.MethodGet, PathSpecs, nil)
	if err != nil {
		return nil, err
	}
	var r SpecsResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return r.Specs, nil
}

func (c *Client) Describe(ctx context.Context, name string) (*process.Spec, []process.View, error) {
	u := PathDescribe + "?" + url.Values{"name": {name}}.Encode()
	b, _, err := c.do(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	var r DescribeResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, nil, err
	}
	return r.Spec, r.Procs, nil
}

func (c *Client) StartSpec(ctx context.Context, spec *process.Spec) error {
	_, _, err := c.do(ctx, http.MethodPost, PathStartSpec, spec)
	return err
}

func (c *Client) Start(ctx context.Context, name string) error {
	_, _, err := c.do(ctx, http.MethodPost, PathStart, NameRequest{Name: name})
	return err
}
func (c *Client) Stop(ctx context.Context, name string) error {
	_, _, err := c.do(ctx, http.MethodPost, PathStop, NameRequest{Name: name})
	return err
}
func (c *Client) Restart(ctx context.Context, name string) error {
	_, _, err := c.do(ctx, http.MethodPost, PathRestart, NameRequest{Name: name})
	return err
}
func (c *Client) Reload(ctx context.Context, name string) error {
	_, _, err := c.do(ctx, http.MethodPost, PathReload, NameRequest{Name: name})
	return err
}
func (c *Client) Delete(ctx context.Context, name string) error {
	_, _, err := c.do(ctx, http.MethodPost, PathDelete, NameRequest{Name: name})
	return err
}
func (c *Client) StopAll(ctx context.Context) error {
	_, _, err := c.do(ctx, http.MethodPost, PathStopAll, nil)
	return err
}
func (c *Client) StartAll(ctx context.Context) error {
	_, _, err := c.do(ctx, http.MethodPost, PathStartAll, nil)
	return err
}
func (c *Client) DeleteAll(ctx context.Context) error {
	_, _, err := c.do(ctx, http.MethodPost, PathDeleteAll, nil)
	return err
}
func (c *Client) Save(ctx context.Context) error {
	_, _, err := c.do(ctx, http.MethodPost, PathSave, nil)
	return err
}
func (c *Client) Shutdown(ctx context.Context) error {
	_, _, err := c.do(ctx, http.MethodPost, PathShutdown, nil)
	return err
}

func (c *Client) Tail(ctx context.Context, name, stream string, n int) (map[string][]string, error) {
	u := PathTail + "?" + url.Values{"name": {name}, "stream": {stream}, "n": {strconv.Itoa(n)}}.Encode()
	b, _, err := c.do(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var r TailResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return r.Streams, nil
}

// StreamLogs opens an ndjson stream of log lines. The returned reader yields one
// json-encoded line per record; the caller must close it.
func (c *Client) StreamLogs(ctx context.Context, name string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+PathStreamLogs+"?"+url.Values{"name": {name}}.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("daemon: %s: %s", resp.Status, string(b))
	}
	return resp.Body, nil
}
