package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id string `json:"name"`
}

type CustomClaimsExample struct {
	*jwt.StandardClaims
	TokenType string
	SignData
}

type UserDatabase struct {
	Pk       int
	Id       string
	PassHash string
	Uuid     string
}

var privateKey *rsa.PrivateKey
var db *sql.DB

func InitAuth() {
	privateKey, _ = rsa.GenerateKey(rand.Reader, 2048)

	if _db, err := sql.Open("sqlite3", "data.sqlite3"); err != nil {
		log.Fatalln(err)
	} else {
		db = _db
	}

	_, err := db.Exec(`create table if not exists user (
		pk integer primary key autoincrement,
		id text,
		passhash text,
		uuid text
	)`)
	if err != nil {
		log.Fatal(err)
	}
}

type SignData struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}

func SignIn(w http.ResponseWriter, r *http.Request) {
	signInData := SignData{}
	json.NewDecoder(r.Body).Decode(&signInData)

	res := db.QueryRow(`select passhash from user where id=?`, signInData.Id)
	var passhash string
	if err := res.Scan(&passhash); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {
			log.Fatal(err)
		}
	}
	if bcrypt.CompareHashAndPassword([]byte(passhash), []byte(signInData.Password)) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	claim := CustomClaimsExample{
		&jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
		"level1",
		signInData,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claim).SignedString(privateKey)
	if err != nil {
		log.Println(err)
	}

	j := http.Cookie{
		Name:     "Bearer",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		// set in http server
		// SameSite: http.SameSiteNoneMode,
	}
	w.Header().Set("Set-Cookie", j.String())
	json.NewEncoder(w).Encode(User{signInData.Id})
}

func SignUp(w http.ResponseWriter, r *http.Request) {
	signUpData := SignData{}
	json.NewDecoder(r.Body).Decode(&signUpData)

	passhash, err := bcrypt.GenerateFromPassword([]byte(signUpData.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`insert into user (id,passhash,uuid) values (?,?,?)`, signUpData.Id, passhash, uuid.New().String())
	if err != nil {
		log.Fatal(err)
	}

	w.WriteHeader(http.StatusCreated)
}

type AuthKey string

const UserIdKey = AuthKey("UserId")

func JwtAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer, err := r.Cookie("Bearer")
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		claim := CustomClaimsExample{}
		token, err := jwt.ParseWithClaims(bearer.Value, &claim, func(t *jwt.Token) (interface{}, error) {
			return privateKey.Public(), nil
		})
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIdKey, claim.SignData.Id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
