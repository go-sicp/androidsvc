package log_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	asvclog "github.com/go-sicp/androidsvc/log"
)

func TestBufferRing(t *testing.T) {
	b := asvclog.NewBuffer(3)
	for i := 0; i < 5; i++ {
		b.Write(entry(i))
	}
	if got, want := b.Len(), 3; got != want {
		t.Fatalf("Len: got %d want %d", got, want)
	}
	snap := b.Snapshot(0)
	if len(snap) != 3 {
		t.Fatalf("Snapshot: %d", len(snap))
	}
	if snap[0].Message() != "msg-2" || snap[2].Message() != "msg-4" {
		t.Fatalf("ring eviction wrong: %v", snap)
	}
}

func TestBufferSubscribe(t *testing.T) {
	b := asvclog.NewBuffer(10)
	ch, cancel := b.Subscribe(4)
	defer cancel()
	go func() {
		for i := 0; i < 3; i++ {
			b.Write(entry(i))
		}
	}()

	got := 0
	timeout := time.After(2 * time.Second)
	for got < 3 {
		select {
		case <-ch:
			got++
		case <-timeout:
			t.Fatalf("only received %d entries", got)
		}
	}
}

func TestSubscribeCancelClosesChannel(t *testing.T) {
	b := asvclog.NewBuffer(2)
	ch, cancel := b.Subscribe(1)
	cancel()
	cancel() // idempotent
	if _, ok := <-ch; ok {
		t.Fatal("channel should be closed after cancel")
	}
}

func TestHandlerWritesIntoBuffer(t *testing.T) {
	b := asvclog.NewBuffer(8)
	h := asvclog.NewHandler(b, slog.LevelDebug)
	logger := slog.New(h)
	logger.Info("hello", "k", "v")
	logger.Warn("watch", "n", 42)

	snap := b.Snapshot(0)
	if len(snap) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(snap))
	}
	if snap[0].Message() != "hello" || snap[1].Message() != "watch" {
		t.Fatalf("messages: %+v", snap)
	}
	if snap[0].Fields()["k"] != "v" {
		t.Fatalf("attrs lost: %+v", snap[0].Fields())
	}
	_ = context.Background()
}

func TestHandlerLevelGate(t *testing.T) {
	b := asvclog.NewBuffer(4)
	h := asvclog.NewHandler(b, slog.LevelWarn)
	logger := slog.New(h)
	logger.Info("muted")
	logger.Error("kept")
	if got := b.Len(); got != 1 {
		t.Fatalf("level gate: got %d entries, want 1", got)
	}
	if b.Snapshot(0)[0].Message() != "kept" {
		t.Fatalf("kept the wrong entry")
	}
}

// entry builds an Entry via slog round-trip so we exercise the public API.
func entry(i int) asvclog.Entry {
	b := asvclog.NewBuffer(1)
	logger := slog.New(asvclog.NewHandler(b, slog.LevelDebug))
	logger.Info("msg-" + string(rune('0'+i)))
	return b.Snapshot(0)[0]
}
