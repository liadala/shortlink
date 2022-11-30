package main

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"shortlink/database"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	templ  *template.Template
	static fs.FS
	port   int
)

//go:embed web/*
var fshtml embed.FS

//go:embed static/*
var fsstatic embed.FS

func init() {
	// log handler to file and console out
	logFile, err := os.OpenFile("shorturl.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	flag.IntVar(&port, "port", 8473, "listen port")
	flag.Parse()

	// Prepare Filesystem
	static, err = fs.Sub(fsstatic, "static")
	if err != nil {
		log.Fatal(err)
	}

	templ, err = template.ParseFS(fshtml, "web/*")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	db := database.New()
	router := mux.NewRouter()

	router.HandleFunc("/s/{id}", func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		url, err := db.GetURL(vars["id"])
		if err != nil {
			log.Println(err)
			response, err := runTemplate(templ, "index.html", url)
			if err != nil {
				rw.Write([]byte(fmt.Sprint(err)))
			}
			rw.Write([]byte(response))
			return
		}

		response, err := runTemplate(templ, "short.html", url)
		if err != nil {
			rw.Write([]byte(fmt.Sprint(err)))
		}
		rw.Write([]byte(response))
	}).Methods("GET")

	router.HandleFunc("/add", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		r.Body = http.MaxBytesReader(rw, r.Body, 128*1024)

		var post struct {
			Url string `json:"url"`
		} = struct {
			Url string "json:\"url\""
		}{}

		err := json.NewDecoder(r.Body).Decode(&post)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(struct {
				Code    int    `json:"status"`
				Message string `json:"message"`
			}{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprint(err),
			})
			return
		}

		_, err = url.ParseRequestURI(post.Url)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(struct {
				Code    int    `json:"status"`
				Message string `json:"message"`
			}{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprint(err),
			})
			return
		}

		var ip string = r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = strings.Split(r.RemoteAddr, ":")[0]
		}
		err = db.WriteShortURL(post.Url, ip)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(rw).Encode(struct {
				Code    int    `json:"status"`
				Message string `json:"message"`
			}{
				Code:    http.StatusInternalServerError,
				Message: "Something went wrong",
			})
			return
		}

		short, err := db.GetShort(post.Url)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(rw).Encode(struct {
				Code    int    `json:"status"`
				Message string `json:"message"`
			}{
				Code:    http.StatusInternalServerError,
				Message: "Something went wrong",
			})
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(struct {
			Short string `json:"short"`
		}{
			Short: short,
		})
	}).Methods("POST")

	router.HandleFunc("/favicon.ico", func(rw http.ResponseWriter, r *http.Request) {
		f, err := fsstatic.ReadFile("static/favicon.ico")
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		rw.Write(f)
	})

	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		response, err := runTemplate(templ, "index.html", "")
		if err != nil {
			rw.Write([]byte(fmt.Sprint(err)))
		}
		rw.Write([]byte(response))
	}).Methods("GET")

	router.PathPrefix("/static").Handler(http.StripPrefix("/static", func(fs http.Handler) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Add("Cache-Control", "public, max-age=31536000")
			fmt.Println(r.URL.Path)
			fs.ServeHTTP(rw, r)
		}
	}(http.FileServer(http.FS(static))))).Methods("GET")

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadTimeout:       time.Second * 15,
		WriteTimeout:      time.Second * 15,
		IdleTimeout:       time.Second * 30,
		ReadHeaderTimeout: time.Second * 15,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
		Handler: handlers.CompressHandlerLevel(handlers.CORS(
			handlers.AllowedHeaders([]string{"X-Requested-With"}),
			handlers.AllowedOrigins([]string{os.Getenv("ORIGIN_ALLOWED")}),
			handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
		)(router), gzip.BestCompression),
	}

	log.Println("Listen:", fmt.Sprintf("127.0.0.1:%d", port))
	log.Fatal(srv.ListenAndServe())
}

func runTemplate(templ *template.Template, name string, data interface{}) (string, error) {
	buf := new(bytes.Buffer)
	err := templ.ExecuteTemplate(buf, name, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
