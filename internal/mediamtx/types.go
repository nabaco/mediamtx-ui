package mediamtx

import "time"

// PagedResponse is the common envelope for paginated mediamtx API responses.
type PagedResponse[T any] struct {
	ItemCount int `json:"itemCount"`
	PageCount int `json:"pageCount"`
	Items     []T `json:"items"`
}

// PathItem is a currently-active path (stream) from GET /v3/paths/list.
type PathItem struct {
	Name          string     `json:"name"`
	ConfName      string     `json:"confName"`
	Source        *Source    `json:"source"`
	Ready         bool       `json:"ready"`
	ReadyTime     *time.Time `json:"readyTime"`
	Tracks        []string   `json:"tracks"`
	BytesReceived uint64     `json:"bytesReceived"`
	BytesSent     uint64     `json:"bytesSent"`
	Readers       []Reader   `json:"readers"`
}

type Source struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Reader struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// PathConfig represents a configured path from GET /v3/config/paths/get/{name}.
// Only the fields we expose in the UI are listed; the rest pass through as RawConfig.
type PathConfig struct {
	Name           string `json:"name"`
	Source         string `json:"source"`
	SourceOnDemand bool   `json:"sourceOnDemand"`
	Record         bool   `json:"record"`
	// MaxReaders limits simultaneous readers (0 = unlimited)
	MaxReaders int `json:"maxReaders"`
	// RunOnReady / RunOnNotReady are shell commands mediamtx executes on state change
	RunOnReady    string `json:"runOnReady"`
	RunOnNotReady string `json:"runOnNotReady"`
}

// GlobalConfig is the response from GET /v3/config/global/get.
// We only decode the fields we display; the full YAML is shown raw.
type GlobalConfig struct {
	LogLevel          string `json:"logLevel"`
	LogDestinations   []string `json:"logDestinations"`
	LogFile           string `json:"logFile"`
	ReadTimeout       string `json:"readTimeout"`
	WriteTimeout      string `json:"writeTimeout"`
	WriteQueueSize    int    `json:"writeQueueSize"`
	UDPMaxPayloadSize int    `json:"udpMaxPayloadSize"`
	// API
	API        bool   `json:"api"`
	APIAddress string `json:"apiAddress"`
	// RTSP
	RTSP        bool   `json:"rtsp"`
	RTSPAddress string `json:"rtspAddress"`
	// HLS
	HLS        bool   `json:"hls"`
	HLSAddress string `json:"hlsAddress"`
	// WebRTC
	WebRTC        bool   `json:"webrtc"`
	WebRTCAddress string `json:"webrtcAddress"`
	// SRT
	SRT        bool   `json:"srt"`
	SRTAddress string `json:"srtAddress"`
	// RTMP
	RTMP        bool   `json:"rtmp"`
	RTMPAddress string `json:"rtmpAddress"`
	// Auth
	Auth AuthConfig `json:"auth"`
}

type AuthConfig struct {
	Method      string `json:"method"`
	HTTPAddress string `json:"httpAddress"`
}

// RTSPConn is one active RTSP connection.
type RTSPConn struct {
	ID         string    `json:"id"`
	Created    time.Time `json:"created"`
	RemoteAddr string    `json:"remoteAddr"`
	BytesReceived uint64 `json:"bytesReceived"`
	BytesSent     uint64 `json:"bytesSent"`
}

// WebRTCSession is one active WebRTC session.
type WebRTCSession struct {
	ID         string    `json:"id"`
	Created    time.Time `json:"created"`
	RemoteAddr string    `json:"remoteAddr"`
	PeerConnectionState string `json:"peerConnectionState"`
	Path       string    `json:"path"`
	BytesReceived uint64 `json:"bytesReceived"`
	BytesSent     uint64 `json:"bytesSent"`
}
