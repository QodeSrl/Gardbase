// Order-Preserving and Range-Query Encryption.
//
// CRITICAL SECURITY WARNING
// Order-preserving encryption (OPE) leaks order, approximate values, distribution, and frequency information.
// Only use for LOW-SENSITIVITY data where range queries are absolutely necessary.

package crypto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/alessandrofoglia07/goope"
)

var opeIn32, opeOut64 goope.ValueRange

func init() {
	var err error
	opeIn32, err = goope.NewValueRange(0, math.MaxUint32) // for 32-bit normalized values
	if err != nil {
		panic(fmt.Sprintf("failed to initialize OPE input range: %v", err))
	}
	opeOut64, err = goope.NewValueRange(0, math.MaxInt64>>2) // wide ciphertext range to reduce risk of collisions
	if err != nil {
		panic(fmt.Sprintf("failed to initialize OPE output range: %v", err))
	}
}

func NormalizeInt64OPE(v int64) int64 {
	// flip sign bit so negatives sort below positives, then take high 32 bits
	ordered := uint64(v) ^ (1 << 63)
	return int64(ordered >> 32) // range [0, 2^32-1] for all int64 values
}

func DenormalizeInt64OPE(v int64) int64 {
	// from int64 range [0, 2^32-1] back to int64 range [-2^63, 2^63-1]
	ordered := uint64(v) << 32
	return int64(ordered ^ (1 << 63))
}

func NormalizeInt32OPE(v int32) int64 {
	// XOR with sign bit maps [-2^31, 2^31-1] to [0, 2^32-1]
	return int64(uint32(v) ^ (1 << 31))
}

func DenormalizeInt32OPE(v int64) int32 {
	// from int64 range [0, 2^32-1] back to int32 range [-2^31, 2^31-1]
	return int32(uint32(v) ^ (1 << 31))
}

func NormalizeUint32OPE(v uint32) int64 {
	return int64(v)
}

func DenormalizeUint32OPE(v int64) uint32 {
	return uint32(v)
}

func NormalizeFloat64OPE(v float64) (int64, error) {
	if math.IsNaN(v) {
		return 0, errors.New("NaN cannot be encrypted with OPE")
	}
	bits := math.Float64bits(v)
	var ordered uint64
	if bits>>63 == 0 {
		ordered = bits ^ (1 << 63) // positive: flip sign bit
	} else {
		ordered = ^bits // negative: flip all bits
	}
	return int64(ordered >> 32), nil
}

func DenormalizeFloat64OPE(v int64) float64 {
	ordered := uint64(v) << 32
	var bits uint64
	if ordered>>63 == 1 {
		bits = ordered ^ (1 << 63) // was positive: flip sign bit back
	} else {
		bits = ^ordered // was negative: flip all bits back
	}
	return math.Float64frombits(bits)
}

func NormalizeFloat32OPE(v float32) (int64, error) {
	if math.IsNaN(float64(v)) {
		return 0, errors.New("NaN cannot be encrypted with OPE")
	}
	bits := math.Float32bits(v)
	var ordered uint32
	if bits>>31 == 0 {
		ordered = bits ^ (1 << 31) // positive: flip sign bit
	} else {
		ordered = ^bits // negative: flip all bits
	}
	return int64(ordered), nil
}

func DenormalizeFloat32OPE(v int64) float32 {
	u := uint32(v)
	var bits uint32
	if u>>31 == 1 {
		bits = u ^ (1 << 31) // was positive: flip sign bit back
	} else {
		bits = ^u // was negative: flip all bits back
	}
	return math.Float32frombits(bits)
}

func NormalizeTimeOPE(t time.Time) (int64, error) {
	unix := t.UTC().Unix()
	if unix < 0 || unix > math.MaxUint32 {
		return 0, fmt.Errorf("timestamp %v out of OPE range [0, 2^32-1]: use NormalizeTimeExtendedOPE", t)
	}
	return unix, nil
}

func DenormalizeTimeOPE(v int64) time.Time {
	return time.Unix(v, 0).UTC()
}

// NormalizeTimeExtendedOPE handles timestamps before 1970 or after 2106 by
// mapping Unix seconds (int64) using the same high-32-bit truncation as
// NormalizeInt64OPE. Precision is ~136 years per unit.
func NormalizeTimeExtendedOPE(t time.Time) int64 {
	return NormalizeInt64OPE(t.UTC().Unix())
}

func DenormalizeTimeExtendedOPE(v int64) time.Time {
	return time.Unix(DenormalizeInt64OPE(v), 0).UTC()
}

func NormalizeValueOPE(v any) (int64, error) {
	switch val := v.(type) {
	case int64:
		return NormalizeInt64OPE(val), nil
	case int32:
		return NormalizeInt32OPE(val), nil
	case float32:
		return NormalizeFloat32OPE(val)
	case float64:
		return NormalizeFloat64OPE(val)
	case uint32:
		return NormalizeUint32OPE(val), nil
	case uint64:
		return 0, errors.New("uint64 cannot be losslessly normalized for OPE: range exceeds int64; use uint32 or truncate manually")
	case int:
		return NormalizeInt64OPE(int64(val)), nil
	case uint:
		if val > math.MaxUint32 {
			return 0, fmt.Errorf("uint value %d exceeds OPE input range; cast to uint32 explicitly", val)
		}
		return NormalizeUint32OPE(uint32(val)), nil
	case time.Duration:
		return NormalizeInt64OPE(int64(val)), nil
	case int16:
		return NormalizeInt32OPE(int32(val)), nil
	case int8:
		return NormalizeInt32OPE(int32(val)), nil
	case uint16:
		return NormalizeUint32OPE(uint32(val)), nil
	case uint8:
		return NormalizeUint32OPE(uint32(val)), nil
	case time.Time:
		return NormalizeTimeOPE(val)
	case *time.Time:
		if val == nil {
			return 0, errors.New("cannot normalize nil time pointer")
		}
		return NormalizeTimeOPE(*val)
	default:
		return 0, fmt.Errorf("unsupported type for normalization: %T", v)
	}
}

func EncryptObjectOPE(plaintext int64, dek []byte) ([]byte, error) {
	if len(dek) != AESKeySize {
		return nil, fmt.Errorf("invalid DEK size: %d", len(dek))
	}

	ope, err := goope.NewOPE(dek, &opeIn32, &opeOut64)
	if err != nil {
		return nil, fmt.Errorf("failed to create OPE instance: %w", err)
	}

	ciphertext, err := ope.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt with OPE: %w", err)
	}

	// Encode as bytes
	ct := make([]byte, 8)
	binary.BigEndian.PutUint64(ct, uint64(ciphertext))

	return ct, nil
}

func DecryptObjectOPE(ct []byte, dek []byte) (int64, error) {
	if len(dek) != AESKeySize {
		return 0, fmt.Errorf("invalid key size: %d", len(dek))
	}
	if len(ct) != 8 {
		return 0, errors.New("invalid ciphertext size")
	}

	ope, err := goope.NewOPE(dek, &opeIn32, &opeOut64)
	if err != nil {
		return 0, fmt.Errorf("failed to create OPE instance: %w", err)
	}

	// Decode ciphertext
	raw := int64(binary.BigEndian.Uint64(ct))

	plaintext, err := ope.Decrypt(raw)
	if err != nil {
		return 0, fmt.Errorf("failed to decrypt with OPE: %w", err)
	}

	return plaintext, nil
}
