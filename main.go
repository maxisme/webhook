package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"gopkg.in/validator.v2"
)

const configPath = "./config.json"

type endpoint = string
type WebhookConfig struct {
	Cmd   string   `json:"cmd" validate:"nonzero"`
	Token string   `json:"token" validate:"min=20"`
	Args  []string `json:"args"`
}

func writeError(w http.ResponseWriter, err string, code int) {
	log.Println(err)
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err})
}

func ConfigFromPath(path string) (confs map[endpoint]WebhookConfig, err error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &confs)
	if err != nil {
		return
	}
	for endpoint, conf := range confs {
		err = validator.Validate(conf)
		if err != nil {
			err = errors.New(endpoint + " " + err.Error())
			return
		}
	}
	return
}

func CreateConfHandler(conf WebhookConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeError(w, "invalid method", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		if r.Form.Get("token") != conf.Token {
			log.Printf("Invalid token! '%s'\n", r.Form.Get("token"))
			return
		}

		// parse form arguments into command
		var args []interface{}
		for _, arg := range conf.Args {
			args = append(args, r.Form.Get(arg))
		}
		cmd := fmt.Sprintf(conf.Cmd, args...)

		// validate executable
		splitCmds := strings.Split(cmd, " ")
		executable, err := exec.LookPath(splitCmds[0])
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// run command
		c := exec.Command(executable, splitCmds[1:]...)
		log.Println("Running: " + c.String())
		out, err := c.CombinedOutput()
		if err != nil {
			writeError(w, string(out)+" "+err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(out)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}

func main() {
	// file listener
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// web server
	go func() {
		for {
			r := chi.NewRouter()
			r.Use(middleware.Logger)

			configs, err := ConfigFromPath(configPath)
			if err != nil {
				log.Println(err)
				time.Sleep(10 * time.Second)
			} else {
				// add handlers from config
				for endpoint, conf := range configs {
					r.HandleFunc(endpoint, CreateConfHandler(conf))
				}
				r.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {})
				srv := http.Server{Addr: ":8080", Handler: r}

				// listen for change of configPath and restart server
				err = watcher.Add(configPath)
				if err != nil {
					log.Fatal(err)
				}
				go func(srv *http.Server) {
					select {
					case event, ok := <-watcher.Events:
						if !ok {
							return
						}
						log.Println("event:", event)
						if err := srv.Close(); err != nil {
							log.Println(err)
						}
					case err, ok := <-watcher.Errors:
						if !ok {
							return
						}
						log.Println(err)
					}
				}(&srv)

				log.Println("started server")
				if err := srv.ListenAndServe(); err != nil {
					log.Println(err.Error())
				}
			}
		}
	}()

	select {}
}
