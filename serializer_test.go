package mediumpk

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/the-medium/mediumpk/internal"
)

var (
	strD  = "519b423d715f8b581f4fa8ee59f4771a5b44c8130b4e3eacca54a56dda72b464"
	strK  = "94a1bbb14b906a61a280f245f9e93c7f3b4a6247824f5d33b9670787642a68de"
	strH1 = "ea5cd45052849c4ae816bbc44ed833e832af8a619ba47268aabca2744c4c6268"
	strX  = "e305d41ab27b39c84230ab2faf34fb15e9d0543f4ac19d2520b94d71df9be5bf"
	strY  = "0b97c506c163237d6e9264f7148336e524d32174754198066995a252b1a51f4e"
	strR  = "5806c2774086b61c97afd87585215c09fe57233f232278c0e8976d35f0570641"
	strS  = "6d8a758eb8edfeecbdab2e413bee8bc73a88a887f97a54c2a967de0afcb8b0af"
	strH2 = "00000000000000000000000000000000000000000048656c6c6f20576f726c64"
)

func TestSerializeSignRequest(t *testing.T) {
	// d
	d, err := hex.DecodeString(strD)
	assert.NoError(t, err)

	// k
	k, err := hex.DecodeString(strK)
	assert.NoError(t, err)

	// hash
	h, err := hex.DecodeString(strH1)
	assert.NoError(t, err)

	// create envelop
	d32 := make([]byte, 32)
	k32 := make([]byte, 32)
	h32 := make([]byte, 32)
	copy(d32[32-len(d):], d[:])
	copy(k32[32-len(k):], k[:])
	copy(h32[32-len(h):], h[:])
	env := SignRequestEnvelop{
		d32,
		k32,
		h32,
	}

	// expected
	expected := make([]byte, internal.SignRequestSize)
	header := []byte{170, 170, 170, 170, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16}
	i := 0
	i += copy(expected[i:], header)
	i += copy(expected[i:], d32)
	i += copy(expected[i:], k32)
	i += copy(expected[i:], h32)

	// serialize envelop
	serializer := serializer{}
	serialized := serializer.serializeSignRequest(env, 16)
	assert.Equal(t, expected, serialized)
}

func TestSerializeVerifyRequest(t *testing.T) {
	// qx, qy
	x, err := hex.DecodeString(strX)
	assert.NoError(t, err)
	y, err := hex.DecodeString(strY)
	assert.NoError(t, err)

	// r, s
	r, err := hex.DecodeString(strR)
	assert.NoError(t, err)
	s, err := hex.DecodeString(strS)
	assert.NoError(t, err)

	// hash
	h, err := hex.DecodeString(strH2)
	assert.NoError(t, err)

	// create envelop
	qx32 := make([]byte, 32)
	qy32 := make([]byte, 32)
	r32 := make([]byte, 32)
	s32 := make([]byte, 32)
	h32 := make([]byte, 32)
	copy(qx32[32-len(x):], x[:])
	copy(qy32[32-len(y):], y[:])
	copy(r32[32-len(r):], r[:])
	copy(s32[32-len(s):], s[:])
	copy(h32[32-len(h):], h[:])
	env := VerifyRequestEnvelop{
		qx32,
		qy32,
		r32,
		s32,
		h32,
	}

	// expected
	expected := make([]byte, internal.VerifyRequestSize)
	header := []byte{187, 187, 187, 187, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16}
	i := 0
	i += copy(expected[i:], header)
	i += copy(expected[i:], qx32)
	i += copy(expected[i:], qy32)
	i += copy(expected[i:], r32)
	i += copy(expected[i:], s32)
	i += copy(expected[i:], h32)

	// serialize envelop
	serializer := serializer{}
	serialized := serializer.serializeVerifyRequest(env, 16)
	assert.Equal(t, expected, serialized)
}

func TestDeserializeResponse(t *testing.T) {
	bufStr := "0000aaaa000000000000000000000abc6c0f55fd455d34ac67ca2d987c5b50e795ec0e5eeacfb0bbf3cfdb2a428e17ac84a6603b1e0b5b577b97ba529bd1e1aa758e299e616bbe6fb2e2fd6b5ed4737400000000000000000000000000000000"
	buffer, err := hex.DecodeString(bufStr)
	assert.NoError(t, err)

	env := ResponseEnvelop{}

	// deserialize buffer
	deserializer := deserializer{}
	userctx, err := deserializer.deserializeResponse(&env, buffer)
	assert.NoError(t, err)
	assert.Equal(t, 0xabc, userctx)
	assert.Equal(t, buffer[16:48], env.r)
	assert.Equal(t, buffer[48:80], env.s)
}

func TestDeserializeMetric(t *testing.T) {
	buffer := []byte{236, 160, 0, 0, 218, 69, 0, 0, 122, 154, 0, 0, 226, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	env := MetricEnvelop{}

	// deserialize buffer
	d := deserializer{}
	err := d.deserializeMetric(&env, buffer)
	assert.NoError(t, err)
	assert.Equal(t, "41.486725", env.temperature)
	assert.Equal(t, "0.818573", env.vccint)
	assert.Equal(t, "1.810272", env.vccaux)
	assert.Equal(t, "0.818939", env.vccbram)
	assert.Equal(t, 0, env.signCount)
	assert.Equal(t, 0, env.verifyCount)
	assert.Equal(t, 0, env.errorCount)
}
