package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jasonsoft/napnap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	//upgrade http to websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
		return
	}
	defer conn.Close()
	serverConn, err := net.Dial("tcp", ":8081")
	if err != nil {
		log.Print(err)
		conn.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
		return
	}
	defer serverConn.Close()

	go func() {
		defer func() {
			conn.Close()
			serverConn.Close()
		}()

		//server reader and websocket writer
		for {
			buf := make([]byte, 4096)
			_, err := serverConn.Read(buf)
			if err != nil {
				log.Print(err)
				conn.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
				return
			}
			conn.WriteMessage(websocket.BinaryMessage, buf)
		}
	}()

	for {
		//server writer and websocket reader
		_, reader, err := conn.NextReader()
		if err != nil {
			log.Print(err)
			conn.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
			return
		}

		_, err = io.Copy(serverConn, reader)
		if err != nil {
			log.Print(err)
			conn.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
			return
		}
	}
}

func main() {
	var listen = flag.String("listen", ":8080", "Host:port to listen on")
	nap := napnap.New()
	flag.Parse()
	router := napnap.NewRouter()
	router.Get("/term", napnap.WrapHandler(http.HandlerFunc(handleWebsocket)))
	nap.Use(router)
	httpengine := napnap.NewHttpEngine(*listen)
	log.Fatal(nap.Run(httpengine))
}
