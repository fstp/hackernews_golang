package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

	var storyIds []int
	err = json.Unmarshal(body, &storyIds)
	if err != nil {
		log.Fatal(err)
	}
	/*for story, idx := range storyIds {
		fmt.Printf("%v - %v\n", story, idx)
	}*/

	stories = make(chan string, len(storyIds))

	wg.Add(len(storyIds))
	for id := range storyIds {
		go worker(id)
	}
	wg.Wait()

	close(stories)
	for story := range stories {
		fmt.Println(story)
	}

	elapsed := time.Since(start)
	fmt.Printf("Time elapsed: %s", elapsed)
}
