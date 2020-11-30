package main

import (
	"fmt"
	"encoding/json"
	"net/http"
	"log"
	"strconv"
	elastic "gopkg.in/olivere/elastic.v3"
 	"github.com/pborman/uuid"
	"reflect"
)

type Location struct{
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Post struct{
	User string `json:"user"`
	Message string `json:"message"`
	Location Location `json:"location"`
}

const (
	INDEX = "around"
	TYPE = "post"
	DISTANCE = "200km"
	ES_URL = "http://35.192.180.86:9200"
)

func main() {
	// fmt.Println("Started Service")
	// http.HandleFunc("/post", handlerPost)
	// http.HandleFunc("/search", handlerSearch)
	// log.Fatal(http.ListenAndServe(":8080", nil))


	// Create a client
	client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(INDEX).Do()
	if err != nil {
		panic(err)
	}
	if !exists {
		// Create a new index.
		mapping := `{
			"mappings":{
				"post":{
					"properties":{
						"location":{
							"type":"geo_point"
						}
					}
				}
			}
		}`
		_, err := client.CreateIndex(INDEX).Body(mapping).Do()
		if err != nil {
			// Handle error
			panic(err)
		}
	}
	fmt.Println("started-service")
	http.HandleFunc("/post", handlerPost)
	http.HandleFunc("/search", handlerSearch)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handlerPost(w http.ResponseWriter, r *http.Request){
	fmt.Println("Receiving one post request")
	decoder:= json.NewDecoder(r.Body)
	var p Post
	if err:= decoder.Decode(&p); err !=nil{
		panic(err)
		return
	}
	fmt.Fprintf(w, "Post Received: %s\n", p.Message)
	id := uuid.New()
	saveToES(&p, id)

}

func saveToES(p *Post, id string){
	es_client , err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err!= nil{
		panic(err)
	}
	
	_, e := es_client.Index().
		Index(INDEX).
		Type(TYPE).
		Id(id).
		BodyJson(p).
		Refresh(true).
		Do()
	if e!=nil {
		panic(e)
	}

	fmt.Printf("Post is saved to index %s \n", p.Message)

}

func handlerSearch(w http.ResponseWriter, r *http.Request){
	fmt.Println("Received one request from search")
	lat , _ := strconv.ParseFloat(r.URL.Query().Get("lat"),64)
	lon , _ := strconv.ParseFloat(r.URL.Query().Get("lon"),64)
	ran := DISTANCE
	if val := r.URL.Query().Get("range"); val!=""{
		ran = val + "km";
	}
	client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if  err!= nil{
		panic(err)
	}
	q:= elastic.NewGeoDistanceQuery("location")
	q= q.Distance(ran).Lat(lat).Lon(lon)
	searchResult, err := client.Search().Index(INDEX).Query(q).Pretty(true).Do()

	if err!=nil{
		panic(err)
	}

	fmt.Println("Query took %d milliseconds\n", searchResult.TookInMillis)
	fmt.Printf("Found a total of %d posts \n" , searchResult.TotalHits())

	var typ Post
	var ps []Post

	for _, item := range searchResult.Each(reflect.TypeOf(typ)){
		p := item.(Post)
		fmt.Printf("Post by %s: %s at lat %v and lon %v \n", 
		p.User, p.Message, p.Location.Lat, p.Location.Lon)
		ps = append(ps, p)
	}

	js, err := json.Marshal(ps)
	if err != nil{
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow_Origin", "*")
	w.Write(js)

}
