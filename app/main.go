package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func getAsteroidsToday() map[string]interface{} {
	var data map[string]interface{}
	nasa := "https://api.nasa.gov/neo/rest/v1/feed/today?detailed=true&api_key=%s"
	key := os.Getenv("NASA_API_KEY")
	uri := fmt.Sprintf(nasa, key)

	resp, err := http.Get(uri)
	if err != nil {
		log.Fatalf("Couldn't access NASA API: %s", err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Couldn't read response data: %s", err.Error())
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Fatalf("Couldn't jsonify data: %s", err.Error())
	}
	return data
}

func main() {
	obj := getAsteroidsToday()
	fmt.Println(obj)
}