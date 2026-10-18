package filestorage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// adversarialKeys are image keys aimed at escaping the configured base
// directory through the storage path.
var adversarialKeys = []string{
	"../../../../etc/passwd",
	"../foo/../../bar",
	"a/../../..",
	"foo/../../bar",
	"....//....//escaped",
	// backslash separators are path separators on Windows once they reach
	// filepath.Join, and are only escaped in Normalize when SafeChars is
	// not the no-op "--" config
	"a\\..\\..\\evil",
	"..\\..\\evil",
	// absolute keys are clamped by imagorpath.Normalize onto the pseudo
	// root, so they must stay accepted and confined
	"/../../etc/passwd",
}

func assertConfined(t *testing.T, baseDir, p string) {
	t.Helper()
	rel, err := filepath.Rel(filepath.Clean(baseDir), p)
	require.NoError(t, err)
	assert.True(t, rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)),
		"path %q escapes base dir %q", p, baseDir)
}

// TestFileStoragePathConfinedToBaseDir asserts the security invariant that
// any accepted storage path stays within the configured base directory,
// for every storage configuration, independent of blacklist settings.
func TestFileStoragePathConfinedToBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	configs := map[string]*FileStorage{
		"defaults":         New(baseDir),
		"no-op safe chars": New(baseDir, WithSafeChars("--")),
		"empty blacklists": func() *FileStorage {
			s := New(baseDir)
			// a storage constructed as a struct literal, or a consumer
			// overriding Blacklists, must not lose base dir containment
			s.Blacklists = nil
			return s
		}(),
	}
	for name, s := range configs {
		s := s
		t.Run(name, func(t *testing.T) {
			for _, key := range adversarialKeys {
				p, ok := s.Path(key)
				if !ok {
					assert.Empty(t, p, "rejected key %q must return empty path", key)
					continue
				}
				assertConfined(t, baseDir, p)
			}
		})
	}
}

// TestFileStoragePathRejectsDotTraversalUnderDefaults pins the default
// blacklist defense: the built-in dot-file blacklist rejects any image key
// containing dot segments, which is what blocks relative traversal keys
// in the default configuration.
func TestFileStoragePathRejectsDotTraversalUnderDefaults(t *testing.T) {
	s := New(t.TempDir())
	for _, key := range []string{
		"../../../../etc/passwd",
		"../foo/../../bar",
		"a/../../..",
		"foo/.git/config",
	} {
		p, ok := s.Path(key)
		assert.False(t, ok, "key %q must be rejected under defaults", key)
		assert.Empty(t, p)
	}
}

// TestFileStoragePutConfinedToBaseDir proves the write primitive stays
// inside the base directory even without the blacklist line of defense.
func TestFileStoragePutConfinedToBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	s := New(baseDir)
	s.Blacklists = nil
	err := s.Put(context.Background(), "../../escaped.txt",
		imagor.NewBlobFromBytes([]byte("pwned")))
	assert.ErrorIs(t, err, imagor.ErrInvalid)
	_, statErr := os.Stat(filepath.Join(filepath.Dir(baseDir), "escaped.txt"))
	assert.True(t, os.IsNotExist(statErr),
		"nothing may be written outside the base dir")
}
