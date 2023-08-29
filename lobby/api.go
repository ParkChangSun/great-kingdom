package lobby

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ParkChangSun/great-kingdom/auth"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func userFromContext(r *http.Request) string {
	ctxVal := r.Context().Value(auth.UserIdKey)
	if ctxVal == nil {
		log.Fatal("context user value nil")
	}
	return ctxVal.(string)
}

func LobbiesSource(w http.ResponseWriter, r *http.Request) {
	f, o := w.(http.Flusher)
	if !o {
		http.Error(w, "cannot use flusher", http.StatusNotAcceptable)
		return
	}

	userId := userFromContext(r)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	c := Subscribe(userId)

	for {
		select {
		case <-r.Context().Done():
			UnSubscribe(userId)
			return
		case e := <-c:
			fmt.Fprintf(w, "data: %s\n\n", e)
			f.Flush()
		}
	}
}

func Create(w http.ResponseWriter, r *http.Request) {
	input := struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}{}
	json.NewDecoder(r.Body).Decode(&input)

	id := uuid.New().String()
	userId := userFromContext(r)
	CreateNewLobby(id, input.Name, input.Password, userId)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		Id string `json:"id"`
	}{id})
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// fix when deploy
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Join(w http.ResponseWriter, r *http.Request) {
	gameId := mux.Vars(r)["gameId"]
	l, o := lobbies[gameId]
	if !o {
		http.Error(w, "cannot find game", http.StatusNotFound)
		return
	}

	userId := userFromContext(r)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	l.Join(userId, conn)
}
