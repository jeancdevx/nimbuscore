package workspace

type SnapshotConfig struct {
	Enabled          bool   `json:"enabled"`
	Repository       string `json:"repository"`
	Password         string `json:"password"`
	S3Endpoint       string `json:"s3_endpoint,omitempty"`
	S3Bucket         string `json:"s3_bucket,omitempty"`
	S3Region         string `json:"s3_region,omitempty"`
	S3AccessKey      string `json:"s3_access_key,omitempty"`
	S3SecretKey      string `json:"s3_secret_key,omitempty"`
	ScheduleInterval string `json:"schedule_interval"`
	RetentionDays    int    `json:"retention_days"`
}

type SnapshotEventPayload struct {
	Action   string `json:"action"`
	SnapshotID string `json:"snapshot_id,omitempty"`
}

type ScheduleSnapshotPayload struct {
	WorkspaceIDs []string `json:"workspace_ids"`
}
