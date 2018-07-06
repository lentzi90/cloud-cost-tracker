package main

import (
	"flag"
	"log"

	"github.com/Azure/go-autorest/autorest"
)

var (
	subscriptionID = flag.String("subscription-id", "", "The ID of the subscription.")
	addr           = flag.String("listen-address", ":8080", "The address to listen on for (scraping) HTTP requests.")
)

var authorizer autorest.Authorizer

func main() {
	flag.Parse()
	if *subscriptionID == "" {
		log.Fatal("You must provide a subscription id by using the --subscription-id flag.")
	}

	usageExplorer := NewUsageExplorer(*subscriptionID)
	usageExplorer.PrintCurrentUsage()
}
