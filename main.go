package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var dbpath string
var basePath string

func init() {
	flag.StringVar(&dbpath, "dbpath", "/downloads/complete/complete/music/musiclibrary.db", "Path to the sqlite database file")
	flag.StringVar(&basePath, "basepath", "/data", "path to storage")
}

func main() {
	flag.Parse()
	db, err := sql.Open("sqlite", dbpath)
	if err != nil {
		log.Fatal(err)
	}
	items, err := ListItems(db)
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range items {
		path := filepath.Join("/data", item.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Default().Println(item.Path)
		}
	}
	http.Handle("/list", listEndpoint(db))
	http.Handle("/list/videos", videosEndpoint())
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

func videosEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := ListVideos(basePath)
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

type Video struct {
	Name      string
	Season    int
	Episode   int
	Link      string
	Subtitles string
}

func ListVideos(basePath string) ([]*Video, error) {
	result := []*Video{}
	filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		for _, suffix := range []string{".mp4", ".mkv", ".avi"} {
			if strings.HasSuffix(path, suffix) {
				result = append(result, &Video{Name: path})
			}
		}
		return nil
	})
	for _, vid := range result {
		FillMetadata(vid, basePath)
	}
	return result, nil
}

var seasonRegex = regexp.MustCompile(`(.+)(\W+(\d\d\d\d))?\W+[Ss](\d\d+)[Ee](\d\d+)`)
var replaces = regexp.MustCompile(`\W+`)

func FillMetadata(video *Video, basepath string) error {
	video.Link = video.Name[len(basepath):]
	paths := strings.Split(video.Name, "/")
	foundMatch := false
	for i := range paths {
		matches := seasonRegex.FindStringSubmatch(paths[len(paths)-1-i])
		if matches != nil {
			foundMatch = true
			season, err := strconv.Atoi(matches[4])
			if err != nil {
				return err
			}
			episode, err := strconv.Atoi(matches[5])
			if err != nil {
				return err
			}
			video.Season = season
			video.Episode = episode
			video.Name = replaces.ReplaceAllString(strings.ToLower(matches[1]), " ")
			break
		}
	}
	if !foundMatch {
		video.Name = paths[len(paths)-1]
	}
	return nil
}
