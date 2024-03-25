package lobby

import (
	"fmt"
	"log"
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type EventType int

const (
	CHAT EventType = iota
	LOBBYINFOCHANGE
	STARTGAME
	GAMEMOVE
)

type ServerProtocol struct {
	Event       EventType        `json:"event"`
	ChatSender  string           `json:"chatSender"`
	ChatMessage string           `json:"chatMessage"`
	Players     []string         `json:"players"`
	HostId      string           `json:"hostId"`
	Board       [9][9]CellStatus `json:"board"`
	LobbyName   string           `json:"lobbyName"`
}

type ClientProtocol struct {
	Event       EventType `json:"event"`
	ChatMessage string    `json:"chatMessage"`
	Point       `json:"point"`
	Pass        bool   `json:"pass"`
	LobbyName   string `json:"lobbyName"`
}

type Player struct {
	UserId string
	*websocket.Conn
}

type Lobby struct {
	Id       string
	Name     string
	Password string

	HostId      string
	Players     []*Player
	PlayerOrder [2]string
	mutex       *sync.Mutex

	EventBroadcastChan chan ServerProtocol

	*Game
}

func (l *Lobby) StartLobby() {
	for v := range l.EventBroadcastChan {
		for _, p := range l.Players {
			if p != nil {
				p.Conn.WriteJSON(v)
			}
		}
	}
}

func (l *Lobby) StartGame() {
	if rand.New(rand.NewSource(time.Now().UnixNano())).Int()%2 == 0 {
		l.PlayerOrder = [2]string{l.Players[0].UserId, l.Players[1].UserId}
	} else {
		l.PlayerOrder = [2]string{l.Players[1].UserId, l.Players[0].UserId}
	}

	l.Game.start()

	l.EventBroadcastChan <- ServerProtocol{Event: STARTGAME, Board: l.Game.board}
	l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: "Game start"}
	l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s's turn.", l.getCurrentPlayerId())}

	EmitEvent()
}

func DeleteLobby(id string) {
	l, o := lobbies[id]
	if !o {
		log.Fatal("deletelobby: lobby not found")
		return
	}

	close(l.EventBroadcastChan)
	delete(lobbies, l.Id)

	EmitEvent()
}

// 1명만있으면 무조건 [0] 위치에 가게 하기?
func (l *Lobby) Join(userId string, conn *websocket.Conn) {
	l.mutex.Lock()

	i := slices.Index(l.Players, nil)
	if i == -1 {
		l.mutex.Unlock()
		conn.Close()
		return
	}

	l.Players[i] = &Player{
		UserId: userId,
		Conn:   conn,
	}

	if l.HostId == "" {
		l.HostId = userId
		l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s is the host.", l.HostId)}
	}

	l.LobbyInfoChangeEvent()
	l.EventBroadcastChan <- ServerProtocol{Event: GAMEMOVE, Board: l.Game.board}
	l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s has joined the lobby.", userId)}

	l.mutex.Unlock()

	EmitEvent()

	go func() {
		for {
			p := ClientProtocol{}
			if err := l.Players[i].Conn.ReadJSON(&p); err != nil {
				l.Leave(userId)
				return
			}

			switch p.Event {
			case CHAT:
				l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s : %s", userId, p.ChatMessage)}
			case STARTGAME:
				if userId == l.HostId {
					l.StartGame()
				}
			case GAMEMOVE:
				if !l.Game.playing || l.getCurrentPlayerId() != userId {
					continue
				}

				if p.Pass {
					l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s pass.", l.getCurrentOpponentId())}
					if f, w := l.Game.pass(); f {
						if w == 2 {
							l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: "Game Set. Draw."}
						} else {
							l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("Game Set. %s won.", l.PlayerOrder[w])}
						}
						EmitEvent()
					} else {
						l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s's turn.", l.getCurrentPlayerId())}
					}
					continue
				}

				if f, w := l.Game.moveCurrentTurn(p.Point); f {
					if w == 2 {
						l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: "Game Set. Draw."}
					}
					l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("Game Set. %s won.", l.PlayerOrder[w])}
					EmitEvent()
				} else {
					l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s's turn.", l.getCurrentPlayerId())}
				}
				l.EventBroadcastChan <- ServerProtocol{Event: GAMEMOVE, Board: l.Game.board}
			}
		}
	}()
}

func (l *Lobby) Leave(userId string) {
	l.mutex.Lock()

	i := slices.IndexFunc(l.Players, func(p *Player) bool {
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

	if l.HostId == userId {
		o := slices.IndexFunc(l.Players, func(p *Player) bool { return p != nil })
		if o == -1 {
			DeleteLobby(l.Id)
			l.mutex.Unlock()
			return
		} else {
			l.HostId = l.Players[o].UserId
			l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s is the host.", l.HostId)}
		}
	}

	l.mutex.Unlock()

	l.LobbyInfoChangeEvent()
	l.EventBroadcastChan <- ServerProtocol{Event: CHAT, ChatMessage: fmt.Sprintf("%s has gone away.", userId)}

	EmitEvent()
}

func (l Lobby) LobbyInfoChangeEvent() {
	players := []string{}
	for _, v := range l.Players {
		if v == nil {
			players = append(players, "")
			continue
		}
		players = append(players, v.UserId)
	}
	l.EventBroadcastChan <- ServerProtocol{
		Event:     LOBBYINFOCHANGE,
		Players:   players,
		HostId:    l.HostId,
		LobbyName: l.Name,
	}
}

func (l Lobby) getCurrentPlayerId() string {
	return l.PlayerOrder[(l.Game.turn-1)%2]
}

func (l Lobby) getCurrentOpponentId() string {
	return l.PlayerOrder[(l.Game.turn)%2]
}
