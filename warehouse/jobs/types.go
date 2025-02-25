package jobs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rudderlabs/rudder-server/services/notifier"

	sqlmw "github.com/rudderlabs/rudder-server/warehouse/integrations/middleware/sqlquerywrapper"

	"github.com/rudderlabs/rudder-go-kit/logger"
)

// StartJobReqPayload For processing requests payload in handlers.go
type StartJobReqPayload struct {
	SourceID      string `json:"source_id"`
	Type          string `json:"type"`
	Channel       string `json:"channel"`
	DestinationID string `json:"destination_id"`
	StartTime     string `json:"start_time"`
	JobRunID      string `json:"job_run_id"`
	TaskRunID     string `json:"task_run_id"`
	AsyncJobType  string `json:"async_job_type"`
	WorkspaceID   string `json:"workspace_id"`
}

type AsyncJobWh struct {
	db                    *sqlmw.DB
	enabled               bool
	notifier              *notifier.Notifier
	context               context.Context
	logger                logger.Logger
	maxBatchSizeToProcess int
	maxCleanUpRetries     int
	maxQueryRetries       int
	retryTimeInterval     time.Duration
	maxAttemptsPerJob     int
	asyncJobTimeOut       time.Duration
}

type WhJobsMetaData struct {
	JobRunID  string `json:"job_run_id"`
	TaskRunID string `json:"task_run_id"`
	JobType   string `json:"jobtype"`
	StartTime string `json:"start_time"`
}

// AsyncJobPayload For creating job payload to wh_async_jobs table
type AsyncJobPayload struct {
	Id            string          `json:"id"`
	SourceID      string          `json:"source_id"`
	DestinationID string          `json:"destination_id"`
	TableName     string          `json:"tablename"`
	AsyncJobType  string          `json:"async_job_type"`
	WorkspaceID   string          `json:"workspace_id"`
	MetaData      json.RawMessage `json:"metadata"`
}

const (
	WhJobWaiting   string = "waiting"
	WhJobExecuting string = "executing"
	WhJobSucceeded string = "succeeded"
	WhJobAborted   string = "aborted"
	WhJobFailed    string = "failed"
)

type NotifierResponse struct {
	Id string `json:"id"`
}

type WhStatusResponse struct {
	Status string
	Err    string
}

type WhAsyncJobRunner interface {
	startAsyncJobRunner(context.Context)
	getTableNamesBy(context.Context, string, string)
	getPendingAsyncJobs(context.Context) ([]AsyncJobPayload, error)
	getStatusAsyncJob(*StartJobReqPayload) (string, error)
	updateMultipleAsyncJobs(*[]AsyncJobPayload, string, string)
}

type AsyncJobStatus struct {
	Id     string
	Status string
	Error  error
}
