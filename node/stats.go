package main

type HttpDirectiveStats struct {
	Url          string `json:"url"`
	Method       string `json:"method"`
	RequestTime  int64  `json:"request_time"`
	ResponseSize int    `json:"response_size"`
	ResponseCode int    `json:"response_code"`
}

type ClusterHttpDirectiveStats struct {
	TotalBytesDownloaded   uint64            `json:"total_bytes_downloaded"`
	TotalRequestsCompleted uint64            `json:"total_requests_completed"`
	UrlRequestsBreakdown   map[string]uint64 `json:"requests_breakdown"`
	GlobalRps              int64             `json:"global_rps"`
}

type SshDirectiveStats struct {
	Host        string `json:"host"`
	Credentials string `json:"credentials"`
	Command     string `json:"cmd"`
	ExitCode    int    `json:"exit_code"`
	ExecTime    int64  `json:"exec_time"`
}

/*
type Worker struct {
	LastCheckIn  int64 `json:"last_checkin"`
	TotalWorkers int64 `json:"total_workers"`
}

type NodeStats struct {
	Node map[string]Worker `json:"node_list"`
}
*/
