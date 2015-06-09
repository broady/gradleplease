package main

import (
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"
)

type VersionInfo struct {
	PlayServicesVersion   string
	AndroidSupportVersion string
}

func main() {
	privateKeyPath := os.Getenv("CLOUD_PRIVATE_KEY")
	if privateKeyPath == "" {
		log.Fatal("did not find private key at $CLOUD_PRIVATE_KEY")
	}

	versions, err := getVersions()
	if err != nil {
		log.Fatalf("could not fetch latest versions: %v", err)
	}
	log.Printf("got versions: %#v", versions)

	c := getContext()
	k := datastore.NewKey(c, "VersionInfo", "info", 0, nil)
	_, err = datastore.Put(c, k, versions)
	if err != nil {
		log.Fatalf("could not update versions: %v", err)
	}
	log.Printf("successfully put to datastore")
}

func getContext() context.Context {
	// Initialize an authorized context with Google Developers Console
	// JSON key. Read the google package examples to learn more about
	// different authorization flows you can use.
	// http://godoc.org/golang.org/x/oauth2/google
	jsonKey, err := ioutil.ReadFile(os.Getenv("CLOUD_PRIVATE_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	conf, err := google.JWTConfigFromJSON(
		jsonKey,
		datastore.ScopeDatastore,
		datastore.ScopeUserEmail,
	)
	if err != nil {
		log.Fatal(err)
	}
	return cloud.NewContext("gradleplease", conf.Client(oauth2.NoContext))
}
