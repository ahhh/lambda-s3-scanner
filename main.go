// main.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/andreyvit/diff"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var sess *session.Session

func init() {
	// set env variable
	//os.Setenv("EXAMPLE_PATH", os.Getenv("LAMBDA_TASK_ROOT"))

	// Setup AWS S3 Session (build once use every function)
	sess = session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}

func handler(ctx context.Context, s3Event events.S3Event) {
	for _, record := range s3Event.Records {
		s3Item := record.S3
		// Print Record Details
		fmt.Printf("[%s - %s] Bucket = %s, Key = %s \n", record.EventSource, record.EventTime, s3Item.Bucket.Name, s3Item.Object.Key)
		// Print full object as test

		// Get the file from S3
		latestContent, err := downloadFile(s3Item.Object.Key, s3Item.Bucket.Name, s3Item.Object.Key)
		if err != nil {
			fmt.Printf("Error fetching object: %v \n", err)
		}
		// Print versions of file (still needs to dynamically fetch version)
		previousVersion, err := listVersions(s3Item.Object.Key, s3Item.Bucket.Name, s3.New(sess))
		if err != nil || previousVersion == "" {
			fmt.Printf("Error listing versions: %v \n", err)
		}

		// Get a previous version (still needs to dynamically fetch version)
		previousContent, err := downloadPreviousVersion(s3Item.Object.Key, s3Item.Bucket.Name, s3Item.Object.Key, previousVersion)
		if err != nil {
			fmt.Printf("Error fetching object: %v \n", err)
		}
		// Diff the versions
		difContents(latestContent, previousContent)

	}
}

func downloadFile(fileItem, bucket, object string) (string, error) {
	file, err := os.Create("/tmp/" + fileItem)
	if err != nil {
		fmt.Printf("Error unable to open file %q, %v \n", fileItem, err)
		return "", err
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(sess)

	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(object),
		})
	if err != nil {
		fmt.Printf("Error unable to download item %q, %v \n", fileItem, err)
		return "", err
	}
	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")
	//fmt.Println("File Contents:")
	//scanner := bufio.NewScanner(file)
	//for scanner.Scan() {
	//	fmt.Println(scanner.Text())
	//}
	buf := new(bytes.Buffer)
	buf.ReadFrom(file)
	contents := buf.String()
	//fmt.Println(contents)
	return contents, nil
}

func downloadPreviousVersion(fileItem, bucket, object, version string) (string, error) {
	file, err := os.Create("/tmp/" + fileItem)
	if err != nil {
		fmt.Printf("Error unable to open file %q, %v \n", fileItem, err)
		return "", err
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(sess)

	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket:    aws.String(bucket),
			Key:       aws.String(object),
			VersionId: aws.String(version),
		})
	if err != nil {
		fmt.Printf("Error unable to download item %q, %v \n", fileItem, err)
		return "", err
	}
	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")
	//fmt.Println("File Contents:")
	buf := new(bytes.Buffer)
	buf.ReadFrom(file)
	contents := buf.String()
	//fmt.Println(contents)
	return contents, nil
}

// eventually this should return an array of the versions, in order from latest to oldest
func listVersions(object string, bucket string, S3 *s3.S3) (string, error) {
	req := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}

	res, err := S3.ListObjectVersions(req)
	if err != nil {
		return "", err
	}

	//for _, d := range res.DeleteMarkers {
	//	fmt.Printf("The bucket %s, has the object %s which has the following delete markerker: %v \n", bucket, *d.Key, *d.VersionId)
	//}
	var versions []string
	var previous = ""
	for _, v := range res.Versions {
		//fmt.Printf("The bucket %s, has the object %s which has the following version: %v \n", bucket, *v.Key, *v.VersionId)
		if *v.Key == object {
			fmt.Printf("The bucket %s, has the object %s which has the following version: %v \n", bucket, *v.Key, *v.VersionId)
			versions = append(versions, *v.VersionId)
			if len(versions) > 1 {
				//fmt.Printf("pervious version: %s \n", versions[1])
				previous = versions[1]
			}
		}
	}
	if previous == "" {
		return "", nil
	}
	fmt.Printf("All versions of the object %s : %v \n", object, versions)
	fmt.Printf("vMax: %s ; vMin: %s \n", versions[len(versions)-1], versions[0])
	return previous, nil
}

// compare lines function, this function is our line diff
func difContents(latest, previous string) {
	results := diff.LineDiff(previous, latest)
	fmt.Println(results)
}

// Send email or save to db aka alert

// Display on alerts?
