package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("/bin/bash", "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm")

	tty, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Process.Wait()
		tty.Close()
		conn.Close()
	}()

	go func() {
		for {
			buf := make([]byte, 1024)
			read, err := tty.Read(buf)
			if err != nil {
				panic(err)
			}
			conn.WriteMessage(websocket.BinaryMessage, buf[:read])
		}
	}()

	for {
		_, reader, err := conn.NextReader()
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(tty, reader)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	var listen = flag.String("listen", ":12345", "Host:port to listen on")

	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/term", handleWebsocket)
	log.Fatal(http.ListenAndServe(*listen, r))
}
