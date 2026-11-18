package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/platform"
)

type backupIntegrationRunner struct {
	backupRunner
}

func (r *backupIntegrationRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "tmutil" {
		switch args[0] {
		case "listlocalsnapshotdates":
			return []byte("2026-01-01-100000\n2026-01-01-100100\n"), nil
		case "localsnapshot":
			return []byte("Created local snapshot with date: 2026-11-17-130000"), nil
		case "status", "deletelocalsnapshots":
			return r.backupRunner.Run(ctx, name, args...)
		}
	}
	return nil, fmt.Errorf("unmocked command: %s %v", name, args)
}

func TestThinSkipsBackup(t *testing.T) {
	for _, jsonOut := range []string{"false", "true"} {
		t.Run(jsonOut, func(t *testing.T) {
			viper.Reset()
			defer viper.Reset()
			config.SetDefaults()
			originalRunner, originalRequire := newRunner, requireTmutil
			defer func() { newRunner, requireTmutil = originalRunner, originalRequire }()
			r := &backupIntegrationRunner{}
			newRunner = func() platform.CommandRunner { return r }
			requireTmutil = func() error { return nil }
			var out, diagnostic bytes.Buffer
			thinCmd.SetOut(&out)
			thinCmd.SetErr(&diagnostic)
			defer func() { thinCmd.SetOut(nil); thinCmd.SetErr(nil) }()
			setFlag(t, thinCmd, "json", jsonOut)
			setTestContext(thinCmd)
			if err := runThin(thinCmd, nil); err != nil {
				t.Fatal(err)
			}
			want := "Thinned 0 snapshot(s)\n"
			if jsonOut == "true" {
				want = "{\n  \"thinned\": 0\n}\n"
			}
			if out.String() != want || !strings.Contains(diagnostic.String(), "backup is running") || len(r.deleted) != 0 {
				t.Fatalf("out=%q diagnostic=%q deletions=%v", out.String(), diagnostic.String(), r.deleted)
			}
		})
	}
}

func TestRunIterationSkipsBackup(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()
	r := &backupIntegrationRunner{}
	log := logger.New(logger.Options{MaxEntries: 50})
	defer log.Close()
	var out bytes.Buffer
	if err := runIteration(context.Background(), &out, log, r, config.Load()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Created:", "Thinning skipped: Time Machine backup is running", "2 snapshot(s)"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in %q", want, out.String())
		}
	}
	if len(r.deleted) != 0 {
		t.Fatalf("backup snapshot deleted: %v", r.deleted)
	}
	found := false
	for _, entry := range log.Entries() {
		if strings.Contains(entry.Message, "Thinning skipped:") {
			found = true
		}
	}
	if !found {
		t.Fatal("skip reason missing from service log")
	}
}
