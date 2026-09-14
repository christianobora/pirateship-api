// Command rates prints public Pirate Ship rate estimates.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	pirateship "github.com/christianobora/pirateship-api"
)

func main() {
	originZIP := flag.String("from", "", "origin ZIP code")
	destinationZIP := flag.String("to", "", "destination ZIP code")
	weight := flag.Float64("ounces", 0, "package weight in ounces")
	length := flag.Float64("length", 6, "package length in inches")
	width := flag.Float64("width", 4, "package width in inches")
	height := flag.Float64("height", 2, "package height in inches")
	flag.Parse()

	if *originZIP == "" || *destinationZIP == "" || *weight <= 0 {
		log.Fatal("-from, -to, and a positive -ounces value are required")
	}

	client, err := pirateship.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	rates, _, err := client.Rates(context.Background(), pirateship.RatesRequest{
		OriginZIP:        *originZIP,
		DestinationZIP:   destinationZIP,
		WeightOunces:     weight,
		DimensionXInches: length,
		DimensionYInches: width,
		DimensionZInches: height,
		MailClassKeys:    []pirateship.MailClassKey{pirateship.MailClassGroundAdvantage},
		PackageTypeKeys:  []pirateship.PackageTypeKey{pirateship.PackageTypeParcel},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, rate := range rates {
		fmt.Printf("%-24s %-8s $%.2f\n", rate.Title, rate.Carrier.Key, rate.TotalPrice)
	}
}
