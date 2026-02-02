package schema

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andyballingall/json-schema-manager/internal/validator"
)

func TestTestLog_AddTest(t *testing.T) {
	t.Parallel()
	l := NewTestLog()
	k := Key("key")
	s1 := &Spec{}
	s2 := &Spec{}
	l.AddTest(k, s1)
	l.AddTest(k, s2)
	assert.Len(t, l[k], 2)
}

func TestTestReport_AddTests(t *testing.T) {
	t.Parallel()
	r := NewTestReport()
	k := Key("key")
	s1 := &Spec{}
	s2 := &Spec{}
	r.AddPassedTest(k, s1)
	r.AddFailedTest(k, s2)
	assert.Len(t, r.PassedTests[k], 1)
	assert.Len(t, r.FailedTests[k], 1)
}

func TestTester_Configuration(t *testing.T) {
	t.Parallel()
	reg := &Registry{}
	tr := NewTester(reg)

	tr.SetScope(TestScopePass)
	assert.Equal(t, TestScopePass, tr.scope)

	tr.SetStopOnFirstError(false)
	assert.False(t, tr.stopOnFirstError)

	tr.SetNumWorkers(10)
	assert.Equal(t, 10, tr.numWorkers)
}

func TestTester_TestSingleSchema(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})

	s, _ := r.GetSchemaByKey(k)
	homeDir := s.Path(HomeDir)
	passDir := filepath.Join(homeDir, string(TestDocTypePass))
	failDir := filepath.Join(homeDir, string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(passDir, 0o755))
	require.NoError(t, os.MkdirAll(failDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "valid.json"), []byte("{}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(failDir, "invalid.json"), []byte("[]"), 0o600))

	tr := NewTester(r)
	tr.SetStopOnFirstError(false)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mc, ok := r.compiler.(*mockCompiler)
		require.True(t, ok)
		mc.CompileFunc = func(_ string) (validator.Validator, error) {
			return &mockValidator{
				Err: nil, // Always pass validation
			}, nil
		}

		report, err := tr.TestSingleSchema(context.Background(), k)
		require.NoError(t, err)
		assert.NotNil(t, report)
		// 1 pass test passed, 1 fail test FAILED (unexpectedly passed validation)
		assert.Len(t, report.PassedTests[k], 1)
		assert.Len(t, report.FailedTests[k], 1)
	})

	// Note: invalid path/key resolution is now tested in arg_resolver_test.go
}

func TestTester_TestFoundSchemas(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	k1 := Key("domain1_family_1_0_0")
	k2 := Key("domain2_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k1: `{"type": "object"}`,
		k2: `{"type": "object"}`,
	})

	tr := NewTester(r)
	tr.SetStopOnFirstError(false)

	// Ensure pass/fail dirs exist for both schemas
	for _, k := range []Key{k1, k2} {
		s, _ := r.GetSchemaByKey(k)
		require.NoError(t, os.MkdirAll(filepath.Join(s.Path(HomeDir), string(TestDocTypePass)), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(s.Path(HomeDir), string(TestDocTypeFail)), 0o755))
	}

	report, err := tr.TestFoundSchemas(context.Background(), "domain1")
	require.NoError(t, err)
	assert.NotNil(t, report)
}

func TestTester_testSchema_Errs(t *testing.T) {
	t.Parallel()

	t.Run("schema not found", func(t *testing.T) {
		t.Parallel()
		r := setupTestRegistry(t)
		tr := NewTester(r)
		err := tr.testSchema(context.Background(), Key("missing_family_1_0_0"))
		require.Error(t, err)
	})

	t.Run("render error", func(t *testing.T) {
		t.Parallel()
		r := setupTestRegistry(t)
		k := Key("domain_family_1_0_0")
		createSchemaFiles(t, r, schemaMap{
			k: `{"type": "object"}`,
		})
		tr := NewTester(r)

		mc, ok := r.compiler.(*mockCompiler)
		require.True(t, ok)
		mc.CompileFunc = func(_ string) (validator.Validator, error) {
			return nil, errors.New("render failure")
		}
		err := tr.testSchema(context.Background(), k)
		require.Error(t, err)
		assert.ErrorContains(t, err, "render failure")
	})

	t.Run("getSpecs error", func(t *testing.T) {
		t.Parallel()
		r := setupTestRegistry(t)
		k := Key("domain_family_1_0_0")
		createSchemaFiles(t, r, schemaMap{
			k: `{"type": "object"}`,
		})
		tr := NewTester(r)

		s, _ := r.GetSchemaByKey(k)
		// Restore valid schema but trigger an error in getSpecsForSchema by removing the home directory.
		require.NoError(t, os.WriteFile(s.Path(FilePath), []byte(`{"type": "object"}`), 0o600))
		os.RemoveAll(s.Path(HomeDir))
		err := tr.testSchema(context.Background(), k)
		require.Error(t, err)
	})
}

