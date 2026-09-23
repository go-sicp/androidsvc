//go:build android

package notif

import (
	"github.com/go-sicp/androidsvc/internal/jni"
)

// New builds a Notification under the given channel.
func New(ch Channel, opts ...Option) Notification {
	n := &notification{cfg: notifConfig{channel: ch, ongoing: true}}
	for _, o := range opts {
		o(&n.cfg)
	}
	return n
}

type notification struct {
	cfg notifConfig
}

type notifHandle struct {
	gref jni.GlobalRef
}

func (h *notifHandle) Release() { h.gref.Release() }

// Raw returns the global ref so the service package can call
// service.startForeground(id, notif). Not part of the public Handle interface.
func (h *notifHandle) Raw() jni.Object { return h.gref }

func (n *notification) Build() (Handle, error) {
	if err := n.cfg.channel.Ensure(); err != nil {
		return nil, err
	}
	var out *notifHandle
	err := jni.Do(func(e jni.Env) error {
		ctx := e.AppContext()

		// new Notification.Builder(ctx, channelId)
		builderCls, err := e.FindClass("android/app/Notification$Builder")
		if err != nil {
			return err
		}
		ctor, err := e.GetMethodID(builderCls, "<init>",
			"(Landroid/content/Context;Ljava/lang/String;)V")
		if err != nil {
			return err
		}
		b, err := e.NewObject(builderCls, ctor,
			jni.ObjectValue(ctx),
			jni.ObjectValue(e.NewString(n.cfg.channel.ID())),
		)
		if err != nil {
			return err
		}

		setTitle, err := e.GetMethodID(builderCls, "setContentTitle",
			"(Ljava/lang/CharSequence;)Landroid/app/Notification$Builder;")
		if err != nil {
			return err
		}
		if _, err := e.CallObject(b, setTitle, jni.ObjectValue(e.NewString(n.cfg.title))); err != nil {
			return err
		}

		setText, err := e.GetMethodID(builderCls, "setContentText",
			"(Ljava/lang/CharSequence;)Landroid/app/Notification$Builder;")
		if err != nil {
			return err
		}
		if _, err := e.CallObject(b, setText, jni.ObjectValue(e.NewString(n.cfg.text))); err != nil {
			return err
		}

		if n.cfg.icon != 0 {
			setIcon, err := e.GetMethodID(builderCls, "setSmallIcon",
				"(I)Landroid/app/Notification$Builder;")
			if err != nil {
				return err
			}
			if _, err := e.CallObject(b, setIcon, jni.IntValue(n.cfg.icon)); err != nil {
				return err
			}
		}

		setOngoing, err := e.GetMethodID(builderCls, "setOngoing",
			"(Z)Landroid/app/Notification$Builder;")
		if err != nil {
			return err
		}
		if _, err := e.CallObject(b, setOngoing, jni.BoolValue(n.cfg.ongoing)); err != nil {
			return err
		}

		build, err := e.GetMethodID(builderCls, "build",
			"()Landroid/app/Notification;")
		if err != nil {
			return err
		}
		built, err := e.CallObject(b, build)
		if err != nil {
			return err
		}
		out = &notifHandle{gref: e.NewGlobalRef(built)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
