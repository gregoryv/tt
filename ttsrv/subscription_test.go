package ttsrv

import (
	"testing"

	"github.com/gregoryv/tt/ttx"
)

func TestSubscription_String(t *testing.T) {
	sub := MustNewSubscription("all/gophers/#", ttx.NoopPub)
	if v := sub.String(); v != "all/gophers/#" {
		t.Errorf("unexpected subscription string %q", sub)
	}
}

func TestMustNewSubscription(t *testing.T) {
	defer catchPanic(t)
	MustNewSubscription("")
}

func catchPanic(t *testing.T) {
	if e := recover(); e == nil {
		t.Fatal("expect panic")
	}
}
