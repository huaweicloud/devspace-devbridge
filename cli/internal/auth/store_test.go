package auth

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"

	"github.com/zalando/go-keyring"
)

// fakeSecretBackend records every call made to the Secret Service backend so a
// test can assert that go-keyring was never touched when no session bus is
// available.
type fakeSecretBackend struct {
	setCalls    int
	getCalls    int
	deleteCalls int

	setErr error
	stored map[string]string
}

func (f *fakeSecretBackend) Set(service, user, password string) error {
	f.setCalls++
	if f.setErr != nil {
		return f.setErr
	}
	f.stored[service+"/"+user] = password
	return nil
}

func (f *fakeSecretBackend) Get(service, user string) (string, error) {
	f.getCalls++
	if value, ok := f.stored[service+"/"+user]; ok {
		return value, nil
	}
	return "", keyring.ErrNotFound
}

func (f *fakeSecretBackend) Delete(service, user string) error {
	f.deleteCalls++
	delete(f.stored, service+"/"+user)
	return nil
}

// useFakeSecretBackend replaces the Secret Service backend for one test.
func useFakeSecretBackend(t *testing.T) *fakeSecretBackend {
	t.Helper()
	fake := &fakeSecretBackend{stored: make(map[string]string)}
	previous := credentialsBackend
	credentialsBackend = fake
	t.Cleanup(func() { credentialsBackend = previous })
	return fake
}

// prepareHeadlessEnv puts the process in a state with no D-Bus session bus and
// a private HOME, resetting the cached probe and the one-time notice.
func prepareHeadlessEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(envSessionBusAddress, "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	resetBusProbe()
	resetFallbackNotice()
	t.Cleanup(func() {
		_ = os.Unsetenv(envSessionBusAddress)
		resetBusProbe()
		resetFallbackNotice()
	})
	return home
}

func configFilePath(t *testing.T, home string) string {
	t.Helper()
	return filepath.Join(home, ".huawei", "devbridge", "config.yaml")
}

