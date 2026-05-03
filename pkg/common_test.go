package common

import (
	"os"
	"os/exec"
	"strconv"
	"sync"
	"testing"
)

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if id == "" {
		t.Fatal("GenerateID returned empty string")
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			t.Fatalf("GenerateID returned non-numeric string: %s", id)
		}
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID()
		if ids[id] {
			t.Fatalf("GenerateID returned duplicate: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateID_SnowflakeNodeEnv(t *testing.T) {
	if os.Getenv("GO_TEST_SNOWFLAKE_ENV") == "1" {
		id := GenerateID()
		if id == "" {
			os.Exit(1)
		}
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestGenerateID_SnowflakeNodeEnv")
	cmd.Env = append(os.Environ(), "GO_TEST_SNOWFLAKE_ENV=1", "SNOWFLAKE_NODE=99")
	if err := cmd.Run(); err != nil {
		t.Fatalf("subprocess with SNOWFLAKE_NODE=99 failed: %v", err)
	}
}

func TestGenerateID_Fallback(t *testing.T) {
	orig := sfNode
	sfNode = nil
	t.Cleanup(func() { sfNode = orig })

	id := GenerateID()
	if id == "" {
		t.Fatal("GenerateID fallback returned empty string")
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			t.Fatalf("GenerateID fallback returned non-numeric string: %s", id)
		}
	}
}

func TestGenerateID_FallbackConcurrency(t *testing.T) {
	orig := sfNode
	sfNode = nil
	t.Cleanup(func() { sfNode = orig })

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID()
		if ids[id] {
			t.Fatalf("GenerateID fallback returned duplicate: %s", id)
		}
		ids[id] = true
	}
}

func TestWeakRandInt63n(t *testing.T) {
	if got := WeakRandInt63n(0); got != 0 {
		t.Errorf("WeakRandInt63n(0) = %d, want 0", got)
	}
	if got := WeakRandInt63n(-1); got != 0 {
		t.Errorf("WeakRandInt63n(-1) = %d, want 0", got)
	}

	for i := 0; i < 100; i++ {
		got := WeakRandInt63n(10)
		if got < 0 || got >= 10 {
			t.Errorf("WeakRandInt63n(10) = %d, want in [0, 10)", got)
		}
	}
}

func TestSecureRandInt63n(t *testing.T) {
	got, err := SecureRandInt63n(0)
	if err != nil {
		t.Fatalf("SecureRandInt63n(0) error: %v", err)
	}
	if got != 0 {
		t.Errorf("SecureRandInt63n(0) = %d, want 0", got)
	}

	got, err = SecureRandInt63n(-5)
	if err != nil {
		t.Fatalf("SecureRandInt63n(-5) error: %v", err)
	}
	if got != 0 {
		t.Errorf("SecureRandInt63n(-5) = %d, want 0", got)
	}

	for i := 0; i < 10; i++ {
		got, err := SecureRandInt63n(100)
		if err != nil {
			t.Fatalf("SecureRandInt63n(100) error: %v", err)
		}
		if got < 0 || got >= 100 {
			t.Errorf("SecureRandInt63n(100) = %d, want in [0, 100)", got)
		}
	}
}

func TestErrorToString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, ""},
		{"error", &testError{"test error"}, "test error"},
		{"string", "plain string", "plain string"},
		{"int", 42, "42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorToString(tt.in); got != tt.want {
				t.Errorf("ErrorToString(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

type testStruct struct {
	Name  string `json:"name"`
	Age   int    `json:"age,omitempty"`
	Skip  string `json:"-"`
	Email string `json:"email"`
}

func TestStructToMap(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		in := testStruct{Name: "Alice", Age: 25, Skip: "skip_me", Email: "a@b.com"}
		m, err := StructToMap(in, "json")
		if err != nil {
			t.Fatalf("StructToMap error: %v", err)
		}
		if len(m) != 3 {
			t.Fatalf("expected 3 fields, got %d: %v", len(m), m)
		}
		if m["name"] != "Alice" {
			t.Errorf("name = %v, want Alice", m["name"])
		}
		if m["email"] != "a@b.com" {
			t.Errorf("email = %v, want a@b.com", m["email"])
		}
		if _, ok := m["Skip"]; ok {
			t.Error("json:\"-\" field should be skipped")
		}
	})

	t.Run("omitempty", func(t *testing.T) {
		in := testStruct{Name: "Bob", Email: "b@c.com"}
		m, err := StructToMap(in, "json")
		if err != nil {
			t.Fatalf("StructToMap error: %v", err)
		}
		if _, ok := m["age,omitempty"]; ok {
			t.Error("tag options like ,omitempty should be stripped from key")
		}
		if m["name"] != "Bob" {
			t.Errorf("name = %v, want Bob", m["name"])
		}
	})

	t.Run("pointer", func(t *testing.T) {
		in := &testStruct{Name: "Charlie", Email: "c@d.com"}
		m, err := StructToMap(in, "json")
		if err != nil {
			t.Fatalf("StructToMap error: %v", err)
		}
		if m["name"] != "Charlie" {
			t.Errorf("name = %v, want Charlie", m["name"])
		}
	})

	t.Run("nonStruct", func(t *testing.T) {
		_, err := StructToMap("not a struct", "json")
		if err == nil {
			t.Fatal("expected error for non-struct input")
		}
	})

	t.Run("nilInput", func(t *testing.T) {
		_, err := StructToMap(nil, "json")
		if err == nil {
			t.Fatal("expected error for nil input")
		}
	})

	t.Run("noMatchingTag", func(t *testing.T) {
		type s struct {
			Name  string `json:"name"`
			NoTag string
		}
		m, err := StructToMap(s{Name: "test"}, "json")
		if err != nil {
			t.Fatalf("StructToMap error: %v", err)
		}
		if len(m) != 1 {
			t.Fatalf("expected 1 field, got %d: %v", len(m), m)
		}
	})
}

