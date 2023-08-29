package lobby

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"sync"

	"github.com/ParkChangSun/great-kingdom/game"
	"github.com/gorilla/websocket"
)

type Player struct {
	UserId string
	Conn   *websocket.Conn
}

// 걍 다 소켓으로 박아

type Lobby struct {
	Id   string
	Name string

	Players []*Player
	Host    string

	EventChannel chan ServerProtocol

	mutex *sync.Mutex

	Game *game.Game
}

// todo change into array
var lobbies map[string]*Lobby = make(map[string]*Lobby)

var eventSubscribers = make(map[string]chan string)

func EventData() string {
	l := []LobbyCard{}
	for _, v := range lobbies {
		l = append(l, v.Card())
	}
	data, err := json.Marshal(l)
	if err != nil {
		log.Println("emitevent", err)
	}
	return string(data)
}

func EmitEvent() {
	data := EventData()
	for _, v := range eventSubscribers {
		v <- data
	}
}

func Subscribe(id string) chan string {
	c := make(chan string, 1)
	eventSubscribers[id] = c

	c <- EventData()
	return c
}

func UnSubscribe(id string) {
	log.Println(id, "unsubscribe")
	close(eventSubscribers[id])
	delete(eventSubscribers, id)
}

func CreateNewLobby(id, name, password, userId string) {
	lobbies[id] = &Lobby{
		Id:           id,
		Name:         name,
		Players:      make([]*Player, 2),
		EventChannel: make(chan ServerProtocol, 2),
		mutex:        &sync.Mutex{},
	}
	go lobbies[id].StartLobby()

	EmitEvent()
}

func DeleteLobby(id string) {
	l, o := lobbies[id]
	if !o {
		log.Println("deletelobby: lobby not found")
		return
	}

	close(l.EventChannel)
	delete(lobbies, l.Id)

	EmitEvent()
}

func (l *Lobby) StartLobby() {
	for v := range l.EventChannel {
		for _, p := range l.Players {
			if p != nil {
				p.Conn.WriteJSON(v)
			}
		}
	}
}

func (l *Lobby) Join(userId string, conn *websocket.Conn) {
	l.mutex.Lock()

	i := slices.Index[[]*Player, *Player](l.Players, nil)
	if i == -1 {
		l.mutex.Unlock()
		conn.Close()
		return
	}

	l.Players[i] = &Player{
		UserId: userId,
		Conn:   conn,
	}

	l.EventChannel <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s has joined the lobby.", userId)}

	if l.Host == "" {
		l.Host = userId
		l.EventChannel <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s is the host.", l.Host)}
	}

	l.mutex.Unlock()

	l.PlayerChangeEvent()

	go func() {
		for {
			p := ClientProtocol{}
			err := l.Players[i].Conn.ReadJSON(&p)
			if err != nil {
				l.Left(userId)
				return
			}

			switch p.Event {
			case CHAT:
				l.EventChannel <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s : %s", userId, p.ChatMessage)}
			case STARTGAME:
				if userId == l.Host {
					l.StartGame()
				}
			}
		}
	}()
}

func (l *Lobby) PlayerChangeEvent() {
	tp := []string{}
	for _, v := range l.Players {
		if v == nil {
			tp = append(tp, "")
			continue
		}
		tp = append(tp, v.UserId)
	}
	l.EventChannel <- ServerProtocol{Event: PLAYERCHANGE, Players: tp, HostId: l.Host}
}

func (l *Lobby) Left(userId string) {
	l.mutex.Lock()

	i := slices.IndexFunc[[]*Player, *Player](l.Players, func(p *Player) bool {
		if p == nil {
			return false
		}
		return p.UserId == userId
	})
	if i == -1 {
		log.Println("left error userid not found", l.Players)
		l.mutex.Unlock()
		return
	}

	l.Players[i].Conn.Close()
	l.Players[i] = nil

	l.EventChannel <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s has gone away.", userId)}

	if l.Host == userId {
		o := slices.IndexFunc[[]*Player, *Player](l.Players, func(p *Player) bool { return p != nil })
		if o == -1 {
			l.mutex.Unlock()
			DeleteLobby(l.Id)
			log.Println("no player in", l.Id, "deleted")
			return
		} else {
			l.Host = l.Players[o].UserId
			l.EventChannel <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s is the host.", l.Host)}
		}
	}

	l.mutex.Unlock()

	l.PlayerChangeEvent()
}

type EventType int

const (
	CHAT EventType = iota
	PLAYERCHANGE
	STARTGAME
	GAMEMOVE
)

type ServerProtocol struct {
	Event       EventType  `json:"event"`
	ChatSender  string     `json:"chatSender"`
	ChatMessage string     `json:"chatMessage"`
	Players     []string   `json:"players"`
	HostId      string     `json:"hostId"`
	Board       [][]string `json:"board"`
}

type ClientProtocol struct {
	Event       EventType `json:"event"`
	ChatMessage string    `json:"chatMessage"`
}

type LobbyCard struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	PlayersNum int    `json:"playersNum"`
}

func (l *Lobby) StartGame() {
	l.Game = &game.Game{}
	l.EventChannel <- ServerProtocol{Event: GAMEMOVE}
}

func (l Lobby) Card() LobbyCard {
	return LobbyCard{l.Id, l.Name, len(l.Players)}
}
