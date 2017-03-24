package main

import (
	"encoding/json"
	"time"
	"net/http"
)

// Jobs structure for hook
type HookJob struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	Owner            string            `json:"owner"`
	StartTime        time.Time         `json:"starttime"`
	CrackedHashes    int64             `json:"crackedhashes"`
	TotalHashes      int64             `json:"totalhashes"`
	Progress         float64           `json:"progress"`
	Params           map[string]string `json:"params"`
	ToolID           string            `json:"toolid"`
	PerformanceTitle string            `json:"performancetitle"`
	PerformanceData  map[string]string `json:"performancedata"`
	OutputTitles     []string          `json:"outputtitles"`
	OutputData       [][]string        `json:"outputdata"`
}

// Resource structure to be used for hooks
type HookResource struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Manager string            `json:"manager"`
	Params  map[string]string `json:"params"`
	Status  string            `json:"status"`
	Tools   []APITool         `json:"tools"`
}

type HookQueueOrder struct {
	JobOrder []HookQueueOrderJobData `json:"orderedjobs"`
}

type HookQueueOrderJobData struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	ResourceID       string            `json:"resourceid"`
	Owner            string            `json:"owner"`
	StartTime        time.Time         `json:"starttime"`
	ETC              string            `json:"etc"`
	CrackedHashes    int64             `json:"crackedhashes"`
	TotalHashes      int64             `json:"totalhashes"`
	Progress         float64           `json:"progress"`
}