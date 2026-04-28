package main

import "testing"

func TestTUIEnvironmentDisablesITerm2CapabilityProbe(t *testing.T) {
	env := tuiEnvironment([]string{"TERM_PROGRAM=iTerm.app", "TERM=xterm-256color"})
	if !hasEnv(env, "SSH_TTY") {
		t.Fatalf("expected SSH_TTY workaround in %#v", env)
	}
}

func TestTUIEnvironmentLeavesOtherTerminalsUnchanged(t *testing.T) {
	env := []string{"TERM_PROGRAM=Apple_Terminal", "TERM=xterm-256color"}
	got := tuiEnvironment(env)
	if len(got) != len(env) {
		t.Fatalf("env changed: %#v", got)
	}
}

func TestTUIEnvironmentPreservesRealSSH(t *testing.T) {
	env := []string{"TERM_PROGRAM=iTerm.app", "SSH_TTY=/dev/ttys001"}
	got := tuiEnvironment(env)
	if len(got) != len(env) {
		t.Fatalf("env changed: %#v", got)
	}
}
