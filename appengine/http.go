package gradlepls

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"text/template"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/feedback", feedbackHandler)
	http.HandleFunc("/app.js", appjsHandler)
}

var appjsTmpl = template.Must(template.ParseFiles("app.js"))

type BaseLog struct {
	Timestamp time.Time
	Session   string
	UserAgent string
	IP        string
}

type Search struct {
	BaseLog
	Query string
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	item, _ := memcache.Get(c, "_"+r.FormValue("q"))
	if item == nil {
		log.Infof(c, "cache miss: %s", r.FormValue("q"))
		client := urlfetch.Client(c)
		v := make(url.Values)
		v.Add("q", r.FormValue("q"))
		v.Add("wt", "json")
		resp, err := client.Get("http://search.maven.org/solrsearch/select?" + v.Encode())
		if err != nil {
			log.Errorf(c, "could not get url: %s - %v", v.Encode(), err)
			fmt.Fprintf(w, "searchCallback({error:true})")
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf(c, "could not read body: %v", err)
			fmt.Fprintf(w, "searchCallback({error:true})")
			return
		}
		if err := json.Unmarshal(body, new(map[string]interface{})); err != nil {
			fmt.Fprintf(w, "searchCallback({error:true, tryagain:true})")
			return
		}
		item = &memcache.Item{
			Key:        "_" + r.FormValue("q"),
			Value:      body,
			Expiration: time.Hour * 3,
		}
		if err := memcache.Set(c, item); err != nil {
			log.Warningf(c, "could not set item %s: %v", item.Key, err)
		}
	}
	fmt.Fprintf(w, "searchCallback(%s)", item.Value)
	s := &Search{
		Query: r.FormValue("q"),
	}
	s.fill(r)
	k := datastore.NewIncompleteKey(c, "Search", nil)
	if _, err := datastore.Put(c, k, s); err != nil {
		log.Errorf(c, "could not put search: %q %q", err, s)
	}
}

type Feedback struct {
	BaseLog
	Query  string
	Result string
	Good   string
}

type VersionInfo struct {
	PlayServicesVersion   string
	AndroidSupportVersion string
}

func appjsHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	v := &VersionInfo{"+", "+"}
	item, _ := memcache.Get(c, "_"+r.FormValue("q"))
	if item == nil {
		k := datastore.NewKey(c, "VersionInfo", "info", 0, nil)
		if err := datastore.Get(c, k, v); err != nil {
			log.Errorf(c, "could not fetch versioninfo: %v", err)
			appjsTmpl.Execute(w, v)
			return
		}
		body, err := json.Marshal(k)
		if err != nil {
			log.Errorf(c, "could not marshal versioninfo for memcache: %v", err)
			appjsTmpl.Execute(w, v)
			return
		}
		item = &memcache.Item{
			Key:        "m2versions",
			Value:      body,
			Expiration: time.Hour * 3,
		}
		if err := memcache.Set(c, item); err != nil {
			log.Warningf(c, "could not set item %s: %v", item.Key, err)
		}
	} else {
		if err := json.Unmarshal(item.Value, v); err != nil {
			fmt.Fprintf(w, "TODO error: %v", err)
			return
		}
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "max-age: 10800, public")
	if err := appjsTmpl.Execute(w, v); err != nil {
		log.Errorf(c, "could not write app.js tmpl: %q", err)
	}
}

func feedbackHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	f := &Feedback{
		Query:  r.FormValue("q"),
		Result: r.FormValue("result"),
		Good:   r.FormValue("good"),
	}
	f.fill(r)
	k := datastore.NewIncompleteKey(c, "Feedback", nil)
	if _, err := datastore.Put(c, k, f); err != nil {
		log.Errorf(c, "could not put feedback: %q %q", err, f)
	}
}

func (b *BaseLog) fill(r *http.Request) {
	b.Timestamp = time.Now()
	b.UserAgent = r.Header.Get("User-Agent")
	b.IP = r.RemoteAddr
	b.Session = r.FormValue("session")
}
