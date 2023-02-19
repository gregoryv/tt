package ttsrv

import (
	"net/url"
	"time"
)

func NewBindConf(uri, acceptTimeout string) (*BindConf, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	d, err := time.ParseDuration(acceptTimeout)
	if err != nil {
		return nil, err
	}

	return &BindConf{
		URL:           u,
		AcceptTimeout: d,
	}, nil
}

type BindConf struct {
	*url.URL
	AcceptTimeout time.Duration
	Debug         bool
}
