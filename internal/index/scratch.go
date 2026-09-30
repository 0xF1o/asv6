package index

// docState is kept tiny so the hot loop touches as little memory as possible.
type docState struct {
	count   uint16 // matched trigrams so far
	mark    uint8  // last split group (1-based) the doc was hit in
	bestK1  uint8  // best split index + 1 (0 = none yet)
	bestNet int16  // matched - missed trigrams at best split
}

// scratch holds per-query accumulators; reused via sync.Pool.
type scratch struct {
	st      []docState
	touched []uint32 // docs hit at least once
	group   []uint32 // docs hit in the current split group
}

func newScratch(n int) *scratch { return &scratch{st: make([]docState, n)} }

func (s *scratch) reset() {
	for _, d := range s.touched {
		s.st[d] = docState{}
	}
	s.touched = s.touched[:0]
	s.group = s.group[:0]
}
