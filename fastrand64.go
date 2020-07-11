// Package fastrand implements fast pesudorandom number generator
// that should scale well on multi-CPU systems.
//
// Use crypto/rand instead of this package for generating
// cryptographically secure random numbers.
package fastrand64

import (
	"math/rand"
	"sync"
	"time"
)

type ThreadsafePoolRNG struct {
	rngPool sync.Pool
}

type UnsafeRNG interface {
	Uint64() uint64
}

func NewSyncPoolRNG(fn func() UnsafeRNG) *ThreadsafePoolRNG {
	s := &ThreadsafePoolRNG{}
	s.rngPool = sync.Pool{New: func() interface{} { return fn() }}
	return s
}

func NewSyncPoolXoshiro256ssRNG() *ThreadsafePoolRNG {
	rand.Seed(time.Now().UnixNano())
	return NewSyncPoolRNG(func() UnsafeRNG {
		return NewUnsafeXoshiro256ssRNG(int64(rand.Uint64()))
	})
}

// Uint64 returns pseudorandom uint64. Threadsafe
func (s *ThreadsafePoolRNG) Uint64() uint64 {
	r := s.rngPool.Get().(UnsafeRNG)
	x := r.Uint64()
	s.rngPool.Put(r)
	return x
}

// should only be used to match Source64 interface
func (s *ThreadsafePoolRNG) Int63() int64 {
	return int64(0x7FFFFFFFFFFFFFFF & s.Uint64())
}

// should only be used to match Source64 interface
func (s *ThreadsafePoolRNG) Seed(seed int64) {
	// you cant really seed a PoolRNG, since the call order is non-determinate
	panic("Cant seed a ThreadsafePoolRNG")
}

func (s *ThreadsafePoolRNG) Bytes(n int) []byte {
	r := s.rngPool.Get().(UnsafeRNG)
	bytes := make([]byte, n)
	result := Bytes(r, bytes)
	s.rngPool.Put(r)
	return result
}

func (s *ThreadsafePoolRNG) Read(p []byte) []byte {
	r := s.rngPool.Get().(UnsafeRNG)
	Bytes(r, p)
	s.rngPool.Put(r)
	return p
}

func Bytes(r UnsafeRNG, bytes []byte) []byte {
	n := len(bytes)
	bytesToGo := n
	i := 0

	for {
		if bytesToGo < 8 {
			break
		}
		x := r.Uint64()
		bytes[i] = byte(x)
		bytes[i+1] = byte(x >> 8)
		bytes[i+2] = byte(x >> 16)
		bytes[i+3] = byte(x >> 24)
		bytes[i+4] = byte(x >> 32)
		bytes[i+5] = byte(x >> 40)
		bytes[i+6] = byte(x >> 48)
		bytes[i+7] = byte(x >> 56)
		i += 8
		bytesToGo -= 8
	}

	x := r.Uint64()
	for {
		if i >= n {
			break
		}
		bytes[i] = byte(x)
		x >>= 8
		i += 1
	}

	return bytes
}

// Uint32n returns pseudorandom Uint32n in the range [0..maxN).
//
// It is safe calling this function from concurrent goroutines.
func (r *ThreadsafePoolRNG) Uint32n(maxN int) uint32 {
	x := r.Uint64() & 0x00000000FFFFFFFF
	// See http://lemire.me/blog/2016/06/27/a-fast-alternative-to-the-modulo-reduction/
	return uint32((x * uint64(maxN)) >> 32)
}

// UnsafeXoshiro256** is a pseudorandom number generator.
// For an interesting commentary on xoshiro256**
// https://www.pcg-random.org/posts/a-quick-look-at-xoshiro256.html

// It is unsafe to call UnsafeRNG methods from concurrent goroutines.
type UnsafeXoshiro256ssRNG struct {
	s [4]uint64
}

func rol64(x uint64, k uint64) uint64 {
	return (x << k) | (x >> (64 - k))
}

func splitmix64(index uint64) uint64 {
	z := (index + uint64(0x9E3779B97F4A7C15))
	z = (z ^ (z >> 30)) * uint64(0xBF58476D1CE4E5B9)
	z = (z ^ (z >> 27)) * uint64(0x94D049BB133111EB)
	z = z ^ (z >> 31)
	return z
}

func (r *UnsafeXoshiro256ssRNG) Uint64() uint64 {
	// See https://en.wikipedia.org/wiki/Xorshift
	s := &r.s
	result := rol64(s[1]*5, 7) * 9
	t := s[1] << 17

	s[2] ^= s[0]
	s[3] ^= s[1]
	s[1] ^= s[2]
	s[0] ^= s[3]

	s[2] ^= t
	s[3] = rol64(s[3], 45)

	return result
}

func (r *UnsafeXoshiro256ssRNG) Seed(seed int64) {
	for i := 0; i < len(r.s); i++ {
		for r.s[i] == 0 {
			r.s[i] = splitmix64(uint64(seed) + uint64(i))
		}
	}
}

// Thread unsafe PRNG
func NewUnsafeXoshiro256ssRNG(seed int64) *UnsafeXoshiro256ssRNG {
	r := &UnsafeXoshiro256ssRNG{}
	r.Seed(seed)
	return r
}

func NewUnsafeRandRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed).(rand.Source64))
}
