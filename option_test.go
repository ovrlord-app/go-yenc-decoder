package yenc

import "testing"

type testOptionConfig struct {
	Name   string
	Count  int
	Enable bool
}

func TestNewOption_AppliesDefaultsThenOptions(t *testing.T) {
	type option func(*testOptionConfig)

	withName := func(name string) option {
		return func(cfg *testOptionConfig) {
			cfg.Name = name
		}
	}
	withCount := func(count int) option {
		return func(cfg *testOptionConfig) {
			cfg.Count = count
		}
	}

	got := NewOption(
		[]option{withCount(10), withName("override")},
		withName("default"),
		withCount(1),
	)

	if got == nil {
		t.Fatal("expected non-nil option config")
	}
	if got.Name != "override" {
		t.Fatalf("unexpected name: got %q want %q", got.Name, "override")
	}
	if got.Count != 10 {
		t.Fatalf("unexpected count: got %d want %d", got.Count, 10)
	}
}

func TestApplyOption_UsesProvidedStructAndMutatesInPlace(t *testing.T) {
	type option func(*testOptionConfig)

	withEnable := func(enable bool) option {
		return func(cfg *testOptionConfig) {
			cfg.Enable = enable
		}
	}
	withCount := func(count int) option {
		return func(cfg *testOptionConfig) {
			cfg.Count = count
		}
	}

	existing := &testOptionConfig{Name: "persist", Count: 2, Enable: false}
	got := ApplyOption(existing, []option{withEnable(true)}, withCount(5))

	if got != existing {
		t.Fatal("expected ApplyOption to return the provided pointer")
	}
	if got.Name != "persist" {
		t.Fatalf("unexpected name mutation: got %q want %q", got.Name, "persist")
	}
	if got.Count != 5 {
		t.Fatalf("unexpected count: got %d want %d", got.Count, 5)
	}
	if !got.Enable {
		t.Fatal("expected enable to be true")
	}
}

func TestApplyOption_NilInputCreatesStruct(t *testing.T) {
	type option func(*testOptionConfig)

	withName := func(name string) option {
		return func(cfg *testOptionConfig) {
			cfg.Name = name
		}
	}

	got := ApplyOption[testOptionConfig, option](nil, nil, withName("created"))

	if got == nil {
		t.Fatal("expected ApplyOption to allocate a config when input is nil")
	}
	if got.Name != "created" {
		t.Fatalf("unexpected name: got %q want %q", got.Name, "created")
	}
}