func TestTester_TestFoundSchemas_ErrInTraversal(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	tr := NewTester(r)

	// Invalid search scope triggering error in NewSearcher
	_, err := tr.TestFoundSchemas(context.Background(), "!!!")
	require.Error(t, err)
}

func TestTester_StopOnFirstErr(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})

	s, _ := r.GetSchemaByKey(k)
	passDir := filepath.Join(s.Path(HomeDir), string(TestDocTypePass))
	failDir := filepath.Join(s.Path(HomeDir), string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(passDir, 0o755))
	require.NoError(t, os.MkdirAll(failDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "test1.json"), []byte("{}"), 0o600))

	tr := NewTester(r)
	tr.SetStopOnFirstError(true)

	// mockValidator that fails validation
	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return &mockValidator{
			Err: errors.New("validation failed"),
		}, nil
	}

	report, err := tr.TestSingleSchema(context.Background(), k)
	require.NoError(t, err)
	assert.Len(t, report.FailedTests[k], 1)
}

func TestTester_ContextCancellation_FoundSchemas(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})

	tr := NewTester(r)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := tr.TestFoundSchemas(ctx, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestTester_GetSpecsErrs(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})
	s, _ := r.GetSchemaByKey(k)
	passDir := filepath.Join(s.Path(HomeDir), string(TestDocTypePass))
	failDir := filepath.Join(s.Path(HomeDir), string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(passDir, 0o755))
	require.NoError(t, os.MkdirAll(failDir, 0o755))
	// Create an invalid JSON test document
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "invalid.json"), []byte("{"), 0o600))

	tr := NewTester(r)
	_, err := tr.TestSingleSchema(context.Background(), k)
	require.Error(t, err)
	assert.IsType(t, InvalidTestDocumentError{}, err)
}

func TestTester_TestFoundSchemas_StopOnErr(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k1 := Key("domain_family1_1_0_0")
	k2 := Key("domain_family2_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k1: `{"type": "object"}`,
		k2: `{"type": "object"}`,
	})

	// Make k1 have a pass test that fails
	s1, _ := r.GetSchemaByKey(k1)
	passDir1 := filepath.Join(s1.Path(HomeDir), string(TestDocTypePass))
	failDir1 := filepath.Join(s1.Path(HomeDir), string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(passDir1, 0o755))
	require.NoError(t, os.MkdirAll(failDir1, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(passDir1, "test.json"), []byte("{}"), 0o600))

	// Also for k2
	s2, _ := r.GetSchemaByKey(k2)
	require.NoError(t, os.MkdirAll(filepath.Join(s2.Path(HomeDir), string(TestDocTypePass)), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(s2.Path(HomeDir), string(TestDocTypeFail)), 0o755))

	tr := NewTester(r)
	tr.SetStopOnFirstError(true)
	tr.SetNumWorkers(1) // Force sequential for predictable stop

	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return &mockValidator{
			Err: errors.New("fail"),
		}, nil
	}

	report, err := tr.TestFoundSchemas(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, report.FailedTests, 1) // Should stop after first failure
}

