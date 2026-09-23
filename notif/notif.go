// Package notif wraps the small slice of android.app.Notification needed to
// keep a foreground service alive on Android 13.
package notif

// Importance mirrors NotificationManager.IMPORTANCE_*.
type Importance int32

const (
	ImportanceMin     Importance = 1
	ImportanceLow     Importance = 2
	ImportanceDefault Importance = 3
	ImportanceHigh    Importance = 4
)

// Channel is an android.app.NotificationChannel handle (lazily created).
type Channel interface {
	ID() string
	// Ensure creates the channel if it doesn't exist yet. Idempotent.
	// Must be called before the first Notification using this channel
	// is built.
	Ensure() error
}

// ChannelOption configures a Channel.
type ChannelOption func(*channelConfig)

func WithChannelID(id string) ChannelOption     { return func(c *channelConfig) { c.id = id } }
func WithChannelName(n string) ChannelOption    { return func(c *channelConfig) { c.name = n } }
func WithChannelDesc(d string) ChannelOption    { return func(c *channelConfig) { c.desc = d } }
func WithImportance(i Importance) ChannelOption { return func(c *channelConfig) { c.imp = i } }

type channelConfig struct {
	id   string
	name string
	desc string
	imp  Importance
}

// Notification is a built-but-not-posted android.app.Notification handle.
type Notification interface {
	// Build assembles the Notification on the JVM side. Must be called
	// from a context where the bridge is ready (after GoServiceBridge.init).
	Build() (Handle, error)
}

// Handle is an opaque, retained reference to a built android.app.Notification.
// Release when no longer needed (e.g., after replacing it via update).
type Handle interface {
	Release()
	// Reserved: internal access for service.startForeground.
	// Implementation detail; not part of the public contract.
}

// Option configures a Notification.
type Option func(*notifConfig)

func WithTitle(s string) Option        { return func(c *notifConfig) { c.title = s } }
func WithText(s string) Option         { return func(c *notifConfig) { c.text = s } }
func WithSmallIcon(resID int32) Option { return func(c *notifConfig) { c.icon = resID } }
func WithOngoing(b bool) Option        { return func(c *notifConfig) { c.ongoing = b } }

type notifConfig struct {
	channel Channel
	title   string
	text    string
	icon    int32
	ongoing bool
}
