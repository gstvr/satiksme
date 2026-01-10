package satiksme

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type VehicleType string

const (
	VehicleTypeBus        VehicleType = "bus"
	VehicleTypeTram       VehicleType = "tram"
	VehicleTypeTrolleybus VehicleType = "trol"
)

type Line struct {
	Number      string
	VehicleType VehicleType
}

func (l Line) Name() string {
	return string(l.VehicleType) + "-" + l.Number
}

type Flag string

const (
	FlagAccessibleTram Flag = "accessible-tram"
	FlagElectricBus    Flag = "electric-bus"
)

type Departure struct {
	Line        Line
	Destination string
	DepartsAt   time.Time
	VehicleID   string
	Flags       []Flag
}

func (d Departure) IsAccessibleTram() bool {
	return slices.Contains(d.Flags, FlagAccessibleTram)
}

func (d Departure) IsElectricBus() bool {
	return slices.Contains(d.Flags, FlagElectricBus)
}

func (d Departure) RelativeDeparture(relativeTo time.Time) string {
	departsIn := int(d.DepartsAt.Sub(relativeTo).Minutes())
	return fmt.Sprintf("%d min", departsIn)
}

type StopDepartures struct {
	StopID     string
	Departures []Departure
}

const timezoneRiga = "Europe/Riga"

type Client struct {
	baseURL    string
	httpClient *http.Client
	now        func() time.Time
}

func NewClient() (Client, error) {
	rigaTZ, err := time.LoadLocation(timezoneRiga)
	if err != nil {
		return Client{}, fmt.Errorf("could not load %s timezone", timezoneRiga)
	}

	return Client{
		baseURL:    "https://saraksti.lv",
		httpClient: &http.Client{},
		now: func() time.Time {
			return time.Now().In(rigaTZ)
		},
	}, nil
}

func (c Client) GetStopDepartures(ctx context.Context, stopIDs []string) ([]StopDepartures, error) {
	now := c.now()

	url := fmt.Sprintf("%s/gpsdata.ashx?stopid=%s&time=%d", c.baseURL, strings.Join(stopIDs, ","), now.UnixMilli())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Requests without this header are rejected by the server.
	req.Header.Set("Origin-Custom", "saraksti.lv")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed performing request: %w", err)
	}
	defer res.Body.Close()

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	stops := make([]StopDepartures, 0, len(stopIDs))

	s := bufio.NewScanner(res.Body)
	for s.Scan() {
		line := s.Text()
		cols := strings.Split(line, ",")

		if strings.HasPrefix(line, "stop") {
			stops = append(stops, StopDepartures{
				StopID: cols[1],
			})
			continue
		}

		if len(cols) != 6 {
			// Each departure line is expected to contain exactly 6 columns.
			// If it doesn't, the response is likely malformed.
			// TODO: handle the line gracefully instead of skipping silently.
			continue
		}

		colVehicleType := VehicleType(cols[0])
		colLineNumber := cols[1]
		colDepartureTime := cols[3]
		colVehicleID := cols[4]
		colDestination := cols[5]

		departsAt, err := c.parseDepartureTime(colDepartureTime, startOfDay)
		if err != nil {
			return nil, fmt.Errorf("parsing departure time: %w", err)
		}

		vehicleID, isAccessibleTram := c.parseVehicleID(colVehicleID)

		var flags []Flag
		if isAccessibleTram {
			flags = append(flags, FlagAccessibleTram)
		}
		if c.isElectricBus(vehicleID) {
			flags = append(flags, FlagElectricBus)
		}

		lastIdx := len(stops) - 1
		stops[lastIdx].Departures = append(stops[lastIdx].Departures, Departure{
			Line: Line{
				Number:      colLineNumber,
				VehicleType: colVehicleType,
			},
			DepartsAt:   departsAt,
			Destination: colDestination,
			VehicleID:   vehicleID,
			Flags:       flags,
		})
	}

	return stops, nil
}

// The feed returns departure times as the number of seconds since the start of the day.
func (c Client) parseDepartureTime(col string, startOfDay time.Time) (time.Time, error) {
	departureTimeInSeconds, err := strconv.Atoi(col)
	if err != nil {
		return time.Time{}, err
	}

	// Sometimes the departure time is after midnight, but is considered as a departure on the previous day.
	// The feed returns those relative to that day. So we need adjust it relative to the current day.
	const secondsInDay = 86400
	if departureTimeInSeconds > secondsInDay {
		departureTimeInSeconds -= secondsInDay
	}

	return startOfDay.Add(time.Duration(departureTimeInSeconds) * time.Second), nil
}

// The feed suffixes the vehicle ID for accessible trams with "Z". The character is not part of the actual
// vehicle ID and needs to be stripped to make sure the vehicle ID can be correctly used for other purposes.
func (c Client) parseVehicleID(col string) (id string, isAccessibleTram bool) {
	if strings.HasSuffix(col, "Z") {
		return col[:len(col)-1], true
	}
	return col, false
}

// All electric buses in Rigas Satiksme's fleet have a vehicle ID that starts with 71.
func (c Client) isElectricBus(vehicleID string) bool {
	return strings.HasPrefix(vehicleID, "71")
}
