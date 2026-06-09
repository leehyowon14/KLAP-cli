package transcript

import "testing"

func TestMacOSBridgeCommandSpecUsesSwiftForScript(t *testing.T) {
	name, args := NewMacOSBridge("bridges/macos/transcribe.swift").commandSpec()
	if name != "swift" {
		t.Fatalf("command name = %q, want swift", name)
	}
	if len(args) != 1 || args[0] != "bridges/macos/transcribe.swift" {
		t.Fatalf("command args = %#v", args)
	}
}

func TestMacOSBridgeCommandSpecRunsBinaryDirectly(t *testing.T) {
	name, args := NewMacOSBridge("bridges/macos/.build/release/TranscriptBridge").commandSpec()
	if name != "bridges/macos/.build/release/TranscriptBridge" {
		t.Fatalf("command name = %q", name)
	}
	if len(args) != 0 {
		t.Fatalf("command args = %#v, want empty", args)
	}
}
