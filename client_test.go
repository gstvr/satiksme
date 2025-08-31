package satiksme

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

const mockResponse = `stop,3100
tram,5,b-a,49267,31836,Iļģuciems
tram,1,b-a,49350,58545Z,Imanta
stop,1025a
bus,22,a-b,49878,78661,Lidosta
trol,27,a-b,50100,29179,Ziepniekkalns
bus,23,a-b,50100,71301,Baloži`

func TestClient_GetStopDepartures(t *testing.T) {
	tz, err := time.LoadLocation(timezoneRiga)
	requireNoError(t, err)

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, tz)
	stopIDs := []string{"3100", "1025a"}

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireEquals(t, "saraksti.lv", r.Header.Get("Origin-Custom"))
		requireEquals(t, "/gpsdata.ashx", r.URL.Path)
		requireStringContains(t, r.URL.RawQuery, "stopid=3100,1025a")
		requireStringContains(t, r.URL.RawQuery, fmt.Sprintf("time=%d", now.UnixMilli()))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer svr.Close()

	c, err := NewClient()
	requireNoError(t, err)

	c.baseURL = svr.URL
	c.now = func() time.Time { return now }

	actual, err := c.GetStopDepartures(context.Background(), stopIDs)
	requireNoError(t, err)

	fmtTime := func(hhmmss string) time.Time {
		t, _ := time.Parse(time.TimeOnly, hhmmss)
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location())
	}

	expected := []StopDepartures{
		{
			StopID: "3100",
			Departures: []Departure{
				{
					Line:        Line{Number: "5", VehicleType: VehicleTypeTram},
					Destination: "Iļģuciems",
					DepartsAt:   fmtTime("13:41:07"),
					VehicleID:   "31836",
				},
				{
					Line:        Line{Number: "1", VehicleType: VehicleTypeTram},
					Destination: "Imanta",
					DepartsAt:   fmtTime("13:42:30"),
					VehicleID:   "58545",
					Flags:       []Flag{FlagAccessibleTram},
				},
			},
		},
		{
			StopID: "1025a",
			Departures: []Departure{
				{
					Line:        Line{Number: "22", VehicleType: VehicleTypeBus},
					Destination: "Lidosta",
					DepartsAt:   fmtTime("13:51:18"),
					VehicleID:   "78661",
				},
				{
					Line:        Line{Number: "27", VehicleType: VehicleTypeTrolleybus},
					Destination: "Ziepniekkalns",
					DepartsAt:   fmtTime("13:55:00"),
					VehicleID:   "29179",
				},
				{
					Line:        Line{Number: "23", VehicleType: VehicleTypeBus},
					Destination: "Baloži",
					DepartsAt:   fmtTime("13:55:00"),
					VehicleID:   "71301",
					Flags:       []Flag{FlagElectricBus},
				},
			},
		},
	}

	requireEquals(t, expected, actual)
}

func requireEquals(t *testing.T, expected, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %+v, got %+v", expected, actual)
		t.FailNow()
	}
}

func requireStringContains(t *testing.T, str, substr string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Errorf("expected %+v to contain %+v", str, substr)
		t.FailNow()
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %+v", err)
		t.FailNow()
	}
}