func TestTester_ScopedTests(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})
	s, _ := r.GetSchemaByKey(k)
	passDir := filepath.Join(s.Path(HomeDir), string(TestDocTypePass))
	failDir := filepath.Join(s.Path(HomeDir), string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(passDir, 0o755))
	require.NoError(t, os.MkdirAll(failDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "pass.json"), []byte("{}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(failDir, "fail.json"), []byte("[]"), 0o600))

	// Create a future version in the same family with a pass test
	kFuture := Key("domain_family_1_1_0")
	createSchemaFiles(t, r, schemaMap{
		kFuture: `{"type": "object"}`,
	})
	sFuture, _ := r.GetSchemaByKey(kFuture)
	passDirFuture := filepath.Join(sFuture.Path(HomeDir), string(TestDocTypePass))
	require.NoError(t, os.MkdirAll(passDirFuture, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(passDirFuture, "future_pass.json"), []byte("{}"), 0o600))

	t.Run("Pass only", func(t *testing.T) {
		t.Parallel()
		tr := NewTester(r)
		tr.SetScope(TestScopePass)
		specs, err := tr.getSpecsForSchema(s)
		require.NoError(t, err)
		assert.Len(t, specs, 1)
		assert.Equal(t, TestDocTypePass, specs[0].TestDocType)
	})

	t.Run("Fail only", func(t *testing.T) {
		t.Parallel()
		tr := NewTester(r)
		tr.SetScope(TestScopeFail)
		specs, err := tr.getSpecsForSchema(s)
		require.NoError(t, err)
		assert.Len(t, specs, 1)
		assert.Equal(t, TestDocTypeFail, specs[0].TestDocType)
	})

	t.Run("Breaking", func(t *testing.T) {
		t.Parallel()
		tr := NewTester(r)
		tr.SetScope(TestScopeConsumerBreaking)
		specs, err := tr.getSpecsForSchema(s)
		require.NoError(t, err)
		assert.Len(t, specs, 1)
		assert.Equal(t, TestDocTypePass, specs[0].TestDocType)
		assert.Equal(t, "future_pass.json", filepath.Base(specs[0].TestInfo.Path))
		// Check that ForwardVersion is set for breaking tests
		require.NotNil(t, specs[0].ForwardVersion)
		assert.Equal(t, sFuture.core.version, *specs[0].ForwardVersion)
	})

	t.Run("All", func(t *testing.T) {
		t.Parallel()
		tr := NewTester(r)
		tr.SetScope(TestScopeAll)
		specs, err := tr.getSpecsForSchema(s)
		require.NoError(t, err)
		// Should have: 1 local pass, 1 local fail, 1 breaking pass
		assert.Len(t, specs, 3)

		var breakingCount, localPassCount, localFailCount int
		for _, spec := range specs {
			switch {
			case spec.ForwardVersion != nil:
				breakingCount++
				assert.Equal(t, TestDocTypePass, spec.TestDocType)
			case spec.TestDocType == TestDocTypePass:
				localPassCount++
			case spec.TestDocType == TestDocTypeFail:
				localFailCount++
			}
		}
		assert.Equal(t, 1, breakingCount)
		assert.Equal(t, 1, localPassCount)
		assert.Equal(t, 1, localFailCount)
	})
}

func TestTester_TestFoundSchemas_SearchErr(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	// Create a file with invalid structure to trigger search error in searcher.go
	regDir := r.rootDirectory
	require.NoError(t, os.MkdirAll(filepath.Join(regDir, "domain"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(regDir, "domain", "invalid-name.schema.json"), []byte("{}"), 0o600))

	tr := NewTester(r)
	_, err := tr.TestFoundSchemas(context.Background(), "domain")
	require.Error(t, err)
	assert.IsType(t, &InvalidSchemaFilenameError{}, err)
}

func TestTester_testSchema_ContextCancelled(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})
	s, _ := r.GetSchemaByKey(k)
	passDir := filepath.Join(s.Path(HomeDir), string(TestDocTypePass))
	failDir := filepath.Join(s.Path(HomeDir), string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(passDir, 0o755))
	require.NoError(t, os.MkdirAll(failDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "p1.json"), []byte("{}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "p2.json"), []byte("{}"), 0o600))

	tr := NewTester(r)
	ctx, cancel := context.WithCancel(context.Background())

	// mockValidator that cancels context when called
	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return &mockValidator{
			Err: nil,
		}, nil
	}

	// We want to trigger ctx.Error() check in the loop of testSchema.
	// We'll use a hack: first spec runs, we cancel context.
	// But testSchema doesn't call a hook.
	// However, we can make the mockValidator cancel the context!
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return &mockValidator{
			ValidateFunc: func(_ validator.JSONDocument) error {
				cancel()
				return nil
			},
		}, nil
	}

	err := tr.testSchema(ctx, k)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestTester_getSpecsForSchema_ScopedErrs(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})
	s, _ := r.GetSchemaByKey(k)

	t.Run("Fail directory error", func(t *testing.T) {
		t.Parallel()
		tr := NewTester(r)
		tr.SetScope(TestScopeFail)
		// Don't create 'fail' directory
		_, err := tr.getSpecsForSchema(s)
		require.Error(t, err)
	})

	t.Run("Breaking schema missing", func(t *testing.T) {
		t.Parallel()
		tr := NewTester(r)
		tr.SetScope(TestScopeConsumerBreaking)

		// Create directory structure for future version but NO schema file
		// 1.0.0 is 's'. Family dir is .../family
		// Create .../family/1/1/0
		famDir := s.Path(FamilyDir)
		futureDir := filepath.Join(famDir, "1", "1", "0")
		require.NoError(t, os.MkdirAll(futureDir, 0o755))

		_, err := tr.getSpecsForSchema(s)
		require.Error(t, err)
		// Error should come from GetSchemaByKey failing to find domain_family_1_1_0.schema.json
	})

	t.Run("Breaking specs error", func(t *testing.T) { //nolint:paralleltest // uses temp file
		tr := NewTester(r)
		tr.SetScope(TestScopeConsumerBreaking)

		// Create a future version but DON'T create the pass directory.
		// This will cause TestDocuments(TestDocTypePass) to fail with TestDirMissingError
		// when appendBreakingSpecs tries to collect pass tests from the future schema.
		kFuture := Key("domain_family_1_2_0") // Use 1.2.0 to avoid conflict with other tests
		createSchemaFiles(t, r, schemaMap{
			kFuture: `{"type": "object"}`,
		})
		// Note: we intentionally do NOT create the pass directory for kFuture

		_, err := tr.getSpecsForSchema(s)
		require.Error(t, err)
		// Error is either TestDirMissingError or fs.PathError depending on timing
	})

	t.Run("Breaking family error", func(t *testing.T) {
		t.Parallel()
		k2 := Key("d2_f2_1_0_0")
		createSchemaFiles(t, r, schemaMap{
			k2: `{"type": "object"}`,
		})
		s2, _ := r.GetSchemaByKey(k2)

		// Make family directory a file to cause ReadDir error
		famDir := s2.Path(FamilyDir)
		// We need to simulate the error in MajorFamilyFutureSchemas
		// It reads the family directory. If we replace the family directory with a file?
		// But s2.Path depends on family name.
		// We can't rename the directory easily while s2 exists?
		// Actually, MajorFamilyFutureSchemas reads s.Path(FamilyDir).
		// If we delete the directory and replace with file?
		require.NoError(t, os.RemoveAll(famDir))
		require.NoError(t, os.WriteFile(famDir, []byte(""), 0o600))
		t.Cleanup(func() { require.NoError(t, os.Remove(famDir)) })

		tr := NewTester(r)
		tr.SetScope(TestScopeConsumerBreaking)
		_, err := tr.getSpecsForSchema(s2)
		require.Error(t, err)
	})

	t.Run("Breaking multiple future", func(t *testing.T) { //nolint:paralleltest // uses temp file
		kBase := Key("d2_f2_1_0_0")
		kNext := Key("d2_f2_1_1_0")
		kLast := Key("d2_f2_1_2_0")

		createSchemaFiles(t, r, schemaMap{
			kBase: `{"type": "object"}`,
			kNext: `{"type": "object"}`,
			kLast: `{"type": "object"}`,
		})

		// Setup pass tests for all
		for _, k := range []Key{kNext, kLast} {
			sn, _ := r.GetSchemaByKey(k)
			passDir := filepath.Join(sn.Path(HomeDir), string(TestDocTypePass))
			require.NoError(t, os.MkdirAll(passDir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(passDir, "pass.json"), []byte("{}"), 0o600))
		}

		sBase, _ := r.GetSchemaByKey(kBase)
		tr := NewTester(r)
		tr.SetScope(TestScopeConsumerBreaking)

		specs, err := tr.getSpecsForSchema(sBase)
		require.NoError(t, err)
		assert.Len(t, specs, 2)
	})
}

