//go:build !android

package notif

import "errors"

var errNotAndroid = errors.New("androidsvc/notif: only available on android")

func NewChannel(opts ...ChannelOption) Channel    { return stubChannel{} }
func New(ch Channel, opts ...Option) Notification { return stubNotif{} }

type stubChannel struct{}

func (stubChannel) ID() string    { return "" }
func (stubChannel) Ensure() error { return errNotAndroid }

type stubNotif struct{}

func (stubNotif) Build() (Handle, error) { return nil, errNotAndroid }
