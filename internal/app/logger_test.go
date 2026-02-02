package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupLogger(t *testing.T) {
	t.Run("success with rd", //nolint:paralleltest // uses os.Unsetenv
		func(t *testing.T) {
			os.Unsetenv(LogEnvVar)
			tempDir := t.TempDir()
			logLevel := &slog.LevelVar{}
			stderr := &bytes.Buffer{}
			logger, closer, err := setupLogger(stderr, logLevel, tempDir)
			require.NoError(t, err)
			defer closer.Close()
			assert.NotNil(t, logger)
			assert.FileExists(t, filepath.Join(tempDir, LogFile))
		})

	t.Run("success with no rd", //nolint:paralleltest // uses os.Unsetenv
		func(t *testing.T) {
			os.Unsetenv(LogEnvVar)
			defer os.Remove(LogFile)
			logLevel := &slog.LevelVar{}
			stderr := &bytes.Buffer{}
			logger, closer, err := setupLogger(stderr, logLevel, "")
			require.NoError(t, err)
			defer closer.Close()
			assert.NotNil(t, logger)
			assert.FileExists(t, LogFile)
		})

	t.Run("success with file", //nolint:paralleltest // uses temp file
		func(t *testing.T) {
			tempDir := t.TempDir()
			logLevel := &slog.LevelVar{}
			logLevel.Set(slog.LevelInfo)
			stderr := &bytes.Buffer{}

			logger, closer, err := setupLogger(stderr, logLevel, tempDir)
			require.NoError(t, err)
			require.NotNil(t, logger)
			require.NotNil(t, closer)
			defer closer.Close()

			logger.Info("test message", "key", "value")

			// Check console output
			assert.Contains(t, stderr.String(), "test message")
			assert.NotContains(t, stderr.String(), "key=value") // Info doesn't show attrs by default

			// Check file output
			logFile := filepath.Join(tempDir, ".jsm.log")
			data, err := os.ReadFile(logFile)
			require.NoError(t, err)
			assert.Contains(t, string(data), `"msg":"test message"`)
			assert.Contains(t, string(data), `"key":"value"`)
		})

	t.Run("success with env var override",
		func(t *testing.T) {
			tempDir := t.TempDir()
			logFile := filepath.Join(tempDir, "custom.log")
			t.Setenv(LogEnvVar, logFile)

			logLevel := &slog.LevelVar{}
			stderr := &bytes.Buffer{}

			logger, closer, err := setupLogger(stderr, logLevel, "")
			require.NoError(t, err)
			defer closer.Close()

			logger.Info("custom log")
			data, _ := os.ReadFile(logFile)
			assert.Contains(t, string(data), "custom log")
		})

	t.Run("success with empty rd", //nolint:paralleltest // uses cleanup
		func(t *testing.T) {
			// This will create .jsm.log in the current directory, which is fine for a test that cleans up
			defer os.Remove(".jsm.log")

			logLevel := &slog.LevelVar{}
			stderr := &bytes.Buffer{}
			logger, closer, err := setupLogger(stderr, logLevel, "")
			require.NoError(t, err)
			defer closer.Close()

			logger.Info("empty rd")
			assert.FileExists(t, ".jsm.log")
		})

	t.Run("fallback on file error", //nolint:paralleltest // uses files
		func(t *testing.T) {
			logLevel := &slog.LevelVar{}
			stderr := &bytes.Buffer{}

			// Point to a non-existent directory that cannot be created
			logger, closer, err := setupLogger(stderr, logLevel, "/non/existent/path/unwritable")
			require.Error(t, err)
			assert.Nil(t, closer)
			assert.NotNil(t, logger)

			logger.Info("fallback message")
			assert.Contains(t, stderr.String(), "fallback message")
		})
}