func TestTester_TestLog_Initialisation(t *testing.T) {
	t.Parallel()
	l := NewTestLog()
	assert.NotNil(t, l)
}

func TestTester_TestFoundSchemas_NoStopOnErr(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k1 := Key("d1_f1_1_0_0")
	k2 := Key("d1_f2_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k1: `{"type": "object"}`,
		k2: `{"type": "object"}`,
	})

	// Make both have tests
	for _, k := range []Key{k1, k2} {
		s, _ := r.GetSchemaByKey(k)
		p := filepath.Join(s.Path(HomeDir), string(TestDocTypePass))
		err := os.MkdirAll(p, 0o755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "t.json"), []byte("{}"), 0o600)
		if err != nil {
			t.Fatal(err)
		}
		// Ensure fail dir exists too
		err = os.MkdirAll(filepath.Join(s.Path(HomeDir), string(TestDocTypeFail)), 0o755)
		if err != nil {
			t.Fatal(err)
		}
	}

	tr := NewTester(r)
	tr.SetStopOnFirstError(false)

	// Mock failure for all
	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return &mockValidator{Err: errors.New("boom")}, nil
	}

	report, err := tr.TestFoundSchemas(context.Background(), "")
	require.NoError(t, err) // Should not error when stopOnFirstError is false
	assert.Len(t, report.FailedTests, 2)
}

func TestTester_TestFoundSchemas_ProducerBreakOnCancel(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	// Create two schemas
	k1 := Key("d_f1_1_0_0")
	k2 := Key("d_f2_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k1: `{"type": "object"}`,
		k2: `{"type": "object"}`,
	})

	for _, k := range []Key{k1, k2} {
		s, _ := r.GetSchemaByKey(k)
		p := filepath.Join(s.Path(HomeDir), string(TestDocTypePass))
		err := os.MkdirAll(p, 0o755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "t.json"), []byte("{}"), 0o600)
		if err != nil {
			t.Fatal(err)
		}
		err = os.MkdirAll(filepath.Join(s.Path(HomeDir), string(TestDocTypeFail)), 0o755)
		if err != nil {
			t.Fatal(err)
		}
	}

	tr := NewTester(r)
	tr.SetStopOnFirstError(true)
	tr.SetNumWorkers(1)

	proceed := make(chan struct{})

	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return &mockValidator{
			ValidateFunc: func(_ validator.JSONDocument) error {
				<-proceed
				return errors.New("fail")
			},
		}, nil
	}

	// Run TestFoundSchemas in a goroutine
	type result struct {
		report *TestReport
		err    error
	}
	resC := make(chan result, 1)
	go func() {
		report, err := tr.TestFoundSchemas(context.Background(), "")
		resC <- result{report, err}
	}()

	// Wait a short time to ensure the first worker has started and the producer
	// is now blocked trying to acquire the semaphore for the second schema.
	time.Sleep(100 * time.Millisecond)

	// Let the worker proceed and fail
	close(proceed)

	res := <-resC
	require.NoError(t, res.err)
	// One schema should have recorded a failure, the other might not have run or been cancelled
	assert.NotEmpty(t, res.report.FailedTests)
}

