package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var basePath string

func init() {
	flag.StringVar(&basePath, "basepath", "/data", "path to storage")
}

func main() {
	flag.Parse()
	_, err := ListSongs(filepath.Join(basePath, "music"), basePath)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/list", listEndpoint(filepath.Join(basePath, "music"), basePath))
	http.Handle("/", http.FileServer(http.Dir(basePath)))
	if err = http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

type Song struct {
	Id          int
	Path        string
	Title       string
	EpochMillis int64
	Artist      string
	Album       string
}

func listEndpoint(basePath, relativeTo string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := ListSongs(basePath, relativeTo)
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

type AlbumInfo struct {
	XMLName   xml.Name `xml:"album"`
	Artist    string   `xml:"artist"`
	Title     string   `xml:"title"`
	Track     []Track  `xml:"track"`
	Premiered string   `xml:"premiered"`
}

type Track struct {
	XMLName  xml.Name `xml:"track"`
	Position int      `xml:"position"`
	Title    string   `xml:"title"`
}

func ListSongs(path, relativeTo string) ([]*Song, error) {
	result := []*Song{}
	idGenerator := 0
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		//for each album.nfo file, add all files in the dir or subdirs to a list, skip processing rest of dir
		if !d.IsDir() {
			return nil
		}
		albumInfo, err := os.Open(filepath.Join(path, "album.nfo"))
		if os.IsNotExist(err) {
			return nil
		}
		defer albumInfo.Close()
		data, err := io.ReadAll(albumInfo)
		if err != nil {
			return err
		}
		parsedInfo := &AlbumInfo{}
		if err = xml.Unmarshal(data, parsedInfo); err != nil {
			return err
		}
		releaseEpochMillis := time.Now()
		if parsedInfo.Premiered != "" {
			releaseEpochMillis, err = time.Parse("2006-01-02", parsedInfo.Premiered)
			if err != nil {
				return err
			}
		}

		allSongs := []string{}
		for _, track := range parsedInfo.Track {
			allSongs = append(allSongs, track.Title)
		}
		if err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			for _, title := range allSongs {
				if strings.Contains(d.Name(), title) {
					httpPath, err := filepath.Rel(relativeTo, path)
					if err != nil {
						return err
					}
					result = append(result, &Song{
						Id:          idGenerator,
						Path:        httpPath,
						Title:       title,
						Album:       parsedInfo.Title,
						Artist:      parsedInfo.Artist,
						EpochMillis: releaseEpochMillis.UnixMilli(),
					})
					idGenerator += 1
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return filepath.SkipDir
	})
	return result, err
}
