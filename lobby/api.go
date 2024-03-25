package lobby

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ParkChangSun/great-kingdom/auth"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func userIdFromContext(r *http.Request) string {
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

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	userId := userIdFromContext(r)
	c := Subscribe(userId)

	for {
		select {
		case e := <-c:
			fmt.Fprintf(w, "data: %s\n\n", e)
			f.Flush()
		case <-r.Context().Done():
			UnSubscribe(userId)
			log.Println("unsubscribe")
			return
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
	userId := userIdFromContext(r)
	CreateNewLobby(id, input.Name, input.Password, userId)

	c := http.Cookie{
		Name:     "GamePassHash",
		Value:    fmt.Sprintf("%x", md5.Sum([]byte(input.Password))),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}

	http.SetCookie(w, &c)
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

	l.mutex.Lock()
	if l.Players[0] != nil && l.Players[1] != nil {
		http.Error(w, "theres no empty seat", http.StatusInternalServerError)
		l.mutex.Unlock()
		return
	}

	if l.Password != "" {
		c, err := r.Cookie("GamePassHash")
		if err != nil {
			http.Error(w, "user payload error", http.StatusUnauthorized)
			return
		}

		if c.Value != fmt.Sprintf("%x", md5.Sum([]byte(l.Password))) {
			http.Error(w, "user payload error", http.StatusUnauthorized)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	l.Join(userIdFromContext(r), conn)
}

func CheckPassForJoin(w http.ResponseWriter, r *http.Request) {
	gameId := mux.Vars(r)["gameId"]

	p := struct {
		Password string `json:"password"`
	}{}
	json.NewDecoder(r.Body).Decode(&p)

	l, o := lobbies[gameId]
	if !o {
		http.Error(w, "cannot find game", http.StatusNotFound)
		return
	}

	c := http.Cookie{
		Name:     "GamePassHash",
		Value:    fmt.Sprintf("%x", md5.Sum([]byte(l.Password))),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}

	log.Println(p.Password)
	if p.Password == l.Password {
		http.SetCookie(w, &c)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
	}
}
