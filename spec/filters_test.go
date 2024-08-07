package spec

import "testing"

func TestVerify_brokenTopicFormater(t *testing.T) {
	impl := func(f string) bool { return false }
	if err := VerifyTopicNameFormat(impl); err == nil {
		t.Error("should return error")
	}
	impl = func(f string) bool { return true }
	if err := VerifyTopicNameFormat(impl); err == nil {
		t.Error("should return error")
	}
}

func TestVerify_brokenFilterFormater(t *testing.T) {
	impl := func(f string) bool { return false }
	if err := VerifyFilterFormat(impl); err == nil {
		t.Error("should return error")
	}
	impl = func(f string) bool { return true }
	if err := VerifyFilterFormat(impl); err == nil {
		t.Error("should return error")
	}
}

func TestVerify_brokenMatcher(t *testing.T) {
	impl := func(f, n string) bool { return false }
	if err := VerifyFilterMatching(impl); err == nil {
		t.Error("should return error")
	}
	impl = func(f, n string) bool { return true }
	if err := VerifyFilterMatching(impl); err == nil {
		t.Error("should return error")
	}
}
