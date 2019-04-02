# ffmpeg ws-relay
This is a simple example show up how to streaming video by broadcasting image frames via websocket.


## Build

1. install go: [golang](https://golang.org/dl/)
2. clone this repo : `git clone https://github.com/cs8425/ffmpeg-ws-relay.git`
3. build: `go build -o ws-relay ws-relay.go`
4. run with ffmpeg, see [Usage example](#usage-example)
5. open browser to: [http://127.0.0.1:8080/](http://127.0.0.1:8080/)


## Usage example:

* transcode a file to websocket via png format:
  * `ffmpeg -re -i v01.mp4 -c:v png -f image2pipe - | ./ws-relay -l :8080 -s png`
* transcode a file to websocket via jpg format:
  * `ffmpeg -re -i v01.mp4 -s 1280x720 -c:v mjpeg -qscale:v 2 -f image2pipe - | ./ws-relay -l :8080`
  * `-s 1280x720` : output size
  * `-qscale:v 2` : jpeg quality, range 2~31, 31 is the worst quality


