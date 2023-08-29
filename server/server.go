package server

import (
	"log"
	"net/http"

	"github.com/ParkChangSun/great-kingdom/auth"
	"github.com/ParkChangSun/great-kingdom/lobby"
	"github.com/gorilla/mux"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		next.ServeHTTP(w, r)
	})
}

func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func Start() {
	auth.InitAuth()

	router := mux.NewRouter()
	router.Use(loggerMiddleware)
	router.Use(corsMiddleware)

	authRouter := router.PathPrefix("/auth").Subrouter()

	authRouter.HandleFunc("/signin", auth.SignIn).Methods(http.MethodPost)
	authRouter.HandleFunc("/signup", auth.SignUp).Methods(http.MethodPost)

	gameRouter := router.PathPrefix("/game").Subrouter()
	gameRouter.Use(auth.JwtAuthMiddleware)

	gameRouter.HandleFunc("/games", lobby.LobbiesSource).Methods(http.MethodGet)

	gameRouter.HandleFunc("/create", lobby.Create).Methods(http.MethodPost)

	// ws
	gameRouter.HandleFunc("/{gameId}", lobby.Join)
	// gameRouter.HandleFunc("/{gameId}/chat", lobby.JoinLobbyChat)
	// gameRouter.HandleFunc("/{gameId}/ready", lobby.PlayerReady)

	// gameRouter.HandleFunc("/{gameId}/join", lobby.PlayerJoinLobby).Methods(http.MethodPost)

	http.ListenAndServe(":8000", router)
}
