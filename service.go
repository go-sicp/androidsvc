package androidsvc

import (
	asvclog "github.com/go-sicp/androidsvc/log"
	"github.com/go-sicp/androidsvc/notif"
)

// Service is the Go-side controller of a single foreground Android service.
// One process owns at most one Service instance, registered via Register so
// that the Java GoForegroundService callbacks can find it.
type Service interface {
	// Logs returns the live ring buffer of structured log entries.
	// Use log.NewHandler(svc.Logs(), level) to plug it into slog.
	Logs() asvclog.Buffer

	// Start asks Android to start GoForegroundService and, once Android
	// has created it, posts the foreground notification built from n.
	//
	// If n is nil, a notification is built from the options passed at
	// construction (WithNotificationContent + WithChannel).
	Start(n notif.Notification) error

	// Stop stops the foreground service.
	Stop() error

	// UpdateNotification re-posts the foreground notification with new
	// content. The service must already be running.
	UpdateNotification(n notif.Notification) error
}

// New returns a new Service. It is not yet started; call Register and Start.
func New(opts ...Option) Service {
	cfg := serviceConfig{
		logCap:  2000,
		notifID: 1,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return newService(cfg)
}

// Register makes s the process-wide Service that the Java
// GoForegroundService companion will dispatch lifecycle events to. Call this
// once at app init, before the Service can be started.
//
// Passing nil clears the registration.
func Register(s Service) { registerService(s) }
