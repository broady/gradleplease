package gradlepls

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"
	"io/ioutil"
	"fmt"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"appengine/memcache"
)

func init() {
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/feedback", feedbackHandler)
}

type BaseLog struct {
	Timestamp time.Time
	Session	string
	UserAgent string
	IP string
}

type Search struct {
	BaseLog
	Query string
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	item, _ := memcache.Get(c, r.FormValue("q"))
	if item == nil {
		c.Infof("cache miss: %s", r.FormValue("q"))
		client := urlfetch.Client(c)
		v := make(url.Values)
		v.Add("q", r.FormValue("q"))
		v.Add("wt", "json")
		resp, err := client.Get("http://search.maven.org/solrsearch/select?" + v.Encode())
		if err != nil {
			c.Errorf("could not get url: %s - %v", v.Encode(), err)
			fmt.Fprintf(w, "searchCallback({error:true})")
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.Errorf("could not read body: %v", err)
			fmt.Fprintf(w, "searchCallback({error:true})")
			return
		}
		if err := json.Unmarshal(body, new(map[string]interface{})); err != nil {
			fmt.Fprintf(w, "searchCallback({error:true, tryagain:true})")
			return
		}
		item = &memcache.Item{
			Key: r.FormValue("q"),
			Value: body,
			Expiration: time.Hour * 3,
		}
		if err := memcache.Set(c, item); err != nil {
			c.Warningf("could not set item %s: %v", item.Key, err)
		}
	}
	fmt.Fprintf(w, "searchCallback(%s)", item.Value)
	s := &Search {
		Query: r.FormValue("q"),
	}
	s.fill(r)
	k := datastore.NewIncompleteKey(c, "Search", nil)
	if _, err := datastore.Put(c, k, s); err != nil {
		c.Errorf("could not put search: %q %q", err, s)
	}
}

type Feedback struct {
	BaseLog
	Query string
	Result string
	Good string
}

func feedbackHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	f := &Feedback {
		Query: r.FormValue("q"),
		Result: r.FormValue("result"),
		Good: r.FormValue("good"),
	}
	f.fill(r)
	k := datastore.NewIncompleteKey(c, "Feedback", nil)
	if _, err := datastore.Put(c, k, f); err != nil {
		c.Errorf("could not put feedback: %q %q", err, f)
	}
}

func (b *BaseLog) fill(r *http.Request) {
	b.Timestamp = time.Now()
	b.UserAgent = r.Header.Get("User-Agent")
	b.IP = r.RemoteAddr
	b.Session = r.FormValue("session")
}
