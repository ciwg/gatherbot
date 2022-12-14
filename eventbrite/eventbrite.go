package eventbrite

// derived from https://github.com/eco/linkedup/blob/master/eventbrite/eventbrite.go

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/stevegt/goadapt"
)

const (
	// EventEnvKey -
	EventEnvKey = "EVENTBRITE_EVENT"
	// AuthEnvKey -
	AuthEnvKey = "EVENTBRITE_AUTH"

	urlFormat  = "https://www.eventbriteapi.com/v3/events/%d/attendees/?page=%d"
	authFormat = "Bearer %s"
)

var netClient = &http.Client{
	Timeout: 5 * time.Second,
}

type Attendee struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	PlatformEmail string
	TicketType    string `json:"ticket_class_name"`
	Answers       []Answer
}

type Answer struct {
	Question   string
	QuestionId string `json:"question_id"`
	Type       string
	Answer     string
}

// AttendeeProfile -
type AttendeeProfile struct {
	ID int `json:"id"`

	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetAttendees -
func GetAttendees(eventID int, authToken string) ([]Attendee, error) {
	currentPage := 1
	hasMore := false
	var attendees []Attendee

	var err error
	attendees, hasMore, err = fetchPage(eventID, authToken, currentPage)
	if err != nil {
		err = fmt.Errorf("page fetch from eventbrite: %s", err)
		return nil, err
	}
	currentPage++

	for hasMore {
		var att2 []Attendee
		att2, hasMore, err = fetchPage(eventID, authToken, currentPage)
		if err != nil {
			err = fmt.Errorf("page fetch from eventbrite: %s", err)
			return nil, err
		}

		currentPage++
		attendees = append(attendees, att2...)
	}

	return attendees, nil
}

// GetAttendeesFromEnv -
func GetAttendeesFromEnv() ([]Attendee, error) {
	eventStr := os.Getenv(EventEnvKey)
	authToken := os.Getenv(AuthEnvKey)
	if len(eventStr) == 0 || len(authToken) == 0 {
		err := fmt.Errorf("%s and %s environment variables must be set to communicate with eventbrite",
			EventEnvKey, AuthEnvKey)
		return nil, err
	}

	eventID, err := strconv.Atoi(eventStr)
	if err != nil {
		err = fmt.Errorf("event id must be a positive number in decimal format: %s", err)
		return nil, err
	}

	return GetAttendees(eventID, authToken)
}

func fetchPage(eventID int, authToken string, page int) (attendees []Attendee, hasMore bool, err error) {
	type pageInfo struct {
		// only things we care about within the page struct
		HasMore bool `json:"has_more_items"`
	}
	type bodyResp struct {
		Page      pageInfo          `json:"pagination"`
		Attendees []json.RawMessage `json:"attendees"`
	}

	auth := fmt.Sprintf(authFormat, authToken)
	url, err := url.Parse(fmt.Sprintf(urlFormat, eventID, page))
	if err != nil {
		err = fmt.Errorf("parsing url: %s", err)
		return nil, false, err
	}
	req := &http.Request{
		URL:    url,
		Method: "GET",
		Header: map[string][]string{
			"Authorization": {auth},
		},
	}
	resp, err := netClient.Do(req)
	if err != nil {
		err = fmt.Errorf("eventbrite request delivery: %s", err)
		return nil, false, err
	}
	defer resp.Body.Close() //nolint
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad response. status code=%d", resp.StatusCode)
		return nil, false, err
	}

	// decode body
	decoder := json.NewDecoder(resp.Body)
	var data bodyResp
	err = decoder.Decode(&data)
	if err != nil {
		err = fmt.Errorf("reading request body: %s", err)
		return nil, false, err
	}

	// retrieve attendees
	numAttendees := len(data.Attendees)
	attendees = make([]Attendee, numAttendees)
	for i := 0; i < numAttendees; i++ {
		// Pl(string(data.Attendees[i]))
		profile, err2 := getProfile(data.Attendees[i])
		if err2 != nil {
			return nil, false, err2
		}

		/*
			billing, err2 := getBilling(data.Attendees[i])
			if err2 != nil {
				return nil, false, err2
			}
			_ = billing
		*/

		// Pprint(data.Attendees[i])

		attendee := &Attendee{}
		err := json.Unmarshal(data.Attendees[i], attendee)
		Ck(err)

		attendee.Name = profile.Name
		attendee.Email = strings.TrimSpace(profile.Email)

		for _, ans := range attendee.Answers {
			// "Attendee Email Address (this will be how you login to our platform, so please give us the email you intend to use for login)"
			if ans.QuestionId == "76109979" {
				attendee.PlatformEmail = strings.TrimSpace(ans.Answer)
			}
		}

		// Pprint(attendee)

		attendees[i] = *attendee
	}

	hasMore = data.Page.HasMore
	err = nil

	return attendees, hasMore, err
}

func getBilling(body json.RawMessage) (*AttendeeProfile, error) {
	/** Parse the entire attendee body **/
	var jsonResp map[string]json.RawMessage
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		err = fmt.Errorf("parsing eventbrite attendee: %s", err)
		return nil, err
	}
	// Pprint(jsonResp)
	return nil, nil
}

func getProfile(body json.RawMessage) (*AttendeeProfile, error) {
	/** Parse the entire attendee body **/
	var jsonResp map[string]json.RawMessage
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		err = fmt.Errorf("parsing eventbrite attendee: %s", err)
		return nil, err
	}

	var idStr string
	if err := json.Unmarshal(jsonResp["id"], &idStr); err != nil {
		return nil, fmt.Errorf("unable to parse attendee id: %s", err)
	}

	attendeeID, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, fmt.Errorf("expected attendee id to be a number")
	}

	/** Extract specifically the profile key of the response **/
	var jsonProfile map[string]json.RawMessage
	profileData, ok := jsonResp["profile"]
	if !ok {
		return nil, fmt.Errorf("attendee missing profile")
	} else if err := json.Unmarshal(profileData, &jsonProfile); err != nil { //nolint
		err = fmt.Errorf("parsing attendee profile: %s", err)
		return nil, err
	}

	/** Decode the struct into the fields we want **/
	var profile AttendeeProfile
	err = json.Unmarshal(profileData, &profile)
	if err != nil {
		err = fmt.Errorf("decoding profile: %s", err)
		return nil, err
	}
	profile.ID = attendeeID

	return &profile, nil
}
