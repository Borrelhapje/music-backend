package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var dbpath string

func init() {
	flag.StringVar(&dbpath, "dbpath", "library.bla", "Path to the sqlite database file")
}

func main() {
	flag.Parse()
	db, err := sql.Open("sqlite", dbpath)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = ListItems(db); err != nil {
		log.Fatal(err)
	}
	http.Handle("/list", listEndpoint(db))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

type Song struct {
	Id          int
	Path        string
	Title       string
	added       float64
	EpochMillis int64
	Artist      string
	Album       string
}

func listEndpoint(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := ListItems(db)
		if err != nil {
			log.Default().Println(err)
			w.WriteHeader(500)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		encoder := json.NewEncoder(w)
		encoder.Encode(items)
	}
}

func ListItems(db *sql.DB) ([]Song, error) {
	rows, err := db.Query("SELECT id, path, title, added, artist, album FROM items")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Song{}
	for rows.Next() {
		var song Song
		err := rows.Scan(&song.Id, &song.Path, &song.Title, &song.added, &song.Artist, &song.Album)
		if err != nil {
			return nil, err
		}
		song.EpochMillis = time.Unix(int64(song.added), 0).UnixMilli()
		index := strings.Index(song.Path, "/complete/music")
		song.Path = song.Path[index:len(song.Path)]
		items = append(items, song)
	}
	return items, nil
}
