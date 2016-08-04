package main

import (
	"log"

	"golang.org/x/net/context"

	"cloud.google.com/go/datastore"
)

type VersionInfo struct {
	PlayServicesVersion   string
	AndroidSupportVersion string
}

func main() {
	versions, err := getVersions()
	if err != nil {
		log.Fatalf("could not fetch latest versions: %v", err)
	}
	log.Printf("got versions: %#v", versions)

	ctx := context.Background()

	dc, err := datastore.NewClient(ctx, "gradleplease")
	if err != nil {
		log.Fatalf("could not connect to datastore: %v", err)
	}

	k := datastore.NewKey(ctx, "VersionInfo", "info", 0, nil)
	_, err = dc.Put(ctx, k, versions)
	if err != nil {
		log.Fatalf("could not update versions: %v", err)
	}
	log.Printf("successfully put to datastore")
}
