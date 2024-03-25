package lobby

type CellStatus int

const (
	EmptyCell CellStatus = iota
	Neutral
	BlueCastle
	OrangeCastle
	BlueTerritory
	OrangeTerritory
	SIEGED
	Edge
)

const CELLSTATUSOFFSET = 2

type Point struct {
	R int `json:"r"`
	C int `json:"c"`
}

func (p Point) getNeighbors() []Point {
	var neighborCoords = [4]Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	res := []Point{}
	for _, v := range neighborCoords {
		n := Point{p.R + v.R, p.C + v.C}
		res = append(res, n)
	}
	return res
}

type Game struct {
	board    [9][9]CellStatus
	turn     int
	passFlag bool
	playing  bool
}

func (g *Game) start() {
	g.turn = 1
	g.playing = true
	g.passFlag = false
	g.board = [9][9]CellStatus{}
	g.setCellStatus(Point{4, 4}, Neutral)
}

func (g Game) checkSieged(sidePoint Point) map[Point]struct{} {
	_, defenser := g.getCastleColor()
	cellChecked := make(map[Point]struct{})
	wallChecked := []bool{false, false, false, false}
	queue := []Point{sidePoint}

	for len(queue) > 0 {
		curPoint := queue[0]
		queue = queue[1:]

		if _, b := cellChecked[curPoint]; b {
			continue
		}

		for _, neighbor := range curPoint.getNeighbors() {
			switch c, e := g.getCellStatus(neighbor); c {
			// attacker, neutral continue
			case Edge:
				wallChecked[e] = true
				if wallChecked[0] && wallChecked[1] && wallChecked[2] && wallChecked[3] {
					return nil
				}
			case defenser:
				queue = append(queue, neighbor)
			case EmptyCell:
				return nil
			}
		}
		cellChecked[curPoint] = struct{}{}
	}

	return cellChecked
}

func (g Game) checkTerritory(sidePoint Point) map[Point]struct{} {
	_, defenser := g.getCastleColor()
	cellChecked := make(map[Point]struct{})
	wallChecked := []bool{false, false, false, false}
	queue := []Point{sidePoint}

	for len(queue) > 0 {
		curPoint := queue[0]
		queue = queue[1:]

		if _, b := cellChecked[curPoint]; b {
			continue
		}

		for _, neighbor := range curPoint.getNeighbors() {
			switch c, e := g.getCellStatus(neighbor); c {
			// attacker, neutral continue
			case EmptyCell:
				queue = append(queue, neighbor)
			case Edge:
				wallChecked[e] = true
				if wallChecked[0] && wallChecked[1] && wallChecked[2] && wallChecked[3] {
					return nil
				}
			case defenser:
				return nil
			}
		}
		cellChecked[curPoint] = struct{}{}
	}

	return cellChecked
}

func (g *Game) pass() (bool, int) {
	if g.passFlag {
		g.playing = false
		var winner int
		if b, o := g.count(); b > o {
			winner = 0
		} else if o > b {
			winner = 1
		} else {
			winner = 2
		}
		return true, winner
	}
	g.passFlag = true
	g.turn++
	return false, -1
}

func (g *Game) moveCurrentTurn(p Point) (bool, int) {
	attacker, defenser := g.getCastleColor()
	g.setCellStatus(p, attacker)

	sieged := false
	for _, sidePoint := range p.getNeighbors() {
		switch c, _ := g.getCellStatus(sidePoint); c {
		case EmptyCell:
			if res := g.checkTerritory(sidePoint); res != nil {
				for v := range res {
					g.setCellStatus(v, attacker+CELLSTATUSOFFSET)
				}
			}
		case defenser:
			if res := g.checkSieged(sidePoint); res != nil {
				sieged = true
				for v := range res {
					g.setCellStatus(v, SIEGED)
				}
			}
		}
	}

	g.passFlag = false

	if sieged {
		g.playing = false
		return true, int(attacker) - CELLSTATUSOFFSET
	} else {
		if g.isMovable() {
			g.turn++
			return false, -1
		} else {
			g.playing = false
			var winner int
			if b, o := g.count(); b > o {
				winner = 0
			} else if o > b {
				winner = 1
			} else {
				winner = 2
			}
			return true, winner
		}
	}
}

func (g Game) count() (int, int) {
	blueCount, orangeCount := 0, 0
	for _, v := range g.board {
		for _, c := range v {
			if c == BlueTerritory {
				blueCount++
			}
			if c == OrangeTerritory {
				orangeCount++
			}
		}
	}
	return blueCount, orangeCount
}

func (g Game) getCellStatus(p Point) (CellStatus, int) {
	switch {
	case p.R < 0:
		return Edge, 0
	case p.C > 8:
		return Edge, 1
	case p.R > 8:
		return Edge, 2
	case p.C < 0:
		return Edge, 3
	}
	return g.board[p.R][p.C], 0
}

func (g *Game) setCellStatus(p Point, s CellStatus) {
	g.board[p.R][p.C] = s
}

func (g Game) getCastleColor() (attacker CellStatus, defenser CellStatus) {
	if g.turn%2 == 1 {
		return BlueCastle, OrangeCastle
	} else {
		return OrangeCastle, BlueCastle
	}
}

func (g Game) isMovable() bool {
	for _, v := range g.board {
		for _, c := range v {
			if c == EmptyCell {
				return true
			}
		}
	}
	return false
}
