package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/websocket"
	"github.com/jasonsoft/napnap"
	"github.com/kr/pty"
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

	//set and start command
	cmd := exec.Command("/bin/bash", "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm")
	tty, err := pty.Start(cmd)
	if err != nil {
		log.Print(err)
		return
	}

	defer func() {
		cmd.Process.Kill()
		cmd.Process.Wait()
		tty.Close()
		conn.Close()
	}()

	go func() {
		//server reader and websocket writer
		for {
			buf := make([]byte, 4096)
			_, err := tty.Read(buf)
			if err != nil {
				log.Print(err)
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
			return
		}

		_, err = io.Copy(tty, reader)
		if err != nil {
			log.Print(err)
			return
		}
	}
}

func main() {
	var listen = flag.String("listen", ":12345", "Host:port to listen on")
	nap := napnap.New()
	flag.Parse()
	router := napnap.NewRouter()
	router.Get("/term", napnap.WrapHandler(http.HandlerFunc(handleWebsocket)))
	nap.Use(router)
	httpengine := napnap.NewHttpEngine(*listen)
	log.Fatal(nap.Run(httpengine))
}
