package main

import (
	"context"
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

func handler(ctx context.Context, s3Client s3.S3, bucketName string, listingOptions goopla.ListingOptions) error {
	return nil
}

func main() {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Error().Err(err).Msg("error initializing AWS session")
		return
	}
	s3Session := s3.New(awsSession)

	err = viper.BindEnv("lambda_environment")
	if err != nil {
		log.Error().Err(err).Msg("could not bind environment variable")
		return
	}

	bucketName := fmt.Sprintf("property-scraping-%s-listing-upload", viper.GetString("lambda_environment"))
	listingOptions := &goopla.ListingOptions{Area: "Oxford", Minimum_beds: 2, Maximum_beds: 2, Order_by: "age", Page_size: 10}

	lambda.Start(func(ctx context.Context) error {
		return handler(ctx, *s3Session, bucketName, *listingOptions)
	})
}
