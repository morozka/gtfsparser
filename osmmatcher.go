package gtfsparser

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/morozka/gtfsparser/gtfs"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
)

// GeoPoint Representation of point on Earth
type GeoPoint struct {
	Lon float32
	Lat float32
}
type osmNode struct {
	osmID int64
	Lon   float64
	Lat   float64
}

//GetGTFSBoundingBox - return min and max points for rectangle of stops in GTFS Feed
func (f *Feed) GetGTFSBoundingBox() (minLat, minLon, maxLat, maxLon float32, err error) {
	if len(f.Stops) == 0 {
		return minLat, maxLat, minLon, maxLon, fmt.Errorf("no stops in GTFS Feed")
	}
	first := true
	for _, v := range f.Stops {
		if first {
			minLat = v.Lat
			maxLat = v.Lat
			minLon = v.Lon
			maxLon = v.Lon
			first = false
			continue
		}
		if v.Lat < minLat {
			minLat = v.Lat
		}
		if v.Lat > maxLat {
			maxLat = v.Lat
		}
		if v.Lon < minLon {
			minLon = v.Lon
		}
		if v.Lon > maxLon {
			maxLon = v.Lon
		}
	}
	return minLat - 0.004, minLon - 0.004, maxLat + 0.004, maxLon + 0.004, nil // ~400 meter delta, more info in GO-SYNC Compare-data
}

func LoadOsmMapper(file string) (map[string][]osmNode, error) {
	osmMapper := make(map[string][]osmNode)
	f, err := os.Open(file)
	if err != nil {
		return osmMapper, err
	}
	defer f.Close()

	action := osm.Action{OSM: &osm.OSM{}}
	osmReader := osmpbf.New(context.Background(), f, 1)
	for osmReader.Scan() {
		obj := osmReader.Object()

		if obj.ObjectID().Type() == "node" {
			action.Append(obj)
			tags := obj.(*osm.Node).TagMap()
			if _, stopPos := tags["public_transport"]; stopPos {
				key := fmt.Sprintf("%6.3f %6.3f", obj.(*osm.Node).Lon, obj.(*osm.Node).Lat)
				if _, ok := osmMapper[key]; ok {
					osmMapper[key] = append(osmMapper[key], osmNode{int64(obj.(*osm.Node).ID), obj.(*osm.Node).Lon, obj.(*osm.Node).Lat})
				} else {
					osmMapper[key] = make([]osmNode, 0, 1)
					osmMapper[key] = append(osmMapper[key], osmNode{int64(obj.(*osm.Node).ID), obj.(*osm.Node).Lon, obj.(*osm.Node).Lat})
				}
			}
		}
		if obj.ObjectID().Type() == "way" {
			action.Append(obj)
		}
	}
	return osmMapper, nil
}

func GetGTFSFeed(file string) (*Feed, error) {
	feed := NewFeed()
	err := feed.Parse(file)
	if err != nil {
		return feed, err
	}
	return feed, nil
}

func (feed Feed) MatchFeedStops(osmMapper map[string][]osmNode) map[gtfs.Stop]osmNode {
	matchedStops := make(map[gtfs.Stop]osmNode)
	met20 := 0
	met10 := 0
	met5 := 0
	met2 := 0
	for _, stop := range feed.Stops {
		var minDist float32 = math.MaxFloat32

		var minNode osmNode
		for i := -1; i < 2; i++ {
			for j := -1; j < 2; j++ {
				key := fmt.Sprintf("%6.3f %6.3f", stop.Lon+float32(i)*0.001, stop.Lat+float32(j)*0.001)
				possibleNodes, ok := osmMapper[key]
				if ok {
					for _, node := range possibleNodes {
						//dist := getDistSQR(*stop, node)
						dist := float32(Distance(float64(stop.Lat), float64(stop.Lon), node.Lat, node.Lon))
						if dist < minDist {
							minNode = node
							minDist = dist
						}
					}
				}
			}
		}
		if minDist < math.MaxFloat32 {
			if minDist < 2 {
				matchedStops[*stop] = minNode
			} else {
				switch {
				case minDist < 2.0:
					met2++
					fallthrough
				case minDist < 5.0:
					met5++
					fallthrough
				case minDist < 10.0:
					met10++
					fallthrough

				case minDist < 20.0:
					met20++
				}
			}

		}
	}
	fmt.Println("Number of matched stops in 2m,5m,10m,20m:")
	fmt.Println(met2, met5, met10, met20)
	return matchedStops
}

func getDistSQR(stop gtfs.Stop, node osmNode) float32 {
	sqrLon := math.Pow((float64(stop.Lon) - node.Lon), 2)
	sqrLat := math.Pow((float64(stop.Lat) - node.Lat), 2)
	return float32(math.Sqrt(sqrLon + sqrLat))
}
func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

func Distance(lat1, lon1, lat2, lon2 float64) float64 {

	var la1, lo1, la2, lo2, r float64
	la1 = lat1 * math.Pi / 180
	lo1 = lon1 * math.Pi / 180
	la2 = lat2 * math.Pi / 180
	lo2 = lon2 * math.Pi / 180

	r = 6378100 // Earth radius in METERS

	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * r * math.Asin(math.Sqrt(h))
}
