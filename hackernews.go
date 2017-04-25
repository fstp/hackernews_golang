package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var stories chan string
var wg sync.WaitGroup

// Story doc
type Story struct {
	Title *string `json:"title,omitempty"`
	URL   *string `json:"url,omitempty"`
}

func worker(id int) {
	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	rsp, err := http.Get(url)
	if err != nil {
		log.Fatalf("%s: %v", url, err)
	}
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		log.Fatalf("%s: %v", url, err)
	}
	var story Story
	err = json.Unmarshal(body, &story)
	if err != nil {
		log.Fatalf("%s: %v", url, err)
	}
	if story.Title != nil && story.URL != nil {
		stories <- fmt.Sprintf("%s\n%s\n\n", *story.Title, *story.URL)
	}
	rsp.Body.Close()
	wg.Done()
}

type intArray []uint64

func (a intArray) Len() int           { return len(a) }
func (a intArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a intArray) Less(i, j int) bool { return a[i] < a[j] }

func needUpdate(storyIds intArray) bool {
	var b bytes.Buffer
	err := binary.Write(&b, binary.LittleEndian, storyIds)
	if err != nil {
		// Very unlikely that converting integer to byte array
		// should fail, but we should check for it anyway.
		log.Fatal(err)
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	checksumFile := filepath.Join(usr.HomeDir, "topstories.sha256")
	newChecksum := sha256.Sum256(b.Bytes())
	oldChecksum, err := ioutil.ReadFile(checksumFile)

	if err != nil {
		// If opening the file failed or if the file does not exist
		// at all we just replace it with the new SHA256 and move on.
		err := ioutil.WriteFile(checksumFile, newChecksum[:], 0644)
		if err != nil {
			log.Fatal(err)
		}
		return true
	}

	if !bytes.Equal(newChecksum[:], oldChecksum) {
		err := ioutil.WriteFile(checksumFile, newChecksum[:], 0644)
		if err != nil {
			log.Fatal(err)
		}
		return true
	}

	return false
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	start := time.Now()

	rsp, err := http.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var storyIds intArray
	err = json.Unmarshal(body, &storyIds)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(storyIds)

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	storiesTextfile := filepath.Join(usr.HomeDir, "topstories.txt")

	if !needUpdate(storyIds) {
		stories, err := ioutil.ReadFile(storiesTextfile)
		if err == nil {
			// Stories already present in cache that is up to date,
			// nothing more to do than print it for the user.
			fmt.Printf("%s", stories)

			elapsed := time.Since(start)
			fmt.Printf("Cache up to date, Time elapsed: %s", elapsed)
			return
		}
		// Unable to read the textfile or it has been removed...
		// continue since it will be replaced with new content anyway.
	}

	stories = make(chan string, len(storyIds))

	wg.Add(len(storyIds))
	for id := range storyIds {
		go worker(id)
	}
	wg.Wait()

	fd, err := os.OpenFile(storiesTextfile, os.O_WRONLY, 0222)
	if err != nil {
		log.Printf("failed to save stories to cache: %v", err)
	}
	w := bufio.NewWriter(fd)

	close(stories)
	for story := range stories {
		fmt.Println(story)
		w.Write([]byte(story))
	}

	w.Flush()
	fd.Close()

	elapsed := time.Since(start)
	fmt.Printf("Downloaded stories from HackerNews, Time elapsed: %s", elapsed)
}
