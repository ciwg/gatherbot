package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/stevegt/envi"
	"github.com/stevegt/gatherbot/eventbrite"
	. "github.com/stevegt/goadapt"
)

const csvdir = "/tmp/gatherbot.csv"
const jsondir = "/tmp/gatherbot.json"

type TicketType string

type Conf struct {
	Days map[TicketType]ConfDay
}

type ConfDay struct {
	SpaceId   string
	Overwrite bool
}

type EventBrite struct {
	OrderID        string `csv:"Order ID"`
	OrderDate      string `csv:"Order Date"`
	AttendeeStatus string `csv:"Attendee Status"`
	Name           string `csv:"Name"`
	Email          string `csv:"Email"`
	EventName      string `csv:"Event Name"`
	TicketQuantity string `csv:"Ticket Quantity"`
	TicketType     string `csv:"Ticket Type"`
	TicketPrice    string `csv:"Ticket Price"`
	BuyerName      string `csv:"Buyer Name"`
	BuyerEmail     string `csv:"Buyer Email"`
}

type Gather struct {
	Email       string `csv:"email"`
	Name        string `csv:"name"`
	Role        string `csv:"role"`
	Affiliation string `csv:"affiliation"`
}

type GatherGuestDetail struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Affiliation string `json:"affiliation"`
}

// map[email]GatherGuestDetail
// type GatherGuestList map[string]GatherGuestDetail

type GatherJSON struct {
	ApiKey    string                       `json:"apiKey"`
	SpaceId   string                       `json:"spaceId"`
	Overwrite bool                         `json:"overwrite"`
	Guestlist map[string]GatherGuestDetail `json:"guestlist"`
}

func main() {
	src := os.Args[1]
	dst := os.Args[2]

	var err error
	var evs []EventBrite

	switch src {
	case "api":
		evs, err = getEvs()
		Ck(err)
	default:
		evs = readCsv(src)
	}

	// Pprint(evs)

	days := evs2days(evs)
	err = verify(days)
	Ck(err)

	switch dst {
	case "csv":
		err := writeCSVs(days)
		Ck(err)
	case "json":
		err := writeJSON(days)
		Ck(err)
	default:
		Assert(false)
	}
}

func getEvs() (evs []EventBrite, err error) {
	attendees, err := eventbrite.GetAttendeesFromEnv()
	Ck(err)
	// Pprint(attendees)
	// os.Exit(1)
	for _, a := range attendees {
		ev := EventBrite{
			Name:       a.Name,
			Email:      a.Email,
			TicketType: a.TicketType,
		}
		evs = append(evs, ev)
	}
	return
}

func readCsv(infn string) (evs []EventBrite) {

	infh, err := os.Open(infn)
	defer infh.Close()

	err = gocsv.UnmarshalFile(infh, &evs)
	Ck(err)

	return
}

// convert from eventbrite to gather
func evs2days(evs []EventBrite) (days map[string][]Gather) {
	days = make(map[string][]Gather)
	var allevent []Gather
	for _, ev := range evs {
		g := Gather{
			Email: ev.Email,
			Name:  ev.Name,
			Role:  ev.TicketType,
			// Affiliation:
		}
		if ev.TicketType == "All-NOMCON Event Ticket" {
			allevent = append(allevent, g)
		} else {
			days[ev.TicketType] = append(days[ev.TicketType], g)
		}
	}
	err := verify(days)
	Ck(err)

	// Pprint(days["June 21 - Making a Change in Health & Science"])
	// Pprint(allevent)

	Assert(len(days) == 6, len(days))

	// add all-event people to each day
	for tt, gs := range days {
		Pf("%3d %3d %3d %s\n", len(gs), len(allevent), len(gs)+len(allevent), tt)
		for _, g := range allevent {
			days[tt] = append(days[tt], g)
		}
	}

	/*
		for tt, _ := range days {
			Pf("\"%s\"\n", tt)
		}
	*/

	err = verify(days)
	Ck(err)
	return
}

func verify(days map[string][]Gather) (err error) {
	defer Return(&err)
	for tt, gs := range days {
		// Pl(tt)
		for _, g := range gs {
			if g.Role == "All-NOMCON Event Ticket" {
				continue
			}
			Assert(tt == g.Role, "mismatched ticket/role: '%v' %v", tt, g)
		}
	}
	return
}

// dump gather CSVs
func writeCSVs(days map[string][]Gather) (err error) {
	err = verify(days)
	Ck(err)
	for tt, gs := range days {
		// Pprint(gs)
		fn := day2fn(csvdir, tt, "csv")
		Pl(len(gs), fn)
		fh, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		Ck(err)
		defer fh.Close()
		err = gocsv.MarshalFile(gs, fh)
		Ck(err)
	}
	return
}

// dump gather JSON
func writeJSON(days map[string][]Gather) (err error) {
	err = verify(days)
	Ck(err)

	conffn := envi.String("GATHERBOT_CONF", ".gatherbot-conf.json")
	confbuf, err := ioutil.ReadFile(conffn)
	Ck(err)
	conf := &Conf{}
	err = json.Unmarshal(confbuf, conf)
	Ck(err)

	cmdfn := Spf("%s/cmds.sh", jsondir)
	cmdfh, err := os.OpenFile(cmdfn, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	Ck(err)
	defer cmdfh.Close()

	for tt, gs := range days {
		// Pprint(gs)
		fn := day2fn(jsondir, tt, "json")
		Pl(len(gs), fn)
		fh, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		Ck(err)
		defer fh.Close()

		// Pprint(conf)
		dmap, ok := conf.Days[TicketType(tt)]
		Assert(ok, tt)
		spaceId := dmap.SpaceId
		Assert(spaceId != "")
		overwrite := dmap.Overwrite

		ggl := make(map[string]GatherGuestDetail)
		gj := &GatherJSON{
			ApiKey:    os.Getenv("GATHER_API_KEY"),
			SpaceId:   spaceId,
			Overwrite: overwrite,
			Guestlist: ggl,
		}
		for _, g := range gs {
			ggl[g.Email] = GatherGuestDetail{
				Name: g.Name,
				Role: g.Role,
			}
		}

		buf, err := json.Marshal(gj)
		Ck(err)
		_, err = fh.Write(buf)
		Ck(err)

		if spaceId != "XXX" {
			// append curl command to cmds file
			// curl -i -H "Content-Type: application/json" --data @$dir/$dayfn 'https://api.gather.town/api/setEmailGuestlist'
			cmdtmpl := `curl -i -H "Content-Type: application/json" --data @%s 'https://api.gather.town/api/setEmailGuestlist'`
			cmd := Spf(cmdtmpl, fn)
			cmd = Spf("%s\n", cmd)
			cmdfh.Write([]byte(cmd))
		}

	}

	return
}

func day2fn(dir, day, ext string) (fn string) {
	fn = day
	fn = strings.ReplaceAll(fn, " ", "_")
	fn = strings.ReplaceAll(fn, "/", "_")
	fn = strings.ReplaceAll(fn, "&", "_")
	fn = strings.ReplaceAll(fn, "___", "_")
	fn = strings.ReplaceAll(fn, "__", "_")
	fn = Spf("%s/%s.%s", dir, fn, ext)
	return
}
