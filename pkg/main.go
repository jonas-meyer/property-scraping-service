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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"os"
	"sync"
)

func init() {
	// Set log level
	logLevel, err := zerolog.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// Set output to stdout for Lambda
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func handler(ctx context.Context, s3Client s3.S3, gooplaClient goopla.Client, bucketName string, listingOptions goopla.ListingOptions) error {
	listings, _, err := gooplaClient.Listing.Get(ctx, &listingOptions)
	if err != nil {
		log.Error().Err(err).Msg("error getting listings")
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(listings.Listings))

	for _, listing := range listings.Listings {
		go func(obj goopla.Listing) {
			defer wg.Done()

			jsonBytes, err := json.MarshalIndent(&obj, "", "  ")
			if err != nil {
				log.Error().Err(err).Msg("Error marshaling JSON object")
				return
			}

			fileName := fmt.Sprintf("%s/%s/%s.json", listingOptions.Area, obj.Status, obj.ListingID)

			_, err = s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(fileName),
				Body:   bytes.NewReader(jsonBytes),
			})
			if err != nil {
				log.Error().Err(err).Msgf("Error uploading %s.json to S3: %v", obj.ListingID, err)
				return
			}

			log.Info().Msgf("Uploaded listing: %s to S3", fileName)
		}(listing)
	}

	wg.Wait()

	log.Info().Msgf("All listings have been uploaded to S3 bucket: %s", bucketName)

	return nil
}

func main() {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-2"),
	})
	if err != nil {
		log.Error().Err(err).Msg("error initializing AWS session")
		return
	}
	s3Session := s3.New(awsSession)

	client, err := goopla.NewClient(goopla.Credentials{}, goopla.FromEnv)
	if err != nil {
		log.Error().Err(err).Msg("error creating Goopla client")
		return
	}

	err = viper.BindEnv("lambda_environment")
	if err != nil {
		log.Error().Err(err).Msg("could not bind environment variable")
		return
	}

	bucketName := fmt.Sprintf("property-scraping-%s-listing-upload", viper.GetString("lambda_environment"))
	listingOptions := &goopla.ListingOptions{Area: "Oxford", Minimum_beds: 2, Maximum_beds: 2, Order_by: "age", Page_size: 10}

	lambda.Start(func(ctx context.Context) error {
		return handler(ctx, *s3Session, *client, bucketName, *listingOptions)
	})
}
