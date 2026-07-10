package middleware

import (
	"expvar"
	"net/http"
	"sync/atomic"
)

var (
	RequestsTotal    atomic.Int64
	ActiveWorkspaces atomic.Int64
	WorkspaceHours   atomic.Int64
	SnapshotsTotal   atomic.Int64
	PrebuildsTotal   atomic.Int64
	ErrorsTotal      atomic.Int64
)

func init() {
	expvar.Publish("nimbuscore_requests_total", expvar.Func(func() any { return RequestsTotal.Load() }))
	expvar.Publish("nimbuscore_active_workspaces", expvar.Func(func() any { return ActiveWorkspaces.Load() }))
	expvar.Publish("nimbuscore_workspace_hours_total", expvar.Func(func() any { return WorkspaceHours.Load() }))
	expvar.Publish("nimbuscore_snapshots_total", expvar.Func(func() any { return SnapshotsTotal.Load() }))
	expvar.Publish("nimbuscore_prebuilds_total", expvar.Func(func() any { return PrebuildsTotal.Load() }))
	expvar.Publish("nimbuscore_errors_total", expvar.Func(func() any { return ErrorsTotal.Load() }))
}

func IncRequestsTotal()     { RequestsTotal.Add(1) }
func IncActiveWorkspaces()  { ActiveWorkspaces.Add(1) }
func DecActiveWorkspaces()  { ActiveWorkspaces.Add(-1) }
func AddWorkspaceHours(h int64) { WorkspaceHours.Add(h) }
func IncSnapshotsTotal()    { SnapshotsTotal.Add(1) }
func IncPrebuildsTotal()    { PrebuildsTotal.Add(1) }
func IncErrorsTotal()       { ErrorsTotal.Add(1) }

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		IncRequestsTotal()
	})
}
