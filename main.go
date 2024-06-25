package main

import (
	"archive/zip"
	"bytes"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)



func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println("handler is working")

	var payload struct {
		Images []string `json:"images"`
	}

	err := json.Unmarshal([]byte(request.Body), &payload)
	
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid request body",
		}, nil
	}
	

	zipBuffer, err := createZipFromImages(payload.Images)
	if err != nil {
		log.Println("Error creating zip:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Error creating zip file",
		}, nil
	}

	s3URL, err := uploadToS3(zipBuffer)
	if err != nil {
		log.Println("Error uploading to S3:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Error uploading to S3",
		}, nil
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       s3URL,
	}

	return response, nil
}

func createZipFromImages(images []string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	
	
	for i, imgData := range images {
		// decodedImage, err := base64.StdEncoding.DecodeString(imgData)
		// if err != nil {
		// 	return nil, fmt.Errorf("error decoding image %d: %v", i, err)
		// }

		imgFileName := fmt.Sprintf("image_%d.jpg", i)
		zipFileWriter, err := zipWriter.Create(imgFileName)
		if err != nil {
			return nil, fmt.Errorf("error creating zip entry for image %d: %v", i, err)
		}

		_, err = zipFileWriter.Write([]byte(imgData))
		if err != nil {
			return nil, fmt.Errorf("error writing image %d to zip: %v", i, err)
		}
	}

	err := zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing zip writer: %v", err)
	}

	return buf, nil
}

func uploadToS3(zipBuffer *bytes.Buffer) (string, error) {
	const (
        REGION               = ""
        AWS_ACCESS_KEY_ID    = ""
        AWS_SECRET_ACCESS_KEY = ""
        AWS_IMAGE_BUCKET     = ""
    )

	sess, err := session.NewSessionWithOptions(session.Options{
        Config: aws.Config{
            Region:      aws.String(REGION),
            Credentials: credentials.NewStaticCredentials(AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, ""),
        },
    })

	svc := s3.New(sess)

	fileName := "images_" + time.Now().Format("20060102150405") + ".zip"
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(AWS_IMAGE_BUCKET),
		Key:    aws.String(fileName),
		Body:   bytes.NewReader(zipBuffer.Bytes()),
		ACL:    aws.String("public-read"),
	})

	if err != nil {
		return "", err
	}

	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", AWS_IMAGE_BUCKET, REGION, fileName)

	return s3URL, nil
}
