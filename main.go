package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"github.com/labstack/echo/v4"
	"github.com/coder/websocket"
)

type Client struct {
	Nickname  	string
	conn      	*websocket.Conn
	ctx 		context.Context
}

type Message struct {
	From 		string
	Content 	string
	SentAt 		string
}

var (
	clients 	map[*Client]bool 	= make(map[*Client]bool)
	joinCh  	chan *Client      	= make(chan *Client)
	broadcastCh chan Message 		= make(chan Message)
	notifyCh	chan Message		= make(chan Message)
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	nickname := r.URL.Query().Get("nickname")
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	go broadcast()
	go joiner() // Criando uma nova rotina 

	// Client connected with webSocket
	newClient := Client{Nickname: nickname, conn: conn, ctx: r.Context()}
	joinCh <- &newClient
	reader(&newClient)
}

func reader(newClient *Client) {
	for {
		_, data, err := newClient.conn.Read(newClient.ctx)
		if err != nil {
			log.Println("Encerrando conexão do cliente")
			delete(clients, newClient)
			broadcastCh <- Message{From: newClient.Nickname, Content: newClient.Nickname + " has been disconnected", SentAt: time.Now().Format("02-01-2006 15:04:05")}
			break
		}
		
		var msgRec Message
		json.Unmarshal(data, &msgRec)

		broadcastCh <- Message{From: msgRec.From, Content: msgRec.Content, SentAt: time.Now().Format("02-01-2006 15:04:05")}

		notifyCh <- Message{From: "System", Content: "Nova mensagem recebida de " + newClient.Nickname, SentAt: time.Now().Format("02-01-2006 15:04:05")}
	}
}

func joiner() {
		for newClient := range joinCh {
			clients[newClient] = true

		broadcastCh <- Message{From: newClient.Nickname, Content: "User " + newClient.Nickname + " has been connected.", SentAt: time.Now().Format("02-01-2006 15:04:05")}
	}
}

func broadcast() {
	for {
		select {
		case newMsg := <- broadcastCh:
			sendToAll(newMsg)

		case notify := <- notifyCh:
			sendToAll(notify)
		}
	}
}

func sendToAll(msg Message) {
	for client := range clients {
		data, _ := json.Marshal(msg)
		client.conn.Write(client.ctx, websocket.MessageText, data)
	}
}

func main() {

	e := echo.New()

	e.Static("/", "./public")

	e.GET("/clients", func(ctx echo.Context) error {
		var listaClient []*Client
		for client := range clients {
			listaClient = append(listaClient, client)
		}
		return ctx.JSON(http.StatusOK, listaClient)
	})

	e.GET("/ws", func(ctx echo.Context) error {
		wsHandler(ctx.Response(), ctx.Request())
		return nil
	})

	log.Println("Server initialized on port 8080....")
	e.Logger.Fatal(e.Start(":8080"))
}