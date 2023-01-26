package ksuid

import "github.com/segmentio/ksuid"

// Sequence is a wrapper around ksuid.Sequence used for lock-free ksuid generation.
type Sequence struct {
	seq ksuid.Sequence
}

// NewSequence initializes new Sequence.
func NewSequence() *Sequence {
	return &Sequence{
		seq: ksuid.Sequence{Seed: ksuid.New()},
	}
}

// Get new KSUID from the sequence.
func (s *Sequence) Get() ksuid.KSUID {
	id, err := s.seq.Next()
	if err != nil {
		s.seq = ksuid.Sequence{Seed: ksuid.New()}
		// Only case of error returned by ksuid.Sequence.Next() is when
		// the sequence was spent entirely. Here we generated new
		// sequence, so the error will not occur.
		id, _ = s.seq.Next()
	}

	return id
}
