// Package ipc is the wire protocol between the CLI and the running daemon.
// Transport is HTTP/1.1 over a Unix domain socket.
package ipc

import "github.com/huseinzol05/pm2-go/daemon/internal/process"

const (
	PathPing       = "/v1/ping"
	PathList       = "/v1/list"
	PathDescribe   = "/v1/describe"
	PathStart      = "/v1/start"
	PathStartSpec  = "/v1/start-spec"
	PathStop       = "/v1/stop"
	PathRestart    = "/v1/restart"
	PathReload     = "/v1/reload"
	PathDelete     = "/v1/delete"
	PathStopAll    = "/v1/stop-all"
	PathStartAll   = "/v1/start-all"
	PathDeleteAll  = "/v1/delete-all"
	PathSave       = "/v1/save"
	PathTail       = "/v1/logs/tail"
	PathStreamLogs = "/v1/logs/stream"
	PathShutdown   = "/v1/shutdown"
	PathSpecs      = "/v1/specs"
	PathMetrics    = "/v1/metrics"
)

type NameRequest struct {
	Name string `json:"name"`
}

type ListResponse struct {
	Procs []process.View `json:"procs"`
}

type DescribeResponse struct {
	Spec  *process.Spec  `json:"spec"`
	Procs []process.View `json:"procs"`
}

type TailResponse struct {
	Streams map[string][]string `json:"streams"` // "<runtime>:<stream>" => lines
}

type SpecsResponse struct {
	Specs []*process.Spec `json:"specs"`
}

// MetricSample is the wire shape for a single CPU/mem sample.
type MetricSample struct {
	Time int64   `json:"time"` // unix millis
	CPU  float64 `json:"cpu"`
	Mem  uint64  `json:"mem"`
}

type MetricsResponse struct {
	Series map[string][]MetricSample `json:"series"` // by runtime id
}

type ErrorResponse struct {
	Error string `json:"error"`
}
