package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/kr/pty"
)

type windowSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
	X    uint16
	Y    uint16
}

func serverConn(conn net.Conn) {
	//set and start command
	cmd := exec.Command("/bin/bash", "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm")
	tty, err := pty.Start(cmd)
	if err != nil {
		log.Print(err.Error())
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
			_, err := io.Copy(conn, tty)
			if err != nil {
				log.Print(err.Error())
				return
			}
		}
	}()

	for {
		//server writer and websocket reader
		for {
			dataTypeBuf := make([]byte, 1)
			_, err := conn.Read(dataTypeBuf)
			if err != nil {
				log.Print(err.Error())
				return
			}

			switch dataTypeBuf[0] {
			case 0:
				_, err := io.Copy(tty, conn)
				if err != nil {
					log.Print(err.Error())
					return
				}
			case 1:
				decoder := json.NewDecoder(conn)
				resizeMessage := windowSize{}
				err := decoder.Decode(&resizeMessage)
				if err != nil {
					log.Print(err.Error())
					continue
				}
				_, _, errno := syscall.Syscall(
					syscall.SYS_IOCTL,
					tty.Fd(),
					syscall.TIOCSWINSZ,
					uintptr(unsafe.Pointer(&resizeMessage)),
				)
				if errno != 0 {
					log.Print(errno.Error())
					return
				}
			default:

			}
		}
	}
}

func main() {
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	l, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer l.Close()
	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err.Error())
		}
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go serverConn(conn)
	}
}
