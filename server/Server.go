package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"html/template"
	"io"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"cloud.google.com/go/translate"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/kkdai/youtube/v2"
	"github.com/wangyuche/goutils/log"
	"golang.org/x/text/language"
)

type FileInfo struct {
	url    string
	name   string
	status string
}

var ytqueen map[string]FileInfo
var mutex sync.RWMutex
var source string

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
	ytqueen = make(map[string]FileInfo)
	log.New(log.LogType(os.Getenv("LogType")))
	if len(os.Args) > 1 {
		log.Info("Start")
		flag.StringVar(&source, "s", "yt.txt", "YT URL")
		flag.Parse()
		mutex.Lock()
		file, err := os.Open(source)
		if err != nil {
			log.Fail(err.Error())
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			u, err := url.Parse(scanner.Text())
			if err != nil {
				log.Fail(err.Error())
				break
			}
			m, err := url.ParseQuery(u.RawQuery)
			if err != nil {
				log.Fail(err.Error())
				break
			}
			var f FileInfo
			f.url = scanner.Text()
			f.name = m["v"][0]
			f.status = "idle"
			ytqueen[m["v"][0]] = f
		}
		if err := scanner.Err(); err != nil {
			log.Fail(err.Error())
		}
		mutex.Unlock()
		ProcessQueen()
	} else {
		app := Setup()
		log.Info("Listen Port" + os.Getenv("Port"))
		go ProcessQueen()
		app.Listen(os.Getenv("Port"))
	}
}

func ProcessQueen() {
	var downloadcomplete chan string = make(chan string, 5)
	var getcaptionscomplete chan string = make(chan string, 5)
	go func() {
		for {
			select {
			case k := <-downloadcomplete:
				if k != "" {
					mutex.Lock()
					y := ytqueen[k]
					y.status = "downloadcomplete"
					ytqueen[k] = y
					mutex.Unlock()
				}
			case k := <-getcaptionscomplete:
				log.Info("Complete:" + k)
				if k != "" {
					mutex.Lock()
					y := ytqueen[k]
					y.status = "complete"
					ytqueen[k] = y
					mutex.Unlock()
					var zh string = ""
					var i int = 0
					file, err := os.Open("./data/" + k + ".srt")
					if err != nil {
						log.Error(err.Error())
						break
					}
					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						if i != 2 {
							zh = zh + scanner.Text() + "\r\n"
							i++
							if i == 4 {
								i = 0
							}
						} else {
							zh = zh + translateText(scanner.Text()) + "\r\n"
							i++
						}
					}
					log.Info(zh)
					file.Close()
					f, err := os.Create("./data/" + k + "_zh.srt")

					if err != nil {
						log.Fail(err.Error())
						f.Close()
					}
					_, err2 := f.WriteString(zh)
					if err2 != nil {
						log.Fail(err.Error())
						f.Close()
					}
					f.Close()

				}
			}
			time.Sleep(2 * time.Second)
		}
	}()

	for {
		var c map[string]FileInfo = map[string]FileInfo{}
		mutex.Lock()
		for k, v := range ytqueen {
			c[k] = v
		}
		mutex.Unlock()
		for k, v := range c {
			if v.status == "idle" {
				mutex.Lock()
				y := ytqueen[k]
				y.status = "downloading"
				ytqueen[k] = y
				mutex.Unlock()
				go download_ytvideo(k, downloadcomplete)
			}
			if v.status == "downloadcomplete" {
				mutex.Lock()
				y := ytqueen[k]
				y.status = "captionsing"
				ytqueen[k] = y
				mutex.Unlock()
				getcaptions(getcaptionscomplete, k)
			}
		}
		time.Sleep(2 * time.Second)
	}
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
			c: c,
		}
		Clients[c].ReadMessage()
	}))
}

type YTReq struct {
	Cmd  string `json:"cmd"`
	Data string `json:"data"`
}

type YTRep struct {
	Cmd  string     `json:"cmd"`
	Data []FileInfo `json:"data"`
}

type Client struct {
	c *websocket.Conn
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

		switch req.Cmd {
		case "addqueen":
			u, err := url.Parse(req.Data)
			if err != nil {
				log.Error(err.Error())
				break
			}
			m, err := url.ParseQuery(u.RawQuery)
			if err != nil {
				log.Error(err.Error())
				break
			}
			mutex.Lock()
			_, ok := ytqueen[m["v"][0]]
			if !ok {
				var f FileInfo
				f.url = req.Data
				f.name = m["v"][0]
				f.status = "idle"
				ytqueen[m["v"][0]] = f
			}
			mutex.Unlock()
		case "getqueen":
			var v []FileInfo = make([]FileInfo, 0)
			mutex.Lock()
			for _, value := range ytqueen {
				v = append(v, value)
			}
			mutex.Unlock()
			var rep YTRep
			rep.Cmd = "getqueen"
			rep.Data = v
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

func getcaptions(getcaptionscomplete chan string, file string) {
	log.Info("getcaptions:" + file)
	cmd := exec.Command("whisper", "./data/"+file+".mp4", "--fp16", "False", "--output_dir", "./data")
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
	}
	scannererr := bufio.NewScanner(stderr)
	for scannererr.Scan() {
		e := scannererr.Text()
		log.Info(e)
	}
	cmd.Wait()
	getcaptionscomplete <- file
}

func download_ytvideo(url string, downloadcomplete chan string) {
	videoID := url
	client := youtube.Client{}

	video, err := client.GetVideo(videoID)
	if err != nil {
		log.Error(err.Error())
		downloadcomplete <- ""
		return
	}

	formats := video.Formats.WithAudioChannels()
	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		log.Error(err.Error())
		downloadcomplete <- ""
		return
	}
	file, err := os.Create("data/" + url + ".mp4")
	if err != nil {
		log.Error(err.Error())
		downloadcomplete <- ""
		return
	}
	defer file.Close()
	_, err = io.Copy(file, stream)
	if err != nil {
		log.Error(err.Error())
		downloadcomplete <- ""
		return
	}
	downloadcomplete <- url
}

func translateText(text string) string {
	ctx := context.Background()

	lang, err := language.Parse("zh-TW")
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	client, err := translate.NewClient(ctx)
	if err != nil {
		log.Error(err.Error())
		return ""
	}
	defer client.Close()

	resp, err := client.Translate(ctx, []string{text}, lang, nil)
	if err != nil {
		log.Error(err.Error())
		return ""
	}
	if len(resp) == 0 {
		log.Error(err.Error())
		return ""
	}
	return resp[0].Text
}
