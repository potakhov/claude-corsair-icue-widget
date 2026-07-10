package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
}

func programDataDir() string {
	pd := os.Getenv("ProgramData")
	if pd == "" {
		pd = `C:\ProgramData`
	}
	return filepath.Join(pd, "xeneon-bridge")
}

// ProgramDataDir is the machine-wide config directory used by the Windows service.
func ProgramDataDir() string { return programDataDir() }

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// Dir resolves the config directory. Order: explicit XENEON_BRIDGE_HOME override,
// then the machine-wide ProgramData dir IF it already holds a config (i.e. the
// service has been installed), then the per-user home dir (default for
// non-service use). Only the elevated `service install` ever creates the
// ProgramData config, so non-elevated callers read it when present and otherwise
// behave exactly as before.
func Dir() (string, error) {
	if h := os.Getenv("XENEON_BRIDGE_HOME"); h != "" {
		return h, nil
	}
	if pd := programDataDir(); fileExists(filepath.Join(pd, "config.json")) {
		return pd, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".xeneon-bridge"), nil
}

// InstallProgramDataConfig creates the machine-wide config if absent, migrating an
// existing per-user token+port so the widget keeps working. Idempotent: if the
// ProgramData config already exists it is returned unchanged. Intended to run
// elevated (from `service install`).
func InstallProgramDataConfig() (Config, string, error) {
	dir := programDataDir()
	path := filepath.Join(dir, "config.json")
	if cfg, ok, err := tryRead(path); err != nil {
		return Config{}, "", err
	} else if ok {
		return cfg, path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Config{}, "", err
	}
	cfg := Config{Port: 8787, Token: newToken()}
	if home, err := os.UserHomeDir(); err == nil {
		if hc, ok, _ := tryRead(filepath.Join(home, ".xeneon-bridge", "config.json")); ok {
			cfg = hc // migrate existing token + port verbatim
		}
	}
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return Config{}, "", err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return Config{}, "", err
	}
	return cfg, path, nil
}

func Path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// Load reads the config, atomically creating it with a random token on first
// run. Concurrent first-run callers converge on a single token: one wins the
// exclusive create and writes it; every other caller waits for and reads the
// winner's file (never its own token, never a partial read).
func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return Config{}, err
	}
	// Fast path: an existing, fully-written config.
	if cfg, ok, err := tryRead(p); err != nil {
		return Config{}, err
	} else if ok {
		return cfg, nil
	}
	// Absent, or present-but-empty (a winner is mid-create). Try to become the
	// winner via exclusive create.
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return Config{}, err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return readWithRetry(p) // someone else won; wait for their content
		}
		return Config{}, err
	}
	cfg := Config{Port: 8787, Token: newToken()}
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		f.Close()
		os.Remove(p) // don't strand waiting readers on an empty file
		return Config{}, err
	}
	if _, werr := f.Write(body); werr != nil {
		f.Close()
		os.Remove(p)
		return Config{}, werr
	}
	if cerr := f.Close(); cerr != nil {
		return Config{}, cerr
	}
	return cfg, nil
}

// tryRead reads a fully-written config. ok=false means the file is absent or
// present-but-empty (both mean "not ready — create it or wait for the winner").
func tryRead(p string) (Config, bool, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, false, nil
		}
		return Config{}, false, err
	}
	if len(data) == 0 {
		return Config{}, false, nil
	}
	cfg, perr := parse(data)
	if perr != nil {
		return Config{}, false, perr
	}
	return cfg, true, nil
}

func parse(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Port == 0 {
		cfg.Port = 8787
	}
	return cfg, nil
}

// readWithRetry waits out the brief window between a winner's O_EXCL create and
// its write, retrying an empty or torn read until the file is fully written.
func readWithRetry(p string) (Config, error) {
	lastErr := errors.New("config not ready")
	for i := 0; i < 50; i++ {
		cfg, ok, err := tryRead(p)
		if ok {
			return cfg, nil
		}
		if err != nil {
			lastErr = err
		}
		time.Sleep(2 * time.Millisecond)
	}
	return Config{}, lastErr
}

func Save(cfg Config) error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, "config.json"), data, 0o600)
}

func newToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
