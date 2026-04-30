// Order-Preserving and Range-Query Encryption using AES-GCM.
//
// CRITICAL SECURITY WARNING
// Order-preserving encryption (OPE) leaks order, approximate values, distribution, and frequency information.
// Only use for LOW-SENSITIVITY data where range queries are absolutely necessary.

package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
)

func NormalizeInt64(v int64) uint64 {
	// from int64 range [-2^63, 2^63-1] to uint64 range [0, 2^64-1]
	// XOR with the sign bit to flip the range: negative numbers (sign bit 1) become smaller, positive numbers (sign bit 0) become larger
	return uint64(v) + (1 << 63)
}

func DenormalizeInt64(v uint64) int64 {
	// from uint64 range [0, 2^64-1] back to int64 range [-2^63, 2^63-1]
	return int64(v - (1 << 63))
}

func NormalizeInt32(v int32) uint64 {
	// from int32 range [-2^31, 2^31-1] to uint64 range [0, 2^64-1]
	// the full uint64 range is not used; values fit in [0, 2^32-1]
	return uint64(uint32(v) ^ (1 << 31))
}

func DenormalizeInt32(v uint64) int32 {
	// from uint64 range [0, 2^64-1] back to int32 range [-2^31, 2^31-1]
	return int32(uint32(v) ^ (1 << 31))
}

func NormalizeFloat64(v float64) uint64 {
	if math.IsNaN(v) {
		panic("NaN cannot be encrypted with OPE")
	}
	bits := math.Float64bits(v)
	if bits>>63 == 0 {
		// positive (including +0.0, +Inf): flip sign bit
		return bits ^ (1 << 63)
	}
	// negative (including -0.0, -Inf): flip all bits
	return ^bits
}

func DenormalizeFloat64(v uint64) float64 {
	var bits uint64
	if v>>63 == 0 {
		// was positive: flip sign bit back
		bits = v ^ (1 << 63)
	} else {
		// was negative: flip all bits back
		bits = ^v
	}
	return math.Float64frombits(bits)
}

func NormalizeFloat32(v float32) uint64 {
	if math.IsNaN(float64(v)) {
		panic("NaN cannot be encrypted with OPE")
	}
	bits := uint64(math.Float32bits(v))
	if bits>>31 == 0 {
		return bits ^ (1 << 31)
	}
	return uint64(^uint32(bits))
}

func DenormalizeFloat32(v uint64) float32 {
	u := uint32(v)
	var bits uint32
	if u>>31 == 1 {
		bits = u ^ (1 << 31)
	} else {
		bits = ^u
	}
	return math.Float32frombits(bits)
}

// Second-level normalization for time values to preserve order and allow range queries while encrypting with OPE.
func NormalizeTime(t time.Time) uint64 {
	return NormalizeTimeSeconds(t)
}

func DenormalizeTime(v uint64) time.Time {
	return DenormalizeTimeSeconds(v)
}

// Use NormalizeTimeSeconds for historical dates or far-future timestamps
func NormalizeTimeSeconds(t time.Time) uint64 {
	return NormalizeInt64(t.UTC().Unix())
}

func DenormalizeTimeSeconds(v uint64) time.Time {
	return time.Unix(DenormalizeInt64(v), 0).UTC()
}

func NormalizeUint32(v uint32) uint64 {
	// from uint32 range [0, 2^32-1] to uint64 range [0, 2^64-1]
	return uint64(v)
}

func DenormalizeUint32(v uint64) uint32 {
	// from uint64 range [0, 2^64-1] back to uint32 range [0, 2^32-1]
	return uint32(v)
}

func NormalizeValue(v any) (uint64, error) {
	switch val := v.(type) {
	case time.Time:
		return NormalizeTime(val), nil
	case *time.Time:
		if val == nil {
			return 0, errors.New("cannot normalize nil time pointer")
		}
		return NormalizeTime(*val), nil
	case int64:
		return NormalizeInt64(val), nil
	case int32:
		return NormalizeInt32(val), nil
	case float64:
		return NormalizeFloat64(val), nil
	case float32:
		return NormalizeFloat32(val), nil
	case uint32:
		return NormalizeUint32(val), nil
	default:
		return 0, fmt.Errorf("unsupported type for normalization: %T", v)
	}
}

// WARNING: This is trivially breakable with known plaintext attacks
// TODO: Replace with a more secure OPE scheme (CRITICAL!!!!!)
// Could replace this algorithm with a ported version of pyope (https://github.com/tonyo/pyope), which implements Boldyreva's symmetric OPE scheme.
//
// ct format: encrypted value (8 bytes)
func EncryptObjectLinearOPE(plaintext uint64, dek []byte) ([]byte, error) {
	if len(dek) != AESKeySize {
		return nil, fmt.Errorf("invalid DEK size: %d", len(dek))
	}

	// Encrypt plaintext with DEK using linear OPE

	// Derive slope and intercept from key
	a, b := deriveLinearParams(dek)

	// Perform linear transformation
	// This preserves order: if pt1 < pt2, then ct1 < ct2
	ciphertext := (a * plaintext) + b

	// Encode as bytes
	ct := make([]byte, 8)
	binary.BigEndian.PutUint64(ct, ciphertext)

	return ct, nil
}

// DecryptObjectLinearOPE decrypts data encrypted with EncryptObjectLinearOPE.
// Performs inverse linear transformation: pt = (ct - b) / a
func DecryptObjectLinearOPE(ct []byte, dek []byte) (uint64, error) {
	// Decrypt ciphertext with DEK
	if len(dek) != AESKeySize {
		return 0, fmt.Errorf("invalid key size: %d", len(dek))
	}
	if len(ct) != 8 {
		return 0, errors.New("invalid ciphertext size")
	}

	// Derive slope and intercept from key
	a, b := deriveLinearParams(dek)

	// Decode ciphertext
	ciphertext := binary.BigEndian.Uint64(ct)

	// Perform inverse transformation
	plaintext := (ciphertext - b) / a

	return plaintext, nil
}

// deriveLinearParams derives slope (a) and intercept (b) from key
func deriveLinearParams(key []byte) (a uint64, b uint64) {
	h := hmac.New(sha256.New, key)
	h.Write([]byte("linear-ope-slope"))
	slopeBytes := h.Sum(nil)
	a = (binary.BigEndian.Uint64(slopeBytes[:8]) >> 48) | 1 // Ensure odd (coprime with 2^64)

	h.Reset()
	h.Write([]byte("linear-ope-intercept"))
	interceptBytes := h.Sum(nil)
	b = binary.BigEndian.Uint64(interceptBytes[:8]) >> 16

	return a, b
}
