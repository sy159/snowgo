package xcryption_test

import (
	"math/rand"
	"snowgo/pkg/xcryption"
	"sync"
	"testing"
	"time"
)

func TestId2CodeAndCode2Id(t *testing.T) {
	tests := []struct {
		id        uint
		minLength int
	}{
		{0, 6},
		{1, 6},
		{10, 6},
		{12345, 8},
		{999999, 10},
		{123456789, 12},
	}

	for _, test := range tests {
		testId := int64(test.id)
		code := xcryption.Id2Code(testId, test.minLength)
		if len(code) < test.minLength {
			t.Fatalf("code length %d < minLength %d", len(code), test.minLength)
		}

		decoded, err := xcryption.Code2Id(code)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if decoded != testId {
			t.Fatalf("decoded id %d != original id %d", decoded, test.id)
		}
		t.Logf("id=%d -> code=%s -> decoded=%d", test.id, code, decoded)
	}
}

func BenchmarkId2CodeConcurrent(b *testing.B) {
	concurrency := 50 // 并发 goroutine 数
	total := 100000   // 总生成数量

	var wg sync.WaitGroup
	errCount := 0
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < total/concurrency; j++ {
				id := int64(rand.Intn(1000000))
				minLen := 6 + rand.Intn(6) // 随机 minLength 6~11
				code := xcryption.Id2Code(id, minLen)
				decoded, err := xcryption.Code2Id(code)
				if err != nil {
					errCount++
				} else if decoded != id {
					errCount++
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)
	b.Logf("total=%d, errors=%d, duration=%s, TPS=%.2f", total, errCount, duration, float64(total)/duration.Seconds())
	if errCount > 0 {
		b.Fatal("some codes failed to decode correctly")
	}
}

func TestRandomStress(t *testing.T) {
	concurrency := 20
	total := 50000
	var wg sync.WaitGroup
	errCount := 0

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < total/concurrency; j++ {
				id := int64(rand.Intn(1_000_000_000))
				minLen := 6 + rand.Intn(12)
				code := xcryption.Id2Code(id, minLen)
				decoded, err := xcryption.Code2Id(code)
				if err != nil || decoded != id {
					errCount++
				}
			}
		}()
	}

	wg.Wait()
	if errCount > 0 {
		t.Fatalf("random stress test failed with %d errors", errCount)
	}
	t.Logf("random stress test passed for %d iterations", total)
}