func TestTester_TestFoundSchemas_WorkerErr(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k := Key("d_f_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})

	tr := NewTester(r)

	// Force a render error
	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return nil, errors.New("render failure")
	}

	_, err := tr.TestFoundSchemas(context.Background(), "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "render failure")
}

func TestTester_TestFoundSchemas_ContextCancelledDuringExecution(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k := Key("d_f_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})

	tr := NewTester(r)
	tr.SetNumWorkers(1)

	ctx, cancel := context.WithCancel(context.Background())

	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return &mockValidator{
			ValidateFunc: func(_ validator.JSONDocument) error {
				cancel() // Cancel the parent context
				return nil
			},
		}, nil
	}

	// Need a test document to actually run validation
	s, _ := r.GetSchemaByKey(k)
	p := filepath.Join(s.Path(HomeDir), string(TestDocTypePass))
	require.NoError(t, os.MkdirAll(p, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(p, "t.json"), []byte("{}"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(s.Path(HomeDir), string(TestDocTypeFail)), 0o755))

	_, err := tr.TestFoundSchemas(ctx, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestNewTestScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    TestScope
		wantErr bool
	}{
		{
			name:    "local",
			input:   "local",
			want:    TestScopeLocal,
			wantErr: false,
		},
		{
			name:    "pass-only",
			input:   "pass-only",
			want:    TestScopePass,
			wantErr: false,
		},
		{
			name:    "fail-only",
			input:   "fail-only",
			want:    TestScopeFail,
			wantErr: false,
		},
		{
			name:    "consumer-breaking",
			input:   "consumer-breaking",
			want:    TestScopeConsumerBreaking,
			wantErr: false,
		},
		{
			name:    "all",
			input:   "all",
			want:    TestScopeAll,
			wantErr: false,
		},
		{
			name:    "invalid",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewTestScope(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.IsType(t, &InvalidTestScopeError{}, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestErrs_Strings(t *testing.T) {
	t.Parallel()
	assert.Contains(t, (&NoSchemaTargetsError{}).Error(), "must specify a schema")
	assert.Contains(t, (&InvalidTestScopeError{Scope: "foo"}).Error(), "Invalid test scope: foo")
}

func TestTester_SetSkipCompatible(t *testing.T) {
	t.Parallel()
	tr := NewTester(&Registry{})
	assert.False(t, tr.skipCompatible)
	tr.SetSkipCompatible(true)
	assert.True(t, tr.skipCompatible)
}

func TestTester_ProviderCompatibility(t *testing.T) { //nolint:paralleltest // shared state across subtests
	// Create a test registry with multiple schema versions
	r := setupTestRegistry(t)

	// Create two versions: 1.0.0 (earlier) and 1.0.1 (target)
	earlierKey := Key("domain_family_1_0_0")
	targetKey := Key("domain_family_1_0_1")

	// Create schema files
	createSchemaFiles(t, r, schemaMap{
		earlierKey: `{"type": "object", "required": ["name"]}`,
		targetKey:  `{"type": "object"}`,
	})

	// Create pass/fail test directories
	targetSchema, _ := r.GetSchemaByKey(targetKey)
	targetHome := targetSchema.Path(HomeDir)
	passDir := filepath.Join(targetHome, string(TestDocTypePass))
	failDir := filepath.Join(targetHome, string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(passDir, 0o755))
	require.NoError(t, os.MkdirAll(failDir, 0o755))

	earlierSchema, _ := r.GetSchemaByKey(earlierKey)
	earlierHome := earlierSchema.Path(HomeDir)
	earlierPassDir := filepath.Join(earlierHome, string(TestDocTypePass))
	earlierFailDir := filepath.Join(earlierHome, string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(earlierPassDir, 0o755))
	require.NoError(t, os.MkdirAll(earlierFailDir, 0o755))

	// Create a pass test for the target schema that would FAIL against the earlier schema
	// (target allows empty objects, but earlier requires "name" property)
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "valid.json"), []byte(`{}`), 0o600))

	// Configure the mock compiler to return failing validator for the earlier schema
	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok, "compiler must be mockCompiler")
	mc.CompileFunc = func(id string) (validator.Validator, error) {
		// Match the earlier schema by checking if the ID contains the earlier key
		if strings.Contains(id, string(earlierKey)) {
			// Earlier schema fails validation (simulating stricter requirements)
			return &mockValidator{Err: errors.New("required property 'name' missing")}, nil
		}
		// Target schema passes validation
		return &mockValidator{Err: nil}, nil
	}

	t.Run("compatibility check detects breaking change", func(t *testing.T) { //nolint:paralleltest // shared state
		tr := NewTester(r)
		tr.SetStopOnFirstError(false)
		tr.SetSkipCompatible(false)

		report, err := tr.TestSingleSchema(context.Background(), targetKey)
		require.NoError(t, err)
		require.NotNil(t, report)

		// The target schema's pass test should have been run against earlier schema and failed
		assert.NotEmpty(t, report.FailedTests[earlierKey], "earlier schema should have failed tests")
	})

	t.Run("skip compatibility check", func(t *testing.T) { //nolint:paralleltest // shared state
		tr := NewTester(r)
		tr.SetStopOnFirstError(false)
		tr.SetSkipCompatible(true)

		report, err := tr.TestSingleSchema(context.Background(), targetKey)
		require.NoError(t, err)
		require.NotNil(t, report)

		// When compatibility is skipped, earlier schema should not be tested
		assert.Empty(t, report.FailedTests[earlierKey], "no earlier schema tests when skipped")
	})

	t.Run("context cancellation stops compatibility check", func(t *testing.T) { //nolint:paralleltest // shared state
		tr := NewTester(r)
		tr.SetStopOnFirstError(false)
		tr.SetSkipCompatible(false)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := tr.TestSingleSchema(ctx, targetKey)
		// Should return without error but may have incomplete results
		assert.True(t, err == nil || errors.Is(err, context.Canceled))
	})

	t.Run("stop on first error in compatibility check", func(t *testing.T) { //nolint:paralleltest // shared state
		tr := NewTester(r)
		tr.SetStopOnFirstError(true)
		tr.SetSkipCompatible(false)

		report, err := tr.TestSingleSchema(context.Background(), targetKey)
		require.NoError(t, err)
		require.NotNil(t, report)

		// Should stop after first failure
		assert.NotEmpty(t, report.FailedTests[earlierKey])
	})
}

func TestTester_ProviderCompatibility_NoEarlierSchemas(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	// Create a single version - no earlier versions to check against
	targetKey := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		targetKey: `{"type": "object"}`,
	})

	targetSchema, _ := r.GetSchemaByKey(targetKey)
	targetHome := targetSchema.Path(HomeDir)
	passDir := filepath.Join(targetHome, string(TestDocTypePass))
	failDir := filepath.Join(targetHome, string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(passDir, 0o755))
	require.NoError(t, os.MkdirAll(failDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "valid.json"), []byte(`{}`), 0o600))

	tr := NewTester(r)
	tr.SetSkipCompatible(false)

	report, err := tr.TestSingleSchema(context.Background(), targetKey)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Should succeed with no compatibility issues (no earlier versions)
	assert.Len(t, report.PassedTests[targetKey], 1)
	assert.Empty(t, report.FailedTests)
}

