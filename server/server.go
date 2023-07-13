package server

import (
	"fmt"
	"net/http"
)

type Move struct {
	Player int `json:"player"`
	X      int `json:"x"`
	Y      int `json:"y"`
}

func server() {
	http.HandleFunc("/move", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := r.ParseForm(); err != nil {
			fmt.Printf("move error %d", err)
			return
		}

		// var move Move
		// json.NewDecoder(r.Body).Decode(&move)

		// board[move.X][move.Y] = BoardCell{true, false, move.Player, turn}

		// json.NewEncoder(w).Encode(board)
	})

	http.ListenAndServe(":8000", nil)
}
