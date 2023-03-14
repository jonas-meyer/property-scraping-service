package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jonas-meyer/goopla/goopla"
	"sync"
)

func handler(ctx context.Context) error {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-2"),
	})
	if err != nil {
		return fmt.Errorf("error initializing AWS session: %v", err)
	}
	s3Session := s3.New(awsSession)

	client, err := goopla.NewClient(goopla.Credentials{}, goopla.FromEnv)
	if err != nil {
		return err
	}

	listingOptions := &goopla.ListingOptions{Area: "Oxford", Minimum_beds: 2, Maximum_beds: 2, Order_by: "age", Page_size: 10}
	listings, _, err := client.Listing.Get(ctx, listingOptions)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(listings.Listings))

	for _, listing := range listings.Listings {
		go func(obj goopla.Listing) {
			defer wg.Done()

			jsonBytes, err := json.MarshalIndent(obj, "", "  ")
			if err != nil {
				fmt.Printf("Error marshaling JSON object: %v\n", err)
				return
			}

			_, err = s3Session.PutObjectWithContext(ctx, &s3.PutObjectInput{
				Bucket: aws.String("property-scraping-development-listing-upload"), //TODO set bucket dynamically depending on environment                                                                          // Change this to your S3 bucket name
				Key:    aws.String(fmt.Sprintf("%s/%s/%s.json", listingOptions.Area, obj.Status, obj.ListingID)),
				Body:   bytes.NewReader(jsonBytes),
			})
			if err != nil {
				fmt.Printf("Error uploading %s.json to S3: %v\n", obj.ListingID, err)
				return
			}

			fmt.Printf("Uploaded %s.json to S3\n", obj.ListingID)
		}(listing)
	}

	wg.Wait()

	fmt.Println("All uploads complete")

	return nil
}

func main() {
	lambda.Start(handler)
}
