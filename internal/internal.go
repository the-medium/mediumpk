/*
Copyright Medium Corp. 2020 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package internal

import (
	"crypto/elliptic"
	"io"
	"math/big"
	"sync"
)

const (
	AesIV = "IV for ECDSA CTR"
)

var (
	closedChanOnce sync.Once
	closedChan     chan struct{}
	one            = new(big.Int).SetInt64(1)
)

type zr struct {
	io.Reader
}

// randFieldElement returns a random element of the field underlying the given
// curve using the procedure given in [NSA] A.2.1.
func RandFieldElement(c elliptic.Curve, rand io.Reader) (k []byte, err error) {
	params := c.Params()
	b := make([]byte, params.BitSize/8+8)
	_, err = io.ReadFull(rand, b)
	if err != nil {
		return
	}

	_k := new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	_k.Mod(_k, n)
	_k.Add(_k, one)

	k = _k.Bytes()
	return
}

// Read replaces the contents of dst with zeros.
func (z *zr) Read(dst []byte) (n int, err error) {
	for i := range dst {
		dst[i] = 0
	}
	return len(dst), nil
}

var ZeroReader = &zr{}

// maybeReadByte reads a single byte from r with ~50% probability. This is used
// to ensure that callers do not depend on non-guaranteed behaviour, e.g.
// assuming that rsa.GenerateKey is deterministic w.r.t. a given random stream.
//
// This does not affect tests that pass a stream of fixed bytes as the random
// source (e.g. a zeroReader).
func MaybeReadByte(r io.Reader) {
	closedChanOnce.Do(func() {
		closedChan = make(chan struct{})
		close(closedChan)
	})

	select {
	case <-closedChan:
		return
	case <-closedChan:
		var buf [1]byte
		r.Read(buf[:])
	}
}
