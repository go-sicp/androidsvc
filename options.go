package androidsvc

import (
	"github.com/go-sicp/androidsvc/notif"
)

// Option configures a Service constructed via New.
type Option func(*serviceConfig)

type serviceConfig struct {
	channel    notif.Channel
	logCap     int
	notifID    int32
	notifTitle string
	notifText  string
	notifIcon  int32
	fgType     int32 // ServiceInfo.FOREGROUND_SERVICE_TYPE_*
}

// WithChannel sets the NotificationChannel used by the foreground notification.
func WithChannel(c notif.Channel) Option { return func(s *serviceConfig) { s.channel = c } }

// WithLogCapacity sets the in-memory log ring buffer size.
func WithLogCapacity(n int) Option { return func(s *serviceConfig) { s.logCap = n } }

// WithNotificationID sets the ID passed to startForeground. Default: 1.
func WithNotificationID(id int32) Option { return func(s *serviceConfig) { s.notifID = id } }

// WithNotificationContent presets the title/text/icon used when Start is
// called without an explicit notif.Notification.
func WithNotificationContent(title, text string, icon int32) Option {
	return func(s *serviceConfig) {
		s.notifTitle, s.notifText, s.notifIcon = title, text, icon
	}
}

// WithForegroundServiceType maps to ServiceInfo.FOREGROUND_SERVICE_TYPE_*.
// Recommended on Android 13, mandatory on Android 14+. Default: 0 (none).
//
// Common values:
//
//	1   FOREGROUND_SERVICE_TYPE_DATA_SYNC
//	2   FOREGROUND_SERVICE_TYPE_MEDIA_PLAYBACK
//	8   FOREGROUND_SERVICE_TYPE_LOCATION
//	16  FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE
//	64  FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION
func WithForegroundServiceType(t int32) Option {
	return func(s *serviceConfig) { s.fgType = t }
}
