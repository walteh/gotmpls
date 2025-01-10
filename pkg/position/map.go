package position

type PositionsSeenMap struct {
	positions map[positionKey]RawPosition
}

func NewPositionsSeenMap() *PositionsSeenMap {
	return &PositionsSeenMap{
		positions: make(map[positionKey]RawPosition),
	}
}

type positionKey struct {
	str string
	num int
}

func (me PositionsSeenMap) Add(pos RawPosition) {
	me.positions[positionKey{str: pos.Text(), num: pos.Offset()}] = pos
}

func (me PositionsSeenMap) Has(pos RawPosition) bool {
	_, ok := me.positions[positionKey{str: pos.Text(), num: pos.Offset()}]
	return ok
}

func (me PositionsSeenMap) PositionsWithText(text string) RawPositionArray {
	var positions []RawPosition
	for _, pos := range me.positions {
		if pos.Text() == text {
			positions = append(positions, pos)
		}
	}
	return positions
}
