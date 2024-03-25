package lobby

import (
	"encoding/json"
	"log"
	"sync"
)

var lobbies map[string]*Lobby = make(map[string]*Lobby)

func CreateNewLobby(id, name, password, userId string) {
	lobbies[id] = &Lobby{
		Id:                 id,
		Name:               name,
		Password:           password,
		Players:            make([]*Player, 2),
		EventBroadcastChan: make(chan ServerProtocol, 10),
		mutex:              &sync.Mutex{},
		Game:               &Game{},
	}
	go lobbies[id].StartLobby()

	// EmitEvent()
}

type LobbyCard struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Player1   string `json:"player1"`
	Player2   string `json:"player2"`
	IsPlaying bool   `json:"isPlaying"`
	IsLocked  bool   `json:"isLocked"`
}

func (l Lobby) Card() LobbyCard {
	p1, p2 := "empty", "empty"
	if l.Players[0] != nil {
		p1 = l.Players[0].UserId
	}
	if l.Players[1] != nil {
		p2 = l.Players[1].UserId
	}
	return LobbyCard{
		l.Id,
		l.Name,
		p1, p2,
		l.Game.playing,
		l.Password != "",
	}
}

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
	eventSubscribers[id] = make(chan string, 1)
	eventSubscribers[id] <- EventData()
	return eventSubscribers[id]
}

func UnSubscribe(id string) {
	close(eventSubscribers[id])
	delete(eventSubscribers, id)
}
