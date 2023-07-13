package main

import "fmt"

type Board [9][9]int

var board Board

var neighbors = [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

var turn = 1

var debug = false

var winner = 0

func main() {
	move(4, 4, 3)
	for winner == 0 {
		board.debug()
		fmt.Printf("Turn: %d\nPlayer: %d\n", turn, getPlayingPiece())
		player := getPlayingPiece()
		oppo := getOpponentPiece(player)

		var x, y int
		fmt.Scanln(&x, &y)
		move(x, y, player)

		for _, v := range neighbors {
			dx, dy := x+v[0], y+v[1]
			p, e := board.getCoordinate(dx, dy)

			if p == 0 && e == 0 {
				checkedList = map[[2]int]int{}
				o := board.checkOccupation(dx, dy, player)
				fmt.Println("occupation", dx, dy, o)
			}

			if p == oppo {
				checkedList = map[[2]int]int{}
				o := board.checkSieged(dx, dy, oppo)
				fmt.Println("sieged", dx, dy, o)
				if o {
					fmt.Println("game over")
					winner = player
					board.debug()
					break
				}
			}
		}
		fmt.Println("---------------------")
	}
	fmt.Println(winner)
}

func move(x, y, player int) {
	board[y][x] = player
	turn++
}

func getPlayingPiece() int {
	if turn%2 == 1 {
		return 1
	} else {
		return 2
	}
}

func getOpponentPiece(p int) int {
	if p == 1 {
		return 2
	} else if p == 2 {
		return 1
	} else {
		// error piece is neutral
		return 0
	}
}

var checkedList map[[2]int]int

func (b Board) checkSieged(x, y, siegedFlag int) bool {
	if _, b := checkedList[[2]int{x, y}]; b {
		return true
	}

	templist := [][2]int{}

	for _, v := range neighbors {
		dx, dy := x+v[0], y+v[1]
		c, _ := b.getCoordinate(dx, dy)
		if c == 0 {
			return false
		} else if c == siegedFlag {
			templist = append(templist, [2]int{dx, dy})
		}
	}

	checkedList[[2]int{x, y}] = siegedFlag

	for _, v := range templist {
		if !b.checkSieged(v[0], v[1], siegedFlag) {
			return false
		}
	}
	return true
}

var edgeCount = make(map[int]bool)

func (b Board) checkOccupation(x, y, playerFlag int) bool {
	if _, b := checkedList[[2]int{x, y}]; b {
		return true
	}

	opponentFlag := getOpponentPiece(playerFlag)
	templist := [][2]int{}

	// 현재 칸은 빈칸이라고 가정
	// 현재 칸 위의 칸이
	// 상대 말이다 -> false
	// 내 말/중립말 이다 -> 계산할 수 없음. true
	// 빈칸이다 -> 그 칸도 계산해야함
	// 가장자리다 -> 가장자리는 계산할 수 없음.
	// 가장자리가 4개이면 false

	for _, v := range neighbors {
		dx, dy := x+v[0], y+v[1]
		c, e := b.getCoordinate(dx, dy)
		if c == opponentFlag {
			return false
		} else if c == 0 {
			templist = append(templist, [2]int{dx, dy})
		} else if e != 0 {
			edgeCount[e] = true

			if edgeCount[1] && edgeCount[2] && edgeCount[3] && edgeCount[4] {
				return false
			}
		}
	}

	checkedList[[2]int{x, y}] = playerFlag
	for _, v := range templist {
		if !b.checkOccupation(v[0], v[1], playerFlag) {
			return false
		}
	}

	return true
}

// edge 리턴할때 어느쪽 가장자리인지 알려줘야한다
// e s w n
// 1 2 3 4

func (b Board) getCoordinate(x, y int) (playerFlag int, isBoardEdge int) {
	if x > 8 {
		return 3, 1
	} else if x < 0 {
		return 3, 3
	} else if y > 8 {
		return 3, 2
	} else if y < 0 {
		return 3, 4
	} else {
		return b[y][x], 0
	}
}

func (b Board) debug() {
	if debug {
		fmt.Print("####### debug #######\n\n")
	}
	fmt.Println("yx 0 1 2 3 4 5 6 7 8")
	for i, v := range b {
		fmt.Println(i, v)
	}
	if debug {
		fmt.Print("\n####### debug #######\n")
	}
}
