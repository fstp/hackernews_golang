package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

/*
   url = "https://hacker-news.firebaseio.com/v0/item/" <> to_string(id) <> ".json"
   url = "https://hacker-news.firebaseio.com/v0/topstories.json"
*/

var storyIds chan int
var stories chan string
var wg sync.WaitGroup

// Story doc
type Story struct {
	Title *string `json:"title,omitempty"`
	URL   *string `json:"url,omitempty"`
}

func worker() {
	for {
		select {
		case id := <-storyIds:
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
		default:
			wg.Done()
			return
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	rsp, err := http.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var tmp []int
	err = json.Unmarshal(body, &tmp)
	if err != nil {
		log.Fatal(err)
	}
	/*for story, idx := range storyIds {
		fmt.Printf("%v - %v\n", story, idx)
	}*/

	wg.Add(len(tmp))
	storyIds = make(chan int, len(tmp))
	stories = make(chan string, len(tmp))

	for id := range tmp {
		storyIds <- id
		go worker()
	}

	wg.Wait()

	close(storyIds)
	close(stories)

	for story := range stories {
		fmt.Println(story)
	}
}