func TestTester_ProviderCompatibility_NoPassTests(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	// Create two versions but no pass tests
	earlierKey := Key("domain_family_1_0_0")
	targetKey := Key("domain_family_1_0_1")

	createSchemaFiles(t, r, schemaMap{
		earlierKey: `{"type": "object"}`,
		targetKey:  `{"type": "object"}`,
	})

	for _, k := range []Key{earlierKey, targetKey} {
		s, _ := r.GetSchemaByKey(k)
		home := s.Path(HomeDir)
		require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypePass)), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypeFail)), 0o755))
	}

	tr := NewTester(r)
	tr.SetSkipCompatible(false)

	report, err := tr.TestSingleSchema(context.Background(), targetKey)
	require.NoError(t, err)
	require.NotNil(t, report)

	// No pass tests means nothing to check
	assert.Empty(t, report.FailedTests)
}

func TestTester_ProviderCompatibility_LocalTestsFail(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	// Create two versions
	earlierKey := Key("domain_family_1_0_0")
	targetKey := Key("domain_family_1_0_1")

	createSchemaFiles(t, r, schemaMap{
		earlierKey: `{"type": "object"}`,
		targetKey:  `{"type": "object"}`,
	})

	targetSchema, _ := r.GetSchemaByKey(targetKey)
	targetHome := targetSchema.Path(HomeDir)
	passDir := filepath.Join(targetHome, string(TestDocTypePass))
	failDir := filepath.Join(targetHome, string(TestDocTypeFail))
	require.NoError(t, os.MkdirAll(passDir, 0o755))
	require.NoError(t, os.MkdirAll(failDir, 0o755))

	earlierSchema, _ := r.GetSchemaByKey(earlierKey)
	earlierHome := earlierSchema.Path(HomeDir)
	require.NoError(t, os.MkdirAll(filepath.Join(earlierHome, string(TestDocTypePass)), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(earlierHome, string(TestDocTypeFail)), 0o755))

	// Create a FAILING pass test for target schema (invalid JSON)
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "invalid.json"), []byte(`"not an object"`), 0o600))

	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return &mockValidator{
			Err: errors.New("validation failed"),
		}, nil
	}

	tr := NewTester(r)
	tr.SetStopOnFirstError(false)
	tr.SetSkipCompatible(false)

	report, err := tr.TestSingleSchema(context.Background(), targetKey)
	require.NoError(t, err)
	require.NotNil(t, report)

	// When local tests fail, compatibility check should not run
	// So only target schema failures are present
	assert.NotEmpty(t, report.FailedTests[targetKey])
	assert.Empty(t, report.FailedTests[earlierKey])
}

