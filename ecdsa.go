package mediumpk

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"io"
	"math/big"

	"github.com/the-medium/mediumpk/internal"
)

var (
	errZeroParam = errors.New("zero parameter")
)

// A invertible implements fast inverse mod Curve.Params().N
type invertible interface {
	// Inverse returns the inverse of k in GF(P)
	Inverse(k *big.Int) *big.Int
}

func fermatInverse(k, N *big.Int) *big.Int {
	two := big.NewInt(2)
	nMinus2 := new(big.Int).Sub(N, two)
	return new(big.Int).Exp(k, nMinus2, N)
}

func hashToInt(hash []byte, c elliptic.Curve) *big.Int {
	orderBits := c.Params().N.BitLen()
	orderBytes := (orderBits + 7) / 8
	if len(hash) > orderBytes {
		hash = hash[:orderBytes]
	}

	ret := new(big.Int).SetBytes(hash)
	excess := len(hash)*8 - orderBits
	if excess > 0 {
		ret.Rsh(ret, uint(excess))
	}
	return ret
}

// CreateRandomK creates random k
func CreateRandomK(d []byte, hash []byte) (k []byte, err error) {
	rand := rand.Reader
	internal.MaybeReadByte(rand)

	// Get min(log2(q) / 2, 256) bits of entropy from rand.
	entropylen := (256 + 7) / 16
	if entropylen > 32 {
		entropylen = 32
	}
	entropy := make([]byte, entropylen)
	_, err = io.ReadFull(rand, entropy)
	if err != nil {
		return
	}

	// Initialize an SHA-512 hash context; digest ...
	md := sha512.New()
	md.Write(d)             // the private key,
	md.Write(entropy)       // the entropy,
	md.Write(hash)          // and the input hash;
	key := md.Sum(nil)[:32] // and compute ChopMD-256(SHA-512),
	// which is an indifferentiable MAC.

	// Create an AES-CTR instance to use as a CSPRNG.
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, err
	}

	// Create a CSPRNG that xors a stream of zeros with
	// the output of the AES-CTR instance.
	csprng := cipher.StreamReader{
		R: internal.ZeroReader,
		S: cipher.NewCTR(block, []byte(internal.AesIV)),
	}

	// See [NSA] 3.4.1
	c := elliptic.P256()
	N := c.Params().N
	if N.Sign() == 0 {
		return nil, errors.New("zero parameter")
	}

	return internal.RandFieldElement(c, csprng)
}

func SignCPU(priv *ecdsa.PrivateKey, k *big.Int, c elliptic.Curve, hash []byte) (r, s *big.Int, err error) {
	N := c.Params().N
	if N.Sign() == 0 {
		return nil, nil, errZeroParam
	}
	var kInv *big.Int
	for {
		for {
			if in, ok := priv.Curve.(invertible); ok {
				kInv = in.Inverse(k)
			} else {
				kInv = fermatInverse(k, N) // N != 0
			}

			r, _ = priv.Curve.ScalarBaseMult(k.Bytes())
			r.Mod(r, N)
			if r.Sign() != 0 {
				break
			}
		}

		e := hashToInt(hash, c)
		s = new(big.Int).Mul(priv.D, r)
		s.Add(s, e)
		s.Mul(s, kInv)
		s.Mod(s, N) // N != 0
		if s.Sign() != 0 {
			break
		}
	}

	return
}

func VerifyCPU(pub *ecdsa.PublicKey, hash []byte, r, s *big.Int) bool {
	return ecdsa.Verify(pub, hash, r, s)
}
