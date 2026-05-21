package xcryption_test

import (
	"math"
	"testing"

	"snowgo/pkg/xcryption"
)

func TestId2Code_Boundary(t *testing.T) {
	// === Boundary values ===
	t.Run("boundary: minLength=0 (no padding)", func(t *testing.T) {
		code, err := xcryption.Id2Code(12345, 0)
		if err != nil {
			t.Fatalf("Id2Code minLength=0 error: %v", err)
		}
		if code == "" {
			t.Fatal("code should not be empty")
		}
		decoded, err := xcryption.Code2Id(code)
		if err != nil {
			t.Fatalf("Code2Id error: %v", err)
		}
		if decoded != 12345 {
			t.Fatalf("decoded=%d, want 12345", decoded)
		}
	})

	t.Run("boundary: minLength=1 (id=0)", func(t *testing.T) {
		code, err := xcryption.Id2Code(0, 1)
		if err != nil {
			t.Fatalf("Id2Code id=0 minLength=1 error: %v", err)
		}
		if len(code) < 1 {
			t.Fatalf("code length=%d < minLength=1", len(code))
		}
		decoded, err := xcryption.Code2Id(code)
		if err != nil {
			t.Fatalf("Code2Id error: %v", err)
		}
		if decoded != 0 {
			t.Fatalf("decoded=%d, want 0", decoded)
		}
	})

	t.Run("boundary: minLength shorter than natural code length", func(t *testing.T) {
		// 123456789 encodes to ~6 chars, minLength=3 should still produce ~6 chars
		code, err := xcryption.Id2Code(123456789, 3)
		if err != nil {
			t.Fatalf("Id2Code error: %v", err)
		}
		if len(code) < 6 {
			t.Fatalf("code should be at least natural length (~6), got %d", len(code))
		}
		decoded, err := xcryption.Code2Id(code)
		if err != nil {
			t.Fatalf("Code2Id error: %v", err)
		}
		if decoded != 123456789 {
			t.Fatalf("decoded=%d, want 123456789", decoded)
		}
	})

	t.Run("boundary: very large id near int64 max", func(t *testing.T) {
		bigId := int64(math.MaxInt64 / 100)
		code, err := xcryption.Id2Code(bigId, 10)
		if err != nil {
			t.Fatalf("Id2Code large id error: %v", err)
		}
		decoded, err := xcryption.Code2Id(code)
		if err != nil {
			t.Fatalf("Code2Id error: %v", err)
		}
		if decoded != bigId {
			t.Fatalf("decoded=%d, want %d", decoded, bigId)
		}
	})

	t.Run("boundary: negative minLength (treated as no padding)", func(t *testing.T) {
		code, err := xcryption.Id2Code(42, -5)
		if err != nil {
			t.Fatalf("Id2Code negative minLength error: %v", err)
		}
		decoded, err := xcryption.Code2Id(code)
		if err != nil {
			t.Fatalf("Code2Id error: %v", err)
		}
		if decoded != 42 {
			t.Fatalf("decoded=%d, want 42", decoded)
		}
	})
}

func TestCode2Id_Boundary(t *testing.T) {
	// === Boundary values ===
	t.Run("boundary: empty code returns error", func(t *testing.T) {
		_, err := xcryption.Code2Id("")
		if err == nil {
			t.Fatal("expected error for empty code")
		}
	})

	t.Run("boundary: invalid character returns error", func(t *testing.T) {
		_, err := xcryption.Code2Id("xyz!@#")
		if err == nil {
			t.Fatal("expected error for invalid characters")
		}
	})

	t.Run("boundary: single invalid character", func(t *testing.T) {
		_, err := xcryption.Code2Id("!")
		if err == nil {
			t.Fatal("expected error for single invalid char")
		}
	})

	t.Run("boundary: code with padding separator", func(t *testing.T) {
		// Id2Code(1, 8) produces something like "5iXXXXXXX" where 'i' is divider
		code, _ := xcryption.Id2Code(1, 8)
		decoded, err := xcryption.Code2Id(code)
		if err != nil {
			t.Fatalf("Code2Id padded code error: %v", err)
		}
		if decoded != 1 {
			t.Fatalf("decoded=%d, want 1", decoded)
		}
	})

	t.Run("boundary: very long invalid code", func(t *testing.T) {
		// Long string with one invalid char at the end
		long := "5dfucyar5dfucyar5dfucyar5dfucyar!"
		_, err := xcryption.Code2Id(long)
		if err == nil {
			t.Fatal("expected error for long code with invalid char")
		}
	})

	// === Happy path ===
	t.Run("happy: roundtrip with various minLengths", func(t *testing.T) {
		for _, minLen := range []int{0, 1, 4, 6, 8, 12, 20} {
			code, err := xcryption.Id2Code(999, minLen)
			if err != nil {
				t.Fatalf("Id2Code minLength=%d error: %v", minLen, err)
			}
			if minLen > 0 && len(code) < minLen {
				t.Fatalf("code length=%d < minLength=%d", len(code), minLen)
			}
			decoded, err := xcryption.Code2Id(code)
			if err != nil {
				t.Fatalf("Code2Id minLength=%d error: %v", minLen, err)
			}
			if decoded != 999 {
				t.Fatalf("minLength=%d: decoded=%d, want 999", minLen, decoded)
			}
		}
	})
}

func TestId2CodeNegative(t *testing.T) {
	// === Expected errors ===
	t.Run("error: negative id", func(t *testing.T) {
		_, err := xcryption.Id2Code(-1, 6)
		if err == nil {
			t.Fatal("expected error for negative id")
		}
	})

	t.Run("error: large negative id", func(t *testing.T) {
		_, err := xcryption.Id2Code(-999999, 6)
		if err == nil {
			t.Fatal("expected error for large negative id")
		}
	})
}

func TestCode2Id_Overflow(t *testing.T) {
	// Craft a code that decodes to a value exceeding int64 max.
	// The base-32 encoding of math.MaxInt64 is roughly "jzzzzzzzzzzz",
	// so decoding a longer string of max-value chars should overflow.
	t.Run("error: overflow detection", func(t *testing.T) {
		// base-32: charLen=32. Each char shifts left by 5 bits.
		// A string of 13 'z' characters (index=31 each) should overflow int64.
		_, err := xcryption.Code2Id("zzzzzzzzzzzzz")
		if err == nil {
			t.Fatal("expected overflow error for 13 'z' chars")
		}
	})
}
