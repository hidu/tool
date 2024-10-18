//  Copyright(C) 2024 github.com/hidu  All Rights Reserved.
//  Author: hidu <duv123+git@gmail.com>
//  Date: 2024-10-18

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/wtolson/go-taglib"
	"strconv"
)

var view = flag.Bool("v", false, "view tags")
var set = flag.String("s", "", `set new tag values
format: tagName=tagValue [; tagName=tagValue ]
tagName = [ Title, Album, Artist, Comment, Genre, Track, Year ]
eg:
  1. Title=花妖
  2. Title=花妖 ; Track=1 ; Year=2000
  3. Title={Album} {Title}
  4. Title={fileName}
`)

var trackInc = flag.Int("track", 0, "set Track value autoincrement, value >=1 is enable")

func main() {
	flag.Parse()
	files := fileNames()
	if len(files) == 0 {
		log.Fatalln("no files found")
	}
	traceID = *trackInc

	for _, file := range files {
		if *view {
			doView(file)
		} else if *set != "" {
			values := parserSet(*set)
			if len(values) == 0 {
				log.Fatalln("no valid set values")
			}
			doSet(file, values)
		} else if *trackInc > 0 {
			doSetTrackInc(file)
		}
	}
}

func fileNames() []string {
	args := flag.Args()
	files := make([]string, 0, len(args))
	for _, name := range args {
		if !strings.Contains(name, "*") {
			files = append(files, name)
		} else {
			fs, _ := filepath.Glob(name)
			files = append(files, fs...)
		}
	}
	return files
}

func doView(name string) {
	f, err := taglib.Read(name)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	info := map[string]any{
		"Title":   f.Title(),
		"Album":   f.Album(),
		"Artist":  f.Artist(),
		"Comment": f.Comment(),
		"Year":    f.Year(),
		"Genre":   f.Genre(),
		"Track":   f.Track(),
	}
	bf, _ := json.Marshal(info)
	fmt.Println(name, "\t", string(bf))
}

func parserSet(str string) map[string]string {
	arr := strings.Split(str, ";")
	result := make(map[string]string, len(arr))
	for _, v := range arr {
		v = strings.TrimSpace(v)
		k, v, ok := strings.Cut(v, "=")
		if ok {
			k := strings.TrimSpace(k)
			result[k] = strings.TrimSpace(v)
		}
	}
	return result
}

func doSet(name string, values map[string]string) {
	f, err := taglib.Read(name)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	for k, v := range values {
		switch k {
		case "Title":
			f.SetTitle(formatTag(v, name, f))
		case "Album":
			f.SetAlbum(formatTag(v, name, f))
		case "Artist":
			f.SetArtist(formatTag(v, name, f))
		case "Comment":
			f.SetComment(formatTag(v, name, f))
		case "Genre":
			f.SetGenre(formatTag(v, name, f))
		case "Track":
			f.SetTag(taglib.Track, v)
		case "Tear":
			f.SetTag(taglib.Year, v)
		}
	}
	if err := f.Save(); err != nil {
		log.Printf("[Error] file=%q error=%v\n", name, err)
	} else {
		log.Printf("[Info] file=%q save ok\n", name)
	}
}

func formatTag(value string, filename string, f *taglib.File) string {
	value = strings.ReplaceAll(value, "{Title}", f.Title())
	value = strings.ReplaceAll(value, "{Album}", f.Album())
	value = strings.ReplaceAll(value, "{Artist}", f.Artist())
	value = strings.ReplaceAll(value, "{Comment}", f.Comment())
	value = strings.ReplaceAll(value, "{Year}", strconv.Itoa(f.Year()))
	value = strings.ReplaceAll(value, "{Track}", strconv.Itoa(f.Track()))

	name := filepath.Base(filename)
	name, _, _ = strings.Cut(name, ".")
	name = strings.TrimSpace(name)
	value = strings.ReplaceAll(value, "{fileName}", name)

	value = strings.TrimSpace(value)
	return value
}

var traceID int

func doSetTrackInc(name string) {
	f, err := taglib.Read(name)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	f.SetTrack(traceID)
	if err := f.Save(); err != nil {
		log.Printf("[Error] file=%q error=%v\n", name, err)
	} else {
		log.Printf("[Info] file=%q save ok\n", name)
	}
	traceID++
}
