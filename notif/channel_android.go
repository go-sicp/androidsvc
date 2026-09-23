//go:build android

package notif

import (
	"sync"

	"github.com/go-sicp/androidsvc/internal/jni"
)

// NewChannel returns a Channel handle. The channel is not created on the
// system until Ensure is called.
func NewChannel(opts ...ChannelOption) Channel {
	c := &channel{cfg: channelConfig{
		id:   "main",
		name: "Service",
		imp:  ImportanceLow,
	}}
	for _, o := range opts {
		o(&c.cfg)
	}
	return c
}

type channel struct {
	cfg       channelConfig
	once      sync.Once
	created   bool
	createErr error
}

func (c *channel) ID() string { return c.cfg.id }

func (c *channel) Ensure() error {
	c.once.Do(func() {
		c.createErr = jni.Do(func(e jni.Env) error {
			ctx := e.AppContext()

			// NotificationManager nm = (NotificationManager)
			//     ctx.getSystemService(Context.NOTIFICATION_SERVICE);
			ctxCls, err := e.FindClass("android/content/Context")
			if err != nil {
				return err
			}
			getSvc, err := e.GetMethodID(ctxCls, "getSystemService",
				"(Ljava/lang/String;)Ljava/lang/Object;")
			if err != nil {
				return err
			}
			nm, err := e.CallObject(ctx, getSvc, jni.ObjectValue(e.NewString("notification")))
			if err != nil || nm == nil {
				return err
			}

			// NotificationChannel ch = new NotificationChannel(id, name, importance);
			chCls, err := e.FindClass("android/app/NotificationChannel")
			if err != nil {
				return err
			}
			ctor, err := e.GetMethodID(chCls, "<init>",
				"(Ljava/lang/String;Ljava/lang/CharSequence;I)V")
			if err != nil {
				return err
			}
			ch, err := e.NewObject(chCls, ctor,
				jni.ObjectValue(e.NewString(c.cfg.id)),
				jni.ObjectValue(e.NewString(c.cfg.name)),
				jni.IntValue(int32(c.cfg.imp)),
			)
			if err != nil {
				return err
			}

			if c.cfg.desc != "" {
				setDesc, err := e.GetMethodID(chCls, "setDescription",
					"(Ljava/lang/String;)V")
				if err != nil {
					return err
				}
				if err := e.CallVoid(ch, setDesc, jni.ObjectValue(e.NewString(c.cfg.desc))); err != nil {
					return err
				}
			}

			// nm.createNotificationChannel(ch);
			nmCls, err := e.FindClass("android/app/NotificationManager")
			if err != nil {
				return err
			}
			create, err := e.GetMethodID(nmCls, "createNotificationChannel",
				"(Landroid/app/NotificationChannel;)V")
			if err != nil {
				return err
			}
			if err := e.CallVoid(nm, create, jni.ObjectValue(ch)); err != nil {
				return err
			}
			c.created = true
			return nil
		})
	})
	return c.createErr
}