func TestTester_ProviderCompatibility_RenderError(t *testing.T) { //nolint:paralleltest // shared state
	r := setupTestRegistry(t)

	earlierKey := Key("domain_family_1_0_0")
	targetKey := Key("domain_family_1_0_1")

	createSchemaFiles(t, r, schemaMap{
		earlierKey: `{"type": "object"}`,
		targetKey:  `{"type": "object"}`,
	})

	for _, k := range []Key{earlierKey, targetKey} {
		s, _ := r.GetSchemaByKey(k)
		home := s.Path(HomeDir)
		require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypePass)), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypeFail)), 0o755))
	}

	// Create a pass test for the target
	targetSchema, _ := r.GetSchemaByKey(targetKey)
	passDir := filepath.Join(targetSchema.Path(HomeDir), "pass")
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "valid.json"), []byte(`{}`), 0o600))

	// Configure compiler to fail when compiling the earlier schema
	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(id string) (validator.Validator, error) {
		if strings.Contains(id, string(earlierKey)) {
			return nil, errors.New("compile error")
		}
		return &mockValidator{}, nil
	}

	tr := NewTester(r)
	tr.SetSkipCompatible(false)

	_, err := tr.TestSingleSchema(context.Background(), targetKey)
	// Should return the compile error since it's a real error, not a test failure
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile error")
}

func TestTester_ProviderCompatibility_PassTestsPass(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	earlierKey := Key("domain_family_1_0_0")
	targetKey := Key("domain_family_1_0_1")

	createSchemaFiles(t, r, schemaMap{
		earlierKey: `{"type": "object"}`,
		targetKey:  `{"type": "object"}`,
	})

	for _, k := range []Key{earlierKey, targetKey} {
		s, _ := r.GetSchemaByKey(k)
		home := s.Path(HomeDir)
		require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypePass)), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypeFail)), 0o755))
	}

	// Create a pass test for the target that will also pass against earlier
	targetSchema, _ := r.GetSchemaByKey(targetKey)
	passDir := filepath.Join(targetSchema.Path(HomeDir), "pass")
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "valid.json"), []byte(`{}`), 0o600))

	// Both schemas accept the test
	mc, ok := r.compiler.(*mockCompiler)
	require.True(t, ok)
	mc.CompileFunc = func(_ string) (validator.Validator, error) {
		return &mockValidator{Err: nil}, nil
	}

	tr := NewTester(r)
	tr.SetSkipCompatible(false)

	report, err := tr.TestSingleSchema(context.Background(), targetKey)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Both should have passed tests
	assert.Len(t, report.PassedTests[targetKey], 1)
	assert.Len(t, report.PassedTests[earlierKey], 1)
	assert.Empty(t, report.FailedTests)
}

