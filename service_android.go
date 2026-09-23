//go:build android

package androidsvc

import (
	"errors"
	"sync"

	"github.com/go-sicp/androidsvc/internal/jni"
	asvclog "github.com/go-sicp/androidsvc/log"
	"github.com/go-sicp/androidsvc/notif"
)

const (
	startSticky    int32 = 1
	startNotSticky int32 = 2
)

var (
	regMu  sync.RWMutex
	regSvc *service
)

func registerService(s Service) {
	regMu.Lock()
	defer regMu.Unlock()
	if s == nil {
		regSvc = nil
		jni.SetServiceCallbacks(nil, nil, nil)
		return
	}
	impl := s.(*service)
	regSvc = impl
	jni.SetServiceCallbacks(impl.onCreate, impl.onStartCommand, impl.onDestroy)
}

func newService(cfg serviceConfig) *service {
	return &service{
		cfg:  cfg,
		logs: asvclog.NewBuffer(cfg.logCap),
	}
}

type service struct {
	cfg  serviceConfig
	logs asvclog.Buffer

	mu      sync.Mutex
	javaSvc jni.GlobalRef // GoForegroundService instance
	current notif.Handle  // active foreground notification
	pending notif.Notification
}

func (s *service) Logs() asvclog.Buffer { return s.logs }

// --- lifecycle hooks called by jni package ----------------------------------

func (s *service) onCreate(svcRef int64) {
	// Wrap the global ref the cgo trampoline created. We pass it through
	// as an existing global, so we don't need to re-globalref.
	s.mu.Lock()
	s.javaSvc = adoptGlobal(svcRef)
	s.mu.Unlock()
}

func (s *service) onStartCommand(action string) int32 {
	s.mu.Lock()
	pending := s.pending
	s.pending = nil
	javaSvc := s.javaSvc
	s.mu.Unlock()

	if pending == nil {
		// Either Start was not called yet (rare), or this is a sticky
		// re-delivery. Best-effort: build a default notification so we
		// can call startForeground within the 10s window.
		var err error
		pending, err = s.defaultNotification()
		if err != nil {
			return startNotSticky
		}
	}
	if javaSvc == nil {
		return startNotSticky
	}

	if err := s.callStartForeground(javaSvc, pending); err != nil {
		return startNotSticky
	}
	s.mu.Lock()
	if s.current != nil {
		s.current.Release()
	}
	h, _ := pending.Build()
	s.current = h
	s.mu.Unlock()
	return startSticky
}

func (s *service) onDestroy() {
	s.mu.Lock()
	if s.current != nil {
		s.current.Release()
		s.current = nil
	}
	if s.javaSvc != nil {
		s.javaSvc.Release()
		s.javaSvc = nil
	}
	s.mu.Unlock()
}

// --- public API -------------------------------------------------------------

func (s *service) Start(n notif.Notification) error {
	if n == nil {
		var err error
		n, err = s.defaultNotification()
		if err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.pending = n
	s.mu.Unlock()

	return jni.Do(func(e jni.Env) error {
		ctx := e.AppContext()
		bridgeCls, err := e.FindClass("io/gosicp/androidsvc/GoServiceBridge")
		if err != nil {
			return err
		}
		mid, err := e.GetStaticMethodID(bridgeCls, "start", "(Landroid/content/Context;)V")
		if err != nil {
			return err
		}
		return e.CallStaticVoid(bridgeCls, mid, jni.ObjectValue(ctx))
	})
}

func (s *service) Stop() error {
	return jni.Do(func(e jni.Env) error {
		ctx := e.AppContext()
		bridgeCls, err := e.FindClass("io/gosicp/androidsvc/GoServiceBridge")
		if err != nil {
			return err
		}
		mid, err := e.GetStaticMethodID(bridgeCls, "stop", "(Landroid/content/Context;)V")
		if err != nil {
			return err
		}
		return e.CallStaticVoid(bridgeCls, mid, jni.ObjectValue(ctx))
	})
}

func (s *service) UpdateNotification(n notif.Notification) error {
	s.mu.Lock()
	javaSvc := s.javaSvc
	s.mu.Unlock()
	if javaSvc == nil {
		return errors.New("androidsvc: service not running")
	}
	if err := s.callStartForeground(javaSvc, n); err != nil {
		return err
	}
	s.mu.Lock()
	if s.current != nil {
		s.current.Release()
	}
	h, _ := n.Build()
	s.current = h
	s.mu.Unlock()
	return nil
}

// --- helpers ---------------------------------------------------------------

func (s *service) defaultNotification() (notif.Notification, error) {
	if s.cfg.channel == nil {
		return nil, errors.New("androidsvc: no channel configured (use WithChannel)")
	}
	return notif.New(s.cfg.channel,
		notif.WithTitle(s.cfg.notifTitle),
		notif.WithText(s.cfg.notifText),
		notif.WithSmallIcon(s.cfg.notifIcon),
		notif.WithOngoing(true),
	), nil
}

func (s *service) callStartForeground(javaSvc jni.Object, n notif.Notification) error {
	h, err := n.Build()
	if err != nil {
		return err
	}
	notifObj := h.(interface{ Raw() jni.Object }).Raw()

	return jni.Do(func(e jni.Env) error {
		svcCls, err := e.FindClass("io/gosicp/androidsvc/GoForegroundService")
		if err != nil {
			return err
		}
		mid, err := e.GetMethodID(svcCls, "doStartForeground",
			"(ILandroid/app/Notification;I)V")
		if err != nil {
			return err
		}
		return e.CallVoid(javaSvc, mid,
			jni.IntValue(s.cfg.notifID),
			jni.ObjectValue(notifObj),
			jni.IntValue(s.cfg.fgType),
		)
	})
}

// adoptGlobal wraps an existing JNI global reference (created by the cgo
// trampoline in jni package) without re-globalref'ing it.
func adoptGlobal(handle int64) jni.GlobalRef {
	return jni.AdoptGlobal(handle)
}
