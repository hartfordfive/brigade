package main

import (
	"github.com/paulbellamy/ratecounter"
)

type HttpDirectiveStats struct {
	Url          string `json:"url"`
	Method       string `json:"method"`
	RequestTime  int64  `json:"request_time"`
	ResponseSize uint64 `json:"response_size"`
	ResponseCode int    `json:"response_code"`
}

type ScriptDirectiveStats struct {
	TotalScriptExecutions int64 `json:"total_script_executions"`
	TimeStart             int64 `json:"time_start"`
	TimeEnd               int64 `json:"time_end"`
}

type ClusterHttpDirectiveStats struct {
	TotalBytesDownloaded   uint64                       `json:"total_bytes_downloaded"`
	TotalRequestsCompleted uint64                       `json:"total_requests_completed"`
	TotalRequestsFailed    uint64                       `json:"total_requests_failed"`
	UrlRequestsCompleted   map[string]map[string]uint64 `json:"request_breakdown"`
	UrlHits                map[string]uint64            `json:"-"`
	ClusterRPS             *ratecounter.RateCounter     `json:"-"`
	ClusterFailedRPS       *ratecounter.RateCounter     `json:"-"`
	TotalRPS               int64                        `json:"global_rps"`
	TotalFailedRPS         int64                        `json:"global_rps_failed"`
	//GlobalRps              int64                        `json:"global_rps"`
}

type Worker struct {
	LastCheckIn  int64 `json:"last_checkin"`
	TotalWorkers int64 `json:"total_workers"`
}

type NodeStats struct {
	Node map[string]Worker `json:"node_list"`
}