func TestStoreAndLoadCredentialFallBackToConfigFileWithoutSessionBus(t *testing.T) {
	home := prepareHeadlessEnv(t)
	fake := useFakeSecretBackend(t)

	cred := &Credential{APIKey: "headless-api-key"}
	userInfo := &UserInfo{UserName: "tester", UserID: "42"}

	if err := StoreCredential(CredentialName, cred, userInfo); err != nil {
		t.Fatalf("StoreCredential returned an error: %v", err)
	}
	if fake.setCalls != 0 {
		t.Fatalf("go-keyring must not be touched without a session bus: got %d Set calls", fake.setCalls)
	}

	path := configFilePath(t, home)
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("credential file was not written: %v", err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0o600 {
		t.Fatalf("credential file permissions: got %o, want 600", perm)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("config directory does not exist: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Fatalf("config directory permissions: got %o, want 700", perm)
	}

	loaded, loadedUser, err := LoadCredential(CredentialName)
	if err != nil {
		t.Fatalf("LoadCredential returned an error: %v", err)
	}
	if loaded == nil || loaded.APIKey != cred.APIKey {
		t.Fatalf("credential round trip failed: got %+v, want api key %q", loaded, cred.APIKey)
	}
	if loadedUser == nil || loadedUser.UserName != userInfo.UserName {
		t.Fatalf("user info round trip failed: got %+v", loadedUser)
	}
	if fake.getCalls != 0 {
		t.Fatalf("go-keyring must not be touched without a session bus: got %d Get calls", fake.getCalls)
	}
}

func TestStoreCredentialPrefersSecretServiceWithSessionBus(t *testing.T) {
	prepareHeadlessEnv(t)
	t.Setenv(envSessionBusAddress, "unix:path=/run/user/1000/bus")
	resetBusProbe()
	fake := useFakeSecretBackend(t)

	cred := &Credential{APIKey: "keyring-api-key"}
	if err := StoreCredential(CredentialName, cred, nil); err != nil {
		t.Fatalf("StoreCredential returned an error: %v", err)
	}
	if fake.setCalls != 1 {
		t.Fatalf("the Secret Service must be preferred when a session bus exists: got %d Set calls", fake.setCalls)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if _, ok := cfg[credentialsKey]; ok {
		t.Fatal("the credential must not be duplicated into the config file when the keyring accepts it")
	}
}

func TestStoreCredentialKeepsConfigFileWhenKeyringFails(t *testing.T) {
	prepareHeadlessEnv(t)
	t.Setenv(envSessionBusAddress, "unix:path=/run/user/1000/bus")
	resetBusProbe()
	fake := useFakeSecretBackend(t)
	fake.setErr = errors.New("secret service unavailable")

	cred := &Credential{APIKey: "fallback-api-key"}
	if err := StoreCredential(CredentialName, cred, nil); err != nil {
		t.Fatalf("StoreCredential returned an error: %v", err)
	}
	if fake.setCalls != 1 {
		t.Fatalf("expected one keyring attempt before falling back: got %d", fake.setCalls)
	}

	loaded, _, err := LoadCredential(CredentialName)
	if err != nil {
		t.Fatalf("LoadCredential returned an error: %v", err)
	}
	if loaded == nil || loaded.APIKey != cred.APIKey {
		t.Fatalf("credential must survive in the config file: got %+v", loaded)
	}
}

func TestLoadCredentialReadsSecretServiceWithSessionBus(t *testing.T) {
	prepareHeadlessEnv(t)
	t.Setenv(envSessionBusAddress, "unix:path=/run/user/1000/bus")
	resetBusProbe()
	fake := useFakeSecretBackend(t)
	blob := encodeCredential(&Credential{APIKey: "stored-in-keyring"})
	fake.stored[CredentialName+"/"+keyringUser] = blob

	loaded, _, err := LoadCredential(CredentialName)
	if err != nil {
		t.Fatalf("LoadCredential returned an error: %v", err)
	}
	if loaded == nil || loaded.APIKey != "stored-in-keyring" {
		t.Fatalf("unexpected credential: %+v", loaded)
	}
	if fake.getCalls != 1 {
		t.Fatalf("the Secret Service must be read when a session bus exists: got %d Get calls", fake.getCalls)
	}
}

func TestDeleteCredentialSkipsSecretServiceWithoutSessionBus(t *testing.T) {
	prepareHeadlessEnv(t)
	fake := useFakeSecretBackend(t)

	if err := StoreCredential(CredentialName, &Credential{APIKey: "to-be-deleted"}, &UserInfo{UserName: "tester"}); err != nil {
		t.Fatalf("StoreCredential returned an error: %v", err)
	}
	if err := DeleteCredential(CredentialName); err != nil {
		t.Fatalf("DeleteCredential returned an error: %v", err)
	}
	if fake.deleteCalls != 0 {
		t.Fatalf("go-keyring must not be touched without a session bus: got %d Delete calls", fake.deleteCalls)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if _, ok := cfg[credentialsKey]; ok {
		t.Fatal("the config file must not keep the credential after logout")
	}
	if _, ok := cfg[userInfoKey]; ok {
		t.Fatal("the config file must not keep the user info after logout")
	}
}

func TestDeleteCredentialClearsSecretServiceWithSessionBus(t *testing.T) {
	prepareHeadlessEnv(t)
	t.Setenv(envSessionBusAddress, "unix:path=/run/user/1000/bus")
	resetBusProbe()
	fake := useFakeSecretBackend(t)

	if err := DeleteCredential(CredentialName); err != nil {
		t.Fatalf("DeleteCredential returned an error: %v", err)
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("expected one Secret Service delete with a session bus: got %d", fake.deleteCalls)
	}
}

// syncBuffer is an io.Writer safe for the slog handler used in tests.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestCredentialFallbackNoticeIsEmittedOncePerProcess(t *testing.T) {
	home := prepareHeadlessEnv(t)

	var out syncBuffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	for i := 0; i < 3; i++ {
		_, _, _ = LoadCredential(CredentialName)
	}

	if got := strings.Count(out.String(), "level=WARN"); got != 1 {
		t.Fatalf("the degradation notice must be emitted once per process: got %d warnings (%s)", got, out.String())
	}
	if !strings.Contains(out.String(), configFilePath(t, home)) {
		t.Fatalf("the notice must state where the credential is stored: %s", out.String())
	}
	if !strings.Contains(out.String(), "base64") {
		t.Fatalf("the notice must state that the file is not encrypted: %s", out.String())
	}
}

func TestCredentialFallbackNoticeCoversBothLanguages(t *testing.T) {
	msg := i18n.Msg.Auth.CredentialBackendFallback
	if msg.ZH == "" || msg.EN == "" {
		t.Fatal("the degradation notice must be translated for both zh and en")
	}
	if !strings.Contains(msg.ZH, "%s") || !strings.Contains(msg.EN, "%s") {
		t.Fatal("the degradation notice must leave a placeholder for the credential file path")
	}
}

func TestSecretServiceUsableWithSessionBus(t *testing.T) {
	prepareHeadlessEnv(t)
	t.Setenv(envSessionBusAddress, "unix:path=/run/user/1000/bus")
	resetBusProbe()

	if !useSecretService() {
		t.Fatal("the Secret Service must stay enabled when a session bus exists")
	}
}
