package mediumpk

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testSignAndVerify(t *testing.T, c elliptic.Curve, tag string) {
	priv, _ := ecdsa.GenerateKey(c, rand.Reader)

	hashed := []byte("testing")
	randomK, err := CreateRandomK(priv.D.Bytes(), hashed)
	assert.NoError(t, err)
	k := new(big.Int)
	k.SetBytes(randomK)
	r, s, err := SignCPU(priv, k, c, hashed)
	if err != nil {
		t.Errorf("%s: error signing: %s", tag, err)
		return
	}

	if !VerifyCPU(&priv.PublicKey, hashed, r, s) {
		t.Errorf("%s: Verify failed", tag)
	}

	hashed[0] ^= 0xff
	if VerifyCPU(&priv.PublicKey, hashed, r, s) {
		t.Errorf("%s: Verify always works!", tag)
	}
}

func TestSignAndVerify(t *testing.T) {
	testSignAndVerify(t, elliptic.P224(), "p224")
	if testing.Short() {
		return
	}
	testSignAndVerify(t, elliptic.P256(), "p256")
	testSignAndVerify(t, elliptic.P384(), "p384")
	testSignAndVerify(t, elliptic.P521(), "p521")
}
