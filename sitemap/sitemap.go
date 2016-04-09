package sitemap

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type Freq struct {
	time.Duration
}

func (freq *Freq) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	switch s {
	case "":
		*freq = Freq{0}
	case "always":
		// Use second as the minimum unit of change frequence
		*freq = Freq{time.Second}
	case "hourly":
		*freq = Freq{time.Hour}
	case "daily":
		*freq = Freq{time.Hour * 24}
	case "weekly":
		*freq = Freq{time.Hour * 24 * 7}
	case "monthly":
		*freq = Freq{time.Hour * 24 * 30}
	case "yearly":
		*freq = Freq{time.Hour * 24 * 365}
	case "never":
		// time.Duration is int64
		*freq = Freq{1<<63 - 1}
	default:
		return errors.New("invalid frequence: " + s)
	}
	return nil
}

type Time struct {
	time.Time
}

func (t *Time) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	layouts := []string{
		"2006-01-02",
		"2006-01-02T15:04Z07:00",
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01",
		"2006",
	}
	var s string
	var err error
	if err = d.DecodeElement(&s, &start); err != nil {
		return err
	}
	for _, layout := range layouts {
		if t.Time, err = time.Parse(layout, s); err == nil {
			break
		}
	}
	return err
}

func (u *URL) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var tmp struct {
		Loc          string  `xml:"loc"`
		Priority     float64 `xml:"priority"`
		ChangeFreq   Freq    `xml:"changefreq"`
		LastModified Time    `xml:"lastmod"`
	}
	var err error
	if err = d.DecodeElement(&tmp, &start); err != nil {
		return err
	}
	var p *url.URL
	if p, err = url.Parse(tmp.Loc); err != nil {
		return err
	}
	u.Loc = *p
	u.LastModified = tmp.LastModified.Time
	u.ChangeFreq = tmp.ChangeFreq.Duration
	u.Priority = tmp.Priority
	return nil
}

type URL struct {
	Loc          url.URL
	Priority     float64
	ChangeFreq   time.Duration
	LastModified time.Time
}

type Sitemap struct {
	URLSet []URL `xml:"url"`
}

func Get(url string) (*Sitemap, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sm Sitemap
	if err = xml.Unmarshal(data, &sm); err != nil {
		return nil, err
	}
	return &sm, nil
}

/*
func main() {
	data, err := ioutil.ReadFile("sitemap_at.xml")
	if err != nil {
		panic(err)
	}

	var sm Sitemap
	if err := xml.Unmarshal(data, &sm); err != nil {
		panic(err)
	}

	data, err = json.MarshalIndent(&sm, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
*/
