package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
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
		rows, err := db.Query("SELECT id, path, title, added, artist, album FROM items")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		defer rows.Close()
		items := []Song{}
		for rows.Next() {
			var song Song
			err := rows.Scan(&song.Id, &song.Path, &song.Title, &song.added, &song.Artist, &song.Album)
			if err != nil {
				w.WriteHeader(500)
				return
			}
			song.EpochMillis = time.Unix(int64(song.added), 0).UnixMilli()
			index := strings.Index(song.Path, "/complete/music")
			song.Path = song.Path[index:len(song.Path)]
			items = append(items, song)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		encoder := json.NewEncoder(w)
		encoder.Encode(items)
	}
}
