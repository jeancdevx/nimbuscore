package workspace

import "time"

type SnapshotStatus string

const (
	SnapshotStatusPending   SnapshotStatus = "pending"
	SnapshotStatusRunning   SnapshotStatus = "running"
	SnapshotStatusCompleted SnapshotStatus = "completed"
	SnapshotStatusFailed    SnapshotStatus = "failed"
)

type SnapshotConfig struct {
	Enabled       bool   `json:"enabled"`
	Repository    string `json:"repository"`
	Password      string `json:"password"`
	S3Endpoint    string `json:"s3_endpoint,omitempty"`
	S3Bucket      string `json:"s3_bucket,omitempty"`
	S3Region      string `json:"s3_region,omitempty"`
	S3AccessKey   string `json:"s3_access_key,omitempty"`
	S3SecretKey   string `json:"s3_secret_key,omitempty"`
	ScheduleInterval string `json:"schedule_interval"` // e.g. "1h", "30m"
	RetentionDays int    `json:"retention_days"`
}

type Snapshot struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	Status      SnapshotStatus `json:"status"`
	Size        int64          `json:"size,omitempty"`
	Message     string         `json:"message,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}
