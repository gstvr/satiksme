# satiksme

A Go library that provides an easy way to retrieve realtime public transport departure times for Riga public transport
authority's (Rīgas Satiksme) stops. The library uses the API consumed by [Rīgas Satiksme's frontend](https://saraksti.lv) 
and parses the data into easy-to-consume data structure. 

### Usage

```go
package main

import (
	// ...
	"github.com/gstvr/satiksme"
)

func main() {
	c, err := satiksme.NewClient()
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	stopDepartures, err := c.GetStopDepartures(context.Background(), []string{"3100"})
	if err != nil {
		log.Fatalf("failed to get departures: %v", err)
	}

	for _, stop := range stopDepartures {
		fmt.Printf("Stop ID: %s\n", stop.StopID)

		for _, dep := range stop.Departures {
			fmt.Printf("  Line: %s, Destination: %s, Departs At: %s, Vehicle ID: %s, Flags: %v\n",
				dep.Line.Name(), dep.Destination, dep.DepartsAt.Format(time.TimeOnly), dep.VehicleID, dep.Flags)
		}
	}
}
```
