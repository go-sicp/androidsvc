// Package log provides a thread-safe, subscribable ring buffer of structured
// log entries, plumbed in as a log/slog handler. The intended use is: the Go
// service writes via slog.Info / slog.Error; the UI subscribes to the buffer
// to render a live log view.
package log

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Entry is one captured log line. Fields are unexported; access via the
// methods below so the on-disk shape can evolve without breaking callers.
type Entry struct {
	t      time.Time
	level  slog.Level
	msg    string
	fields map[string]any
}

func (e Entry) Time() time.Time        { return e.t }
func (e Entry) Level() slog.Level      { return e.level }
func (e Entry) Message() string        { return e.msg }
func (e Entry) Fields() map[string]any { return e.fields }

// Buffer is a fixed-capacity, thread-safe FIFO of Entry. Once full, oldest
// entries are evicted as new ones arrive.
type Buffer interface {
	Write(Entry)
	// Subscribe returns a channel that receives every entry written after
	// the call. The cancel func detaches the subscriber and closes the
	// channel; it is safe to call multiple times.
	Subscribe(buf int) (<-chan Entry, func())
	Snapshot(n int) []Entry
	Cap() int
	Len() int
}

// NewBuffer returns a Buffer that retains up to cap entries.
func NewBuffer(cap int) Buffer {
	if cap < 1 {
		cap = 1
	}
	return &buffer{ring: make([]Entry, 0, cap), cap: cap}
}

type buffer struct {
	mu    sync.Mutex
	ring  []Entry
	cap   int
	subs  map[int]chan Entry
	nextS int
}

func (b *buffer) Cap() int { return b.cap }

func (b *buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.ring)
}

func (b *buffer) Write(e Entry) {
	b.mu.Lock()
	if len(b.ring) == b.cap {
		copy(b.ring, b.ring[1:])
		b.ring[b.cap-1] = e
	} else {
		b.ring = append(b.ring, e)
	}
	subs := make([]chan Entry, 0, len(b.subs))
	for _, c := range b.subs {
		subs = append(subs, c)
	}
	b.mu.Unlock()
	for _, c := range subs {
		select {
		case c <- e:
		default:
			// Subscriber is slow; drop. The buffer always retains
			// the entry for late Snapshot() calls.
		}
	}
}

func (b *buffer) Subscribe(bufSize int) (<-chan Entry, func()) {
	if bufSize < 1 {
		bufSize = 64
	}
	ch := make(chan Entry, bufSize)
	b.mu.Lock()
	if b.subs == nil {
		b.subs = make(map[int]chan Entry)
	}
	id := b.nextS
	b.nextS++
	b.subs[id] = ch
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs, id)
			b.mu.Unlock()
			close(ch)
		})
	}
	return ch, cancel
}

func (b *buffer) Snapshot(n int) []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	if n <= 0 || n > len(b.ring) {
		n = len(b.ring)
	}
	out := make([]Entry, n)
	copy(out, b.ring[len(b.ring)-n:])
	return out
}

// Handler is a slog.Handler that writes records into a Buffer.
type Handler struct {
	buf   Buffer
	level slog.Leveler
	attrs []slog.Attr
	group string
}

// NewHandler returns a slog.Handler that writes into buf at minLevel and above.
func NewHandler(buf Buffer, minLevel slog.Leveler) *Handler {
	if minLevel == nil {
		minLevel = slog.LevelInfo
	}
	return &Handler{buf: buf, level: minLevel}
}

func (h *Handler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level.Level()
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	fields := make(map[string]any, r.NumAttrs()+len(h.attrs))
	for _, a := range h.attrs {
		fields[h.qualify(a.Key)] = a.Value.Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		fields[h.qualify(a.Key)] = a.Value.Any()
		return true
	})
	h.buf.Write(Entry{
		t:      r.Time,
		level:  r.Level,
		msg:    r.Message,
		fields: fields,
	})
	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &cp
}

func (h *Handler) WithGroup(name string) slog.Handler {
	cp := *h
	if h.group == "" {
		cp.group = name
	} else {
		cp.group = h.group + "." + name
	}
	return &cp
}

func (h *Handler) qualify(key string) string {
	if h.group == "" {
		return key
	}
	return h.group + "." + key
}
