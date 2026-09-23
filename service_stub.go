//go:build !android

package androidsvc

import (
	"errors"

	asvclog "github.com/go-sicp/androidsvc/log"
	"github.com/go-sicp/androidsvc/notif"
)

var errNotAndroid = errors.New("androidsvc: only available on android")

func registerService(s Service) {}

func newService(cfg serviceConfig) *service {
	return &service{logs: asvclog.NewBuffer(cfg.logCap)}
}

type service struct {
	logs asvclog.Buffer
}

func (s *service) Logs() asvclog.Buffer                        { return s.logs }
func (s *service) Start(n notif.Notification) error            { return errNotAndroid }
func (s *service) Stop() error                                 { return errNotAndroid }
func (s *service) UpdateNotification(notif.Notification) error { return errNotAndroid }
