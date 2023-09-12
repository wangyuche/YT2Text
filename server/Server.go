package main

import (
	"bufio"
	"encoding/json"
	"html/template"
	"io"
	"os"
	"os/exec"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/kkdai/youtube/v2"
	"github.com/wangyuche/goutils/log"
)

func Setup() *fiber.App {
	engine := html.New("./web", ".html")
	engine.AddFunc(
		"unescape", func(s string) template.HTML {
			return template.HTML(s)
		},
	)
	engine.Reload(true)
	app := fiber.New(fiber.Config{
		Views: engine,
	})
	WSInit(app)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{})
	})
	app.Static("/", "./web/")
	return app
}

func main() {
	log.New(log.LogType(os.Getenv("LogType")))
	app := Setup()
	log.Info("Listen Port" + os.Getenv("Port"))
	app.Listen(os.Getenv("Port"))
}

var Clients map[*websocket.Conn]*Client = make(map[*websocket.Conn]*Client)

func WSInit(app *fiber.App) {
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		Clients[c] = &Client{
			c:                   c,
			cmd:                 "idle",
			downloadcomplete:    make(chan bool, 1),
			text:                make(chan string),
			getcaptionscomplete: make(chan bool, 1),
		}
		Clients[c].ReadMessage()
	}))
}

type YTReq struct {
	Cmd  string `json:"cmd"`
	Data string `json:"data"`
}

type YTRep struct {
	Cmd  string `json:"cmd"`
	Data string `json:"data"`
}

type Client struct {
	c                   *websocket.Conn
	cmd                 string
	downloadcomplete    chan bool
	text                chan string
	getcaptionscomplete chan bool
}

func (this *Client) ReadMessage() {
	for {
		var (
			msg []byte
			err error
		)
		if _, msg, err = this.c.ReadMessage(); err != nil {
			log.Error(err.Error())
			delete(Clients, this.c)
			break
		}
		var req YTReq
		log.Info(string(msg))
		err = json.Unmarshal(msg, &req)
		if err != nil {
			log.Error(err.Error())
		}

		go func() {
			for {
				select {
				case r := <-Clients[this.c].downloadcomplete:
					if r == true {
						Clients[this.c].cmd = "getcaptions"
						var rep YTRep
						rep.Cmd = Clients[this.c].cmd
						jsondata, _ := json.Marshal(rep)
						Clients[this.c].WriteMessage(string(jsondata))
						go getcaptions(Clients[this.c].text, Clients[this.c].getcaptionscomplete)
					} else {
						Clients[this.c].cmd = "idle"
						var rep YTRep
						rep.Cmd = Clients[this.c].cmd
						jsondata, _ := json.Marshal(rep)
						Clients[this.c].WriteMessage(string(jsondata))
					}
				case d := <-Clients[this.c].text:
					var rep YTRep
					rep.Cmd = "getcaptions"
					rep.Data = d
					jsondata, _ := json.Marshal(rep)
					Clients[this.c].WriteMessage(string(jsondata))
				case <-Clients[this.c].getcaptionscomplete:
					Clients[this.c].cmd = "idle"
					var rep YTRep
					rep.Cmd = Clients[this.c].cmd
					jsondata, _ := json.Marshal(rep)
					Clients[this.c].WriteMessage(string(jsondata))
				}
			}
		}()

		switch req.Cmd {
		case "downloadyt":
			if Clients[this.c].cmd == "idle" {
				go download_ytvideo(req.Data, Clients[this.c].downloadcomplete)
				Clients[this.c].cmd = "downloadyt"
				Clients[this.c].WriteMessage(string(msg))
			}
		case "status":
			var rep YTRep
			rep.Cmd = "status"
			rep.Data = Clients[this.c].cmd
			jsondata, _ := json.Marshal(rep)
			Clients[this.c].WriteMessage(string(jsondata))
		}

	}
}

func (this *Client) WriteMessage(data string) {
	var (
		err error
	)
	if err = this.c.WriteMessage(websocket.TextMessage, []byte(data)); err != nil {
		delete(Clients, this.c)
	}
}

func getcaptions(text chan string, getcaptionscomplete chan bool) {
	cmd := exec.Command("whisper", "/Users/arieswang/Documents/git/YT2Text/server/video.mp4", "--fp16", "False")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error(err.Error())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Error(err.Error())
	}
	err = cmd.Start()
	if err != nil {
		log.Error(err.Error())
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		m := scanner.Text()
		log.Info(m)
		text <- m
	}
	scannererr := bufio.NewScanner(stderr)
	for scannererr.Scan() {
		e := scannererr.Text()
		log.Info(e)
		text <- e
	}
	cmd.Wait()
	getcaptionscomplete <- true
}

func download_ytvideo(url string, downloadcomplete chan bool) {
	videoID := url
	client := youtube.Client{}

	video, err := client.GetVideo(videoID)
	if err != nil {
		log.Error(err.Error())
		downloadcomplete <- false
		return
	}

	formats := video.Formats.WithAudioChannels()
	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		log.Error(err.Error())
		downloadcomplete <- false
		return
	}
	err = os.Remove("video.mp4")
	if err != nil {
		log.Error(err.Error())
	}
	file, err := os.Create("video.mp4")
	if err != nil {
		log.Error(err.Error())
		downloadcomplete <- false
		return
	}
	defer file.Close()
	_, err = io.Copy(file, stream)
	if err != nil {
		log.Error(err.Error())
		downloadcomplete <- false
		return
	}
	downloadcomplete <- true
}
