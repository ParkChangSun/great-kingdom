package game

/*
 */
type BoardCell struct {
	IsPlayed   bool
	IsOccupied bool
	Player     int
	Turn       int
}
