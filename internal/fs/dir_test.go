package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUintSubdirectories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		wantNums []uint64
		wantErr  bool
	}{
		{
			name: "valid subdirectories sorted ascending",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(dir, "1"), 0o755))
				require.NoError(t, os.Mkdir(filepath.Join(dir, "2"), 0o755))
				require.NoError(t, os.Mkdir(filepath.Join(dir, "10"), 0o755))
				return dir
			},
			wantNums: []uint64{1, 2, 10},
		},
		{
			name: "directories created out of order are returned sorted",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				// Create directories in non-sequential order
				require.NoError(t, os.Mkdir(filepath.Join(dir, "10"), 0o755))
				require.NoError(t, os.Mkdir(filepath.Join(dir, "3"), 0o755))
				require.NoError(t, os.Mkdir(filepath.Join(dir, "7"), 0o755))
				require.NoError(t, os.Mkdir(filepath.Join(dir, "1"), 0o755))
				return dir
			},
			wantNums: []uint64{1, 3, 7, 10},
		},
		{
			name: "ignores non-numeric subdirectories",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(dir, "1"), 0o755))
				require.NoError(t, os.Mkdir(filepath.Join(dir, "abc"), 0o755))
				return dir
			},
			wantNums: []uint64{1},
		},
		{
			name: "ignores files",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(dir, "1"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "2"), []byte("file"), 0o600))
				return dir
			},
			wantNums: []uint64{1},
		},
		{
			name: "empty directory",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantNums: nil,
		},
		{
			name: "non-existent directory",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "non-existent")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.setup(t)
			nums, err := GetUintSubdirectories(path)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantNums, nums, "results should be sorted in ascending order")
		})
	}
}
