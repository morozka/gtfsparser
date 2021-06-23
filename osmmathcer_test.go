package gtfsparser

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/LdDl/osm2ch"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
)

func TestGetGTFSBoundingBox(t *testing.T) {

	gtfsMap := make(map[string]struct{})
	gtfsRoutes := make(map[string]struct{})

	feed, err := GetGTFSFeed("example_data/gtfs.zip")
	if err != nil {
		log.Fatal(err)
	}
	for _, stop := range feed.Stops {
		gtfsMap[fmt.Sprintf("%6.3f %6.3f", stop.Lon, stop.Lat)] = struct{}{}
	}

	// Open the PBF file
	f, err := os.Open("example_data/spb.osm.pbf")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	action := osm.Action{OSM: &osm.OSM{}}
	osmReader := osmpbf.New(context.Background(), f, 1)
	counter := 0
	routes := 0
	matched := 0
	altmatched := 0
	for osmReader.Scan() {
		obj := osmReader.Object()
		action.Append(obj)
		if obj.ObjectID().Type() == "node" {
			tags := obj.(*osm.Node).TagMap()
			if _, stopPos := tags["public_transport"]; stopPos {
				if _, ok := gtfsMap[fmt.Sprintf("%6.3f %6.3f", obj.(*osm.Node).Lon, obj.(*osm.Node).Lat)]; ok {

					//fmt.Println(tags)
				}
			}
		}
		if obj.ObjectID().Type() == "relation" {
			rel := obj.(*osm.Relation)
			tags := rel.TagMap()

			if _, ok := tags["route"]; ok /*&& rtype == "tram"*/ {
				routes++
				buf := strings.Split(tags["name"], "№ ")
				if len(buf) < 2 {
					continue
				}
				buf = strings.Split(buf[1], ":")
				number := buf[0]
				if _, ok := gtfsRoutes[tags[number]]; ok {
					altmatched++
					continue
				}
				if _, ok := gtfsRoutes[tags["ref"]]; ok {
					matched++
					//fmt.Println(tags)
					//continue
				}

			}
		}
	}
	fmt.Printf("Counted %d / %d \n", counter, len(feed.Stops))
	fmt.Printf("Counted all(%d) %d(%d)/%d \n", routes, matched, altmatched, len(gtfsRoutes))
	data, _ := action.MarshalJSON()
	ioutil.WriteFile("output.json", data, 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done, parsed %d agencies, %d stops, %d routes, %d trips, %d fare attributes, %v\n\n",
		len(feed.Agencies), len(feed.Stops), len(feed.Routes), len(feed.Trips), len(feed.FareAttributes), len(feed.Shapes))
	fmt.Println("Error: ", err)

	fmt.Println(feed.GetGTFSBoundingBox())

}

func TestGraphPreparation(t *testing.T) {
	tagStr := "motorway,primary,primary_link,road,secondary,secondary_link,residential,tertiary,tertiary_link,unclassified,trunk,trunk_link"
	tags := strings.Split(tagStr, ",")
	cfg := osm2ch.OsmConfiguration{
		EntityName: "highway",
		Tags:       tags,
	}
	graph, err := osm2ch.ImportFromOSMFile("example_data/spb.osm.pbf", &cfg)
	fmt.Println(err)
	_ = graph

}
func TestMatchFeedStops(t *testing.T) {
	mapper, err := LoadOsmMapper("example_data/spb.osm.pbf")
	if err != nil {
		log.Println(err)
	}
	feed, err := GetGTFSFeed("example_data/gtfs.zip")
	if err != nil {
		log.Println(err)
	}
	mathced := feed.MatchFeedStops(mapper)
	log.Println(len(mathced), len(feed.Stops))
	//log.Println(mathced)

}
