package position

type PositionsSeenMap struct {
	positions map[RawPosition]RawPosition
}

func NewPositionsSeenMap() *PositionsSeenMap {
	return &PositionsSeenMap{
		positions: make(map[RawPosition]RawPosition),
	}
}

func (me PositionsSeenMap) Add(pos RawPosition) {
	me.positions[pos] = pos
}

func (me PositionsSeenMap) Has(pos RawPosition) bool {
	_, ok := me.positions[pos]
	return ok
}

func (me PositionsSeenMap) PositionsWithText(text string) RawPositionArray {
	var positions []RawPosition
	for _, pos := range me.positions {
		if pos.Text == text {
			positions = append(positions, pos)
		}
	}
	return positions
}
