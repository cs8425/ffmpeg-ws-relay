// for build: go build -o ws-relay ws-relay.go
/*

usage:

ffmpeg -re -i v01.mp4 -c:v png -f image2pipe - | ./ws-relay -l :8081 -s png
ffmpeg -re -i v01.mp4 -s 1280x720 -c:v mjpeg -qscale:v 2 -f image2pipe - | ./ws-relay -l :8081

*/
package main

import (
	"net/http"
	"log"
	"flag"
	"errors"

	"bufio"
	"bytes"
	"os"

	ws "github.com/gorilla/websocket"
)

var localAddr = flag.String("l", ":8080", "")

var wsComp = flag.Bool("wscomp", false, "ws compression")
var verbosity = flag.Int("v", 3, "verbosity")

var queue = flag.Int("q", 1, "ws queue")

var split = flag.String("s", "jpg", "image type")

var upgrader = ws.Upgrader{ EnableCompression: false } // use default options

var newclients chan *WsClient
var bufCh chan []byte

type WsClient struct {
	*ws.Conn
	data chan []byte
	die bool
}
func NewWsClient(c *ws.Conn) (*WsClient) {
	return &WsClient{ c, make(chan []byte, *queue), false }
}
func (c *WsClient) Send(buf []byte) (error) {
	if c.die {
		return errors.New("ws connection die")
	}

	select {
	case <- c.data:
	default:
	}
	c.data <- buf

	return nil
}
func (c *WsClient) worker() {
	for {
		buf := <- c.data
		//Vln(5, "[dbg]worker()", &c, len(buf))
		err := c.WriteMessage(ws.BinaryMessage, buf)
		if err != nil {
			c.Close()
			c.die = true
			return
		}
	}
}

func broacast() {
	clients := make(map[*WsClient]*WsClient, 0)

	for {
		data := <- bufCh
		//Vln(5, "[dbg]broacast()", len(data))
		for _, c := range clients {
			err := c.Send(data)
			if err != nil {
				delete(clients, c)
				Vln(3, "[ws][client]removed", c.RemoteAddr(), len(clients))
			}
		}
		for len(newclients) > 0 {
			c := <-newclients
			clients[c] = c
			Vln(3, "[ws][client]added", c.RemoteAddr())
		}
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Vln(2, "[ws]upgrade failed:", err)
		return
	}
	defer c.Close()

	Vln(3, "[ws][client]connect", c.RemoteAddr())
	client := NewWsClient(c)
	newclients <- client

	client.worker()

	Vln(3, "[ws][client]disconnect", c.RemoteAddr())
}

func main() {
	log.SetFlags(log.Ldate|log.Ltime)
	flag.Parse()

	upgrader.EnableCompression = *wsComp
	Vf(1, "ws EnableCompression = %v\n", *wsComp)
	Vf(1, "server Listen @ %v\n", *localAddr)
	Vf(1, "input image type = %v\n", *split)

	newclients = make(chan *WsClient, 16)
	bufCh = make(chan []byte, 1)
	go broacast()

	go connCam()

	http.HandleFunc("/ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir("./")))

	err := http.ListenAndServe(*localAddr, nil)
	if err != nil {
		Vln(1, "server listen error:", err)
	}
}

func connCam() {
	markMap := map[string][]byte{
		"jpg": []byte("\xFF\xD9"),
		"png": []byte("IEND\xAE\x42\x60\x82"),
	}

	mark, ok := markMap[*split]
	if !ok {
		Vln(2, "[input][type]err:", *split)
		return
	}
	Vln(5, "[dbg]mark", len(mark), mark)
	reader := bufio.NewReaderSize(os.Stdin, 8*1024*1024)
	for {
		buf, err := readEnd(reader, mark)
		n := len(buf)
		//Vln(5, "[pipe][recv]", n, err)
		if err != nil {
			Vln(2, "[pipe][recv]err:", err)
			return
		}

		Vln(5, "[dbg]connCam()", n, buf[:8])
		pack := make([]byte, n, n)
		copy(pack, buf[0:n])

		// broacast frame
		bufCh <- pack

		// do what you want with the frame
		// ...
	}
}

func readEnd(r *bufio.Reader, delim []byte) (line []byte, err error) {
	mark := delim[len(delim)-1]
	for {
		s, err := r.ReadBytes(mark)
		if err != nil {
			return nil, err
		}

		line = append(line, s...)
		if bytes.HasSuffix(line, delim) {
			return line, nil
		}
	}
}

func Vln(level int, v ...interface{}) {
	if level <= *verbosity {
		log.Println(v...)
	}
}
func Vf(level int, format string, v ...interface{}) {
	if level <= *verbosity {
		log.Printf(format, v...)
	}
}

