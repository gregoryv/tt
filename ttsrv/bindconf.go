package ttsrv

import (
	"net/url"
	"time"
)

type BindConf struct {
	*url.URL
	AcceptTimeout time.Duration
	Debug         bool
}