// TestTester_testSchemaCompatibleWithEarlierVersions_ErrorPaths tests error paths
// in the compatibility checking functions that are difficult to reach through
// normal TestSingleSchema flow.
func TestTester_testSchemaCompatibleWithEarlierVersions_ErrorPaths(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	// Create test schema with pass tests
	targetKey := Key("domain_family_1_0_1")
	earlierKey := Key("domain_family_1_0_0")

	createSchemaFiles(t, r, schemaMap{
		earlierKey: `{"type": "object"}`,
		targetKey:  `{"type": "object"}`,
	})

	for _, k := range []Key{earlierKey, targetKey} {
		s, _ := r.GetSchemaByKey(k)
		home := s.Path(HomeDir)
		require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypePass)), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypeFail)), 0o755))
	}

	// Create pass test
	targetSchema, _ := r.GetSchemaByKey(targetKey)
	passDir := filepath.Join(targetSchema.Path(HomeDir), "pass")
	require.NoError(t, os.WriteFile(filepath.Join(passDir, "valid.json"), []byte(`{}`), 0o600))

	t.Run("GetSchemaByKey error", func(t *testing.T) {
		t.Parallel()
		tr := NewTester(r)
		// Call with an invalid key that doesn't exist
		err := tr.testSchemaCompatibleWithEarlierVersions(context.Background(), Key("invalid_key_1_0_0"))
		require.Error(t, err)
	})

	t.Run("TestDocuments error - pass directory is file not dir", func(t *testing.T) {
		t.Parallel()
		r2 := setupTestRegistry(t)
		key := Key("domain_family_1_0_0")
		createSchemaFiles(t, r2, schemaMap{
			key: `{"type": "object"}`,
		})

		s, _ := r2.GetSchemaByKey(key)
		home := s.Path(HomeDir)

		// Create fail directory but make pass directory a FILE instead of directory
		require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypeFail)), 0o755))
		// This creates a FILE named "pass" instead of a directory
		require.NoError(t, os.WriteFile(filepath.Join(home, string(TestDocTypePass)), []byte(""), 0o600))

		tr := NewTester(r2)
		err := tr.testSchemaCompatibleWithEarlierVersions(context.Background(), key)
		require.Error(t, err)
	})

	t.Run("MajorFamilyEarlierSchemas error", func(t *testing.T) {
		t.Parallel()
		r2 := setupTestRegistry(t)
		key := Key("domain_family_1_0_1")
		createSchemaFiles(t, r2, schemaMap{
			key: `{"type": "object"}`,
		})

		s, _ := r2.GetSchemaByKey(key)
		home := s.Path(HomeDir)
		subPassDir := filepath.Join(home, string(TestDocTypePass))
		subFailDir := filepath.Join(home, string(TestDocTypeFail))
		require.NoError(t, os.MkdirAll(subPassDir, 0o755))
		require.NoError(t, os.MkdirAll(subFailDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(subPassDir, "valid.json"), []byte(`{}`), 0o600))

		// Pre-cache TestDocuments result BEFORE making directory unreadable
		// This ensures TestDocuments succeeds, but MajorFamilyEarlierSchemas fails
		_, err := s.TestDocuments(TestDocTypePass)
		require.NoError(t, err)

		// Make the major version directory unreadable to cause MajorFamilyEarlierSchemas error
		familyDir := s.Path(FamilyDir)
		majorDir := filepath.Join(familyDir, "1")
		require.NoError(t, os.Chmod(majorDir, 0o000))
		t.Cleanup(func() { _ = os.Chmod(majorDir, 0o755) })

		tr := NewTester(r2)
		err = tr.testSchemaCompatibleWithEarlierVersions(context.Background(), key)
		require.Error(t, err)
	})

	t.Run("testEarlierSchemaWithTargetTests - GetSchemaByKey error", func(t *testing.T) {
		t.Parallel()
		tr := NewTester(r)
		ts, _ := r.GetSchemaByKey(targetKey)
		passTests, _ := ts.TestDocuments(TestDocTypePass)

		// Call with an invalid earlier key
		err := tr.testEarlierSchemaWithTargetTests(
			context.Background(),
			Key("invalid_earlier_1_0_0"),
			targetSchema,
			passTests,
		)
		require.Error(t, err)
	})

	t.Run("testEarlierSchemaWithTargetTests - context cancelled in loop", func(t *testing.T) {
		t.Parallel()

		// Create a pass test file for the target
		mc, ok := r.compiler.(*mockCompiler)
		require.True(t, ok)
		mc.CompileFunc = func(_ string) (validator.Validator, error) {
			return &mockValidator{Err: nil}, nil
		}

		tr := NewTester(r)
		ts, _ := r.GetSchemaByKey(targetKey)
		passTests, _ := ts.TestDocuments(TestDocTypePass)
		require.NotEmpty(t, passTests)

		earlierSchema, _ := r.GetSchemaByKey(earlierKey)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := tr.testEarlierSchemaWithTargetTests(ctx, earlierKey, earlierSchema, passTests)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("context cancelled triggers ctx.Err return", func(t *testing.T) {
		t.Parallel()

		// Set up multiple earlier versions to have multiple loop iterations
		r2 := setupTestRegistry(t)
		keys := []Key{
			Key("domain_family_1_0_0"),
			Key("domain_family_1_0_1"),
			Key("domain_family_1_0_2"),
		}

		createSchemaFiles(t, r2, schemaMap{
			keys[0]: `{"type": "object"}`,
			keys[1]: `{"type": "object"}`,
			keys[2]: `{"type": "object"}`,
		})

		for _, k := range keys {
			s, _ := r2.GetSchemaByKey(k)
			home := s.Path(HomeDir)
			require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypePass)), 0o755))
			require.NoError(t, os.MkdirAll(filepath.Join(home, string(TestDocTypeFail)), 0o755))
		}

		// Create a pass test for the target (the last key)
		ts, _ := r2.GetSchemaByKey(keys[2])
		ctxTestPassDir := filepath.Join(ts.Path(HomeDir), "pass")
		require.NoError(t, os.WriteFile(filepath.Join(ctxTestPassDir, "valid.json"), []byte(`{}`), 0o600))

		// Configure compiler
		mc, ok := r2.compiler.(*mockCompiler)
		require.True(t, ok)
		mc.CompileFunc = func(_ string) (validator.Validator, error) {
			return &mockValidator{Err: nil}, nil
		}

		tr := NewTester(r2)

		// Pass a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := tr.testSchemaCompatibleWithEarlierVersions(ctx, keys[2])
		// Should return ctx.Err() which is context.Canceled
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}
