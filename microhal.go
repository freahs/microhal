// Package microhal provides a simple chatbot inspired by MegaHAL
package microhal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"freahs/randmap"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jcelliott/lumber"
)

const stopChars = "!?."

var log = lumber.NewConsoleLogger(lumber.INFO)

type mData struct {
	Name        string
	Order       int
	States      map[string]*randmap.RandMap
	StartStates *randmap.RandMap
	EndStates   *randmap.RandMap
}

type Microhal struct {
	data mData
}

func (m *Microhal) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &m.data)
}
func (m *Microhal) MarshalJSON() ([]byte, error) {
	return json.Marshal(&m.data)
}

func (m *Microhal) GetName() string {
	return m.data.Name
}

func (m *Microhal) getOrder() int {
	return m.data.Order
}

func (m *Microhal) getStates() map[string]*randmap.RandMap {
	return m.data.States
}

func (m *Microhal) getStartStates() *randmap.RandMap {
	return m.data.StartStates
}

func (m *Microhal) getEndStates() *randmap.RandMap {
	return m.data.EndStates
}

// NewMicrohal returns a new instance of Microhal. If there alreade exsitst save
// data associated with name name, it will be overwritten.
// Markov chains of order order will be used for the database.
func NewMicrohal(name string, order int) *Microhal {
	md := mData{
		Name:        name,
		Order:       order,
		States:      make(map[string]*randmap.RandMap),
		StartStates: randmap.New(),
		EndStates:   randmap.New()}
	m := Microhal{md}
	m.processInput("I have nothing to say to you...", order+1)
	m.save()
	return &m
}

// LoadMicrohal returns a new instance of Microhal, initialized from save data
// associated with name.
func LoadMicrohal(name string) *Microhal {
	jsonData, err := ioutil.ReadFile(name + ".json")
	if err != nil {
		log.Fatal("%s", err)
		os.Exit(-1)
	}

	var md mData
	err = json.Unmarshal(jsonData, &md)
	if err != nil {
		log.Fatal("%s", err)
		os.Exit(-1)
	}
	m := Microhal{md}

	return &m
}

// Start returns two chanels for input and output to this microhol. Strings sent
// to input chanel will be processed and a response of at most maxLength words
// will be sent to the output channel. After each processed input there is a 500
// ms. delay. Each saveDuration the current state will be saved to disk.
func (m *Microhal) Start(saveInterval time.Duration, maxLength int) (chan<- string, <-chan string) {

	in := make(chan string)
	out := make(chan string)
	var wg sync.WaitGroup

	go func(recieve <-chan string, transmit chan<- string) {
		for {

			input := <-recieve
			log.Info("Recieved:  \"%s\"", input)
			wg.Wait()
			wg.Add(1)
			output := m.processInput(input, maxLength)
			log.Info("Responded: \"%s\"", output)
			transmit <- output
			wg.Done()
		}
	}(in, out)

	go func(saveInterval time.Duration) {
		for {
			time.Sleep(saveInterval)
			wg.Wait()
			wg.Add(1)
			log.Info("Saving")
			m.save()
			wg.Done()
		}
	}(saveInterval)

	return in, out
}

func (m *Microhal) save() error {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(m.GetName()+".json", jsonData, 0644); err != nil {
		return err
	}

	return nil
}

func (m *Microhal) addTransition(from, to string) {
	if t, ok := m.getStates()[from]; ok {
		t.Add(to)
	} else {
		t := randmap.New()
		t.Add(to)
		m.getStates()[from] = t
	}
}

func (m *Microhal) getTransition(from string) (string, error) {
	if t, ok := m.getStates()[from]; ok {
		return t.Get()
	}
	return "", fmt.Errorf("No transition from %s found.", from)
}

// processInput takes a input string and a maxLength. It will process the string
// and, if applicable, add it to the database and the produce a response. The
// respons may or may not be related to the input, depending on present data in
// the database.
func (m *Microhal) processInput(input string, maxLength int) string {
	order := m.getOrder()
	if maxLength < order {
		log.Fatal("maxLength must greater than order. (Got %d, expected at least %d).", maxLength, order+1)
		os.Exit(-1)
	}
	iStates := regexp.MustCompile("[\\S]+").FindAllString(input, -1)
	rMap := randmap.New()
	n := len(iStates) - order

	// Adding input to database
	for i := 0; i < n; i++ {
		from := processConcat(iStates[i : i+order])
		to := iStates[i+order]
		if m.getStartStates().Contains(from) {
			rMap.Add(from)
		}
		if isStopChar(to[len(to)-1]) {
			m.getEndStates().Add(to)
			a := i + order + 1
			b := a + order
			if b < n {
				s := processConcat(iStates[a:b])
				m.getStartStates().Add(s)
			}
		}
		m.addTransition(from, to)
	}

	// Adding first input-type state to StartStates
	if n > 0 {
		s := processConcat(iStates[0:order])
		m.getStartStates().Add(s)
	}

	// Adding last output-type state to EndStates
	m.getEndStates().Add(iStates[len(iStates)-1])

	// Get a start state. There should exist at least one...
	start, err := rMap.Get()
	if err != nil {
		start, err = m.getStartStates().Get()
		if err != nil {
			log.Fatal("---Couldn't produce a response to input \"%s\"", input)
			os.Exit(-1)
		}
	}

	// Add start states to output states
	oStates := make([]string, maxLength)
	sStates := strings.Split(start, " ")
	for i := 0; i < len(sStates); i++ {
		oStates[i] = sStates[i]
	}

	// Populeate output states slice
	for i := 0; i < cap(oStates)-order; i++ {
		from := processConcat(oStates[i : i+order])
		to, err := m.getTransition(from)
		if err != nil {
			break
		}
		oStates[i+order] = to
		if m.getEndStates().Contains(to) {
			break
		}
	}

	// Trim and return
	r := strings.TrimSpace(processConcat(oStates))
	return r
}

// processConcat is used by processinput to concatinate output-type states to
// a sentence string.
func processConcat(s []string) string {
	var b bytes.Buffer
	for i := 0; i < len(s); i++ {
		b.WriteString(s[i])
		b.WriteString(" ")
	}
	r := b.String()
	return r[:len(r)-1]
}

// isStopChar returns true if the char is present in constant stopChars, false
// if not.
func isStopChar(a uint8) bool {
	for _, b := range stopChars {
		if string(a) == string(b) {
			return true
		}
	}
	return false
}