func TestConsoleHandler_Thorough(t *testing.T) {
	t.Parallel()

	t.Run("levels", func(t *testing.T) {
		t.Parallel()
		logLevel := &slog.LevelVar{}
		buf := &bytes.Buffer{}
		handler := &consoleHandler{w: buf, level: logLevel}

		tests := []struct {
			level slog.Level
			msg   string
			want  string
		}{
			{slog.LevelDebug, "d", "d\n"},
			{slog.LevelInfo, "i", "i\n"},
			{slog.LevelWarn, "w", "Warning: w\n"},
			{slog.LevelError, "e", "Error: e\n"},
		}
		for _, tt := range tests {
			buf.Reset()
			logLevel.Set(slog.LevelDebug) // Enable all
			err := handler.Handle(context.Background(), slog.Record{Level: tt.level, Message: tt.msg})
			require.NoError(t, err)
			assert.Equal(t, tt.want, buf.String())
		}
	})

	t.Run("attributes and formatting", func(t *testing.T) {
		t.Parallel()
		logLevel := &slog.LevelVar{}
		buf := &bytes.Buffer{}
		handler := &consoleHandler{w: buf, level: logLevel}

		buf.Reset()
		logLevel.Set(slog.LevelDebug)

		// Test formatAttr with error
		rec := slog.NewRecord(time.Now(), slog.LevelError, "msg", 0)
		rec.AddAttrs(slog.Attr{Key: "err", Value: slog.StringValue("boom")})
		err := handler.Handle(context.Background(), rec)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Error: msg: boom")

		// Test attributes on non-debug mode (should be hidden unless error)
		buf.Reset()
		logLevel.Set(slog.LevelInfo)
		rec2 := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		rec2.AddAttrs(slog.Attr{Key: "foo", Value: slog.StringValue("bar")})
		err2 := handler.Handle(context.Background(), rec2)
		require.NoError(t, err2)
		assert.Equal(t, "msg\n", buf.String())

		// Test WithAttrs
		buf.Reset()
		logLevel.Set(slog.LevelDebug)
		h2 := handler.WithAttrs([]slog.Attr{slog.Int("pid", 123)})
		err3 := h2.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0))
		require.NoError(t, err3)
		assert.Contains(t, buf.String(), "msg pid=123")

		// Test WithGroup
		h3 := h2.WithGroup("somegroup")
		assert.Equal(t, h2, h3) // Currently returns self
	})
}

type errHandler struct{}

func (e *errHandler) Enabled(context.Context, slog.Level) bool  { return true }
func (e *errHandler) Handle(context.Context, slog.Record) error { return errors.New("handler error") }
func (e *errHandler) WithAttrs(_ []slog.Attr) slog.Handler      { return e }
func (e *errHandler) WithGroup(_ string) slog.Handler           { return e }

func TestMultiHandler_Thorough(t *testing.T) {
	t.Parallel()

	t.Run("Enabled", func(t *testing.T) {
		t.Parallel()
		h1 := &consoleHandler{w: &bytes.Buffer{}, level: &slog.LevelVar{}}
		h2 := &consoleHandler{w: &bytes.Buffer{}, level: &slog.LevelVar{}}
		multi := &multiHandler{handlers: []slog.Handler{h1, h2}}

		assert.True(t, multi.Enabled(context.Background(), slog.LevelInfo))

		// False case: none enabled
		h1.level.Set(slog.LevelError)
		h2.level.Set(slog.LevelError)
		assert.False(t, multi.Enabled(context.Background(), slog.LevelInfo))
	})

	t.Run("Handle error propagation", func(t *testing.T) {
		t.Parallel()
		eh := &errHandler{}
		m2 := &multiHandler{handlers: []slog.Handler{eh}}
		err := m2.Handle(context.Background(), slog.Record{Level: slog.LevelInfo})
		require.Error(t, err)
	})

	t.Run("WithAttrs and WithGroup", func(t *testing.T) {
		t.Parallel()
		h1 := &consoleHandler{w: &bytes.Buffer{}, level: &slog.LevelVar{}}
		h2 := &consoleHandler{w: &bytes.Buffer{}, level: &slog.LevelVar{}}
		multi := &multiHandler{handlers: []slog.Handler{h1, h2}}

		m3 := multi.WithAttrs([]slog.Attr{slog.String("v", "1")})
		assert.IsType(t, &multiHandler{}, m3)

		m4 := m3.WithGroup("g")
		assert.IsType(t, &multiHandler{}, m4)
	})
}