// ========================
// Benchmark
// ========================

func BenchmarkGenerateID(b *testing.B) {
	b.Run("snowflake", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GenerateID()
		}
	})

	b.Run("snowflake_parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				GenerateID()
			}
		})
	})
}

func BenchmarkGenerateID_Fallback(b *testing.B) {
	orig := sfNode
	sfNode = nil
	b.Cleanup(func() { sfNode = orig })

	b.Run("serial", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GenerateID()
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				GenerateID()
			}
		})
	})
}

func BenchmarkWeakRandInt63n(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			WeakRandInt63n(1000)
		}
	})
}

func BenchmarkSecureRandInt63n(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = SecureRandInt63n(1000)
	}
}

func BenchmarkStructToMap(b *testing.B) {
	type benchStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}
	s := benchStruct{Name: "Alice", Age: 25, Email: "a@b.com"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = StructToMap(s, "json")
	}
}

// ========================
// 并发压测
// ========================

func TestGenerateID_ConcurrentStress(t *testing.T) {
	var wg sync.WaitGroup
	ids := make(chan string, 10000)
	concurrency := 50
	perGoroutine := 200

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				ids <- GenerateID()
			}
		}()
	}

	go func() {
		wg.Wait()
		close(ids)
	}()

	seen := make(map[string]bool, concurrency*perGoroutine)
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate ID detected: %s", id)
		}
		seen[id] = true

		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			t.Fatalf("non-numeric ID: %s", id)
		}
	}
	if len(seen) != concurrency*perGoroutine {
		t.Fatalf("expected %d unique IDs, got %d", concurrency*perGoroutine, len(seen))
	}
}

func TestWeakRandInt63n_ConcurrentStress(t *testing.T) {
	var wg sync.WaitGroup
	concurrency := 50
	perGoroutine := 1000

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				got := WeakRandInt63n(1000)
				if got < 0 || got >= 1000 {
					panic("WeakRandInt63n out of range")
				}
			}
		}()
	}
	wg.Wait()
}

// ========================
// Deref
// ========================

func TestDerefOrZero(t *testing.T) {
	t.Run("string nil", func(t *testing.T) {
		if got := DerefOrZero((*string)(nil)); got != "" {
			t.Errorf("DerefOrZero(*string) nil = %q, want empty", got)
		}
	})

	t.Run("string value", func(t *testing.T) {
		s := "test"
		if got := DerefOrZero(&s); got != "test" {
			t.Errorf("DerefOrZero(&string) = %q, want test", got)
		}
	})

	t.Run("int nil", func(t *testing.T) {
		if got := DerefOrZero((*int)(nil)); got != 0 {
			t.Errorf("DerefOrZero(*int) nil = %d, want 0", got)
		}
	})

	t.Run("int value", func(t *testing.T) {
		n := 42
		if got := DerefOrZero(&n); got != 42 {
			t.Errorf("DerefOrZero(&int) = %d, want 42", got)
		}
	})

	t.Run("int32 nil", func(t *testing.T) {
		if got := DerefOrZero((*int32)(nil)); got != 0 {
			t.Errorf("DerefOrZero(*int32) nil = %d, want 0", got)
		}
	})

	t.Run("bool nil", func(t *testing.T) {
		if got := DerefOrZero((*bool)(nil)); got != false {
			t.Errorf("DerefOrZero(*bool) nil = %v, want false", got)
		}
	})

	t.Run("bool value", func(t *testing.T) {
		b := true
		if got := DerefOrZero(&b); !got {
			t.Errorf("DerefOrZero(&bool) = %v, want true", got)
		}
	})
}

func BenchmarkDerefOrZero_String(b *testing.B) {
	s := "hello world"
	for i := 0; i < b.N; i++ {
		_ = DerefOrZero(&s)
	}
}

func BenchmarkDerefOrZero_Int(b *testing.B) {
	n := 42
	for i := 0; i < b.N; i++ {
		_ = DerefOrZero(&n)
	}
}
