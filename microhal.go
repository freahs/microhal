// Package microhal provides a simple chatbot inspired by MegaHAL
package microhal

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/jcelliott/lumber"
)

var log = lumber.NewConsoleLogger(lumber.INFO)

type mData struct {
	Name     string
	Order    int
	Markov   *markov
	Keywords keywords
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

// NewMicrohal returns a new instance of Microhal. If there alreade exsitst save
// data associated with name name, it will be overwritten.
// Markov chains of order order will be used for the database.
func NewMicrohal(name string, order int) *Microhal {
	md := mData{
		Name:     name,
		Order:    order,
		Markov:   newMarkov(order),
		Keywords: make(keywords)}
	m := Microhal{md}
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

// processInput takes a input string and a maxLength and returns a string with a
// response more or less related to the input. Currently the max length of the
// string returned is actually maxLength*2+order...
func (m *Microhal) processInput(input string, maxLength int) string {
	order := m.getOrder()

	// The requested response must be longer than order, else there will be no
	// room for suffixes.
	if maxLength < order {
		log.Fatal("maxLength must greater than order. (Got %d, expected at least %d).", maxLength, order+1)
		os.Exit(-1)
	}

	// In order to produce a response, all potential prefixes are extracted from
	// the input which then are sorted by frequenecy in the database. Then key-
	// words are tried in order to get a response.
	keywords := m.data.Markov.GetKeywords(input)
	keywords = m.data.Keywords.sort(keywords)
	m.data.Keywords.add(keywords)
	for _, keyword := range keywords {
		output, err := m.data.Markov.GetString(keyword, maxLength)
		if err == nil {
			m.data.Markov.AddString(input)
			return output
		}
	}
	m.data.Markov.AddString(input)
	return ""
}
