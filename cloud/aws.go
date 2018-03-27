package cloud

import (
	"bufio"
	"log"
	"os"
	"time"

	"fmt"

	"sync"

	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/dustin/go-humanize"
	"github.com/iandri/snowball/utils"
	"github.com/pkg/errors"
	"gopkg.in/cheggaaa/pb.v1"
)

type Uploader struct {
	Location string
	Size     int64
	Elapsed  time.Duration
}

func ListObjects(s3SVC *s3.S3, bucket, prefix string) (*s3.ListObjectsOutput, error) {
	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		//MaxKeys: aws.Int64(100),
		Prefix: aws.String(prefix),
	}
	objects, err := s3SVC.ListObjects(input)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return objects, nil
}

func ListObjectsAll(s3SVC *s3.S3, bucket string, prefix string) ([]*s3.Object, error) {
	var objects []*s3.Object

	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
	}
	err := s3SVC.ListObjectsPages(input, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		//fmt.Println(page.Contents)
		for _, v := range page.Contents {
			if prefix != "" {
				re := fmt.Sprintf(".*(%s).*", prefix)
				rgx, err := regexp.Compile(re)
				if err != nil {
					log.Println("Invalid regex, err:", err)
					return true
				}
				if rgx.MatchString(*v.Key) {
					objects = append(objects, v)
				}
			} else {
				objects = append(objects, v)
			}
		}
		return lastPage == false
	})
	return objects, err
}

func DeleteObjects(s3SVC *s3.S3, bucket string, keys []string, prefix string) (*s3.DeleteObjectsOutput, error) {
	if prefix != "" {
		objs, err := ListObjectsAll(s3SVC, bucket, prefix)
		if err != nil {
			return nil, err
		}

		for _, v := range objs {
			keys = append(keys, *v.Key)
		}
	}
	objects := make([]*s3.ObjectIdentifier, len(keys))
	for _, v := range keys {
		objects = append(objects, &s3.ObjectIdentifier{Key: aws.String(v)})
	}
	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &s3.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false),
		},
	}

	result, err := s3SVC.DeleteObjects(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return &s3.DeleteObjectsOutput{}, aerr
			}
		} else {
			return &s3.DeleteObjectsOutput{}, err
		}
	}
	return result, nil
}

func UploadObject(s3SVC *s3.S3, bucket string, partSize int64, threads int, src, dst string) (*Uploader, error) {
	uploadResult := new(Uploader)
	uploader := s3manager.NewUploaderWithClient(s3SVC, func(u *s3manager.Uploader) {
		u.PartSize = partSize * 1024 * 1024
		u.Concurrency = threads
		u.LeavePartsOnError = true
	})

	file, err := os.Open(src)
	if err != nil {
		return uploadResult, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		log.Print("could not get file size, ", errors.WithStack(err))
		return uploadResult, err
	}
	totalSize := fi.Size()

	start := time.Now().UTC()
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:                 bufio.NewReader(file),
		Bucket:               aws.String(bucket),
		Key:                  aws.String(dst),
		ServerSideEncryption: aws.String("false"),
	})
	if err != nil {
		//log.Fatal(errors.WithStack(err))
		if multierr, ok := err.(s3manager.MultiUploadFailure); ok {
			// Process error and its associated uploadID
			log.Println("Error:", multierr.Code(), multierr.Message(), multierr.UploadID())
			return uploadResult, err
		} else if aerr, ok := err.(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
			// If the SDK can determine the request or retry delay was canceled
			// by a context the CanceledErrorCode error code will be returned.
			log.Printf("upload canceled due to timeout, %v\n", err)
			return uploadResult, err
		} else {
			// Process error generically
			return uploadResult, err
		}
	}
	elapsed := time.Since(start)

	uploadResult.Location = result.Location

	uploadResult.Size = totalSize
	uploadResult.Elapsed = elapsed
	return uploadResult, nil
}

func MultiUploadObject(pb *pb.ProgressBar, wg *sync.WaitGroup, s3SVC *s3.S3, bucket string, partSize int64, threads int, src, dst string) (*Uploader, error) {
	defer func() {
		wg.Done()
		pb.Increment()
	}()
	uploadResult := new(Uploader)
	uploader := s3manager.NewUploaderWithClient(s3SVC, func(u *s3manager.Uploader) {
		u.PartSize = partSize * 1024 * 1024
		u.Concurrency = threads
		u.LeavePartsOnError = true
	})

	file, err := os.Open(src)
	if err != nil {
		return uploadResult, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		log.Print("could not get file size, ", errors.WithStack(err))
		return uploadResult, err
	}
	totalSize := fi.Size()

	start := time.Now().UTC()
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:                 bufio.NewReader(file),
		Bucket:               aws.String(bucket),
		Key:                  aws.String(dst),
		ServerSideEncryption: aws.String("false"),
	})
	if err != nil {
		if multierr, ok := err.(s3manager.MultiUploadFailure); ok {
			// Process error and its associated uploadID
			log.Println("Error:", multierr.Code(), multierr.Message(), multierr.UploadID())
			return uploadResult, err
		} else if aerr, ok := err.(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
			// If the SDK can determine the request or retry delay was canceled
			// by a context the CanceledErrorCode error code will be returned.
			log.Printf("upload canceled due to timeout, %v\n", err)
			return uploadResult, err
		} else {
			// Process error generically
			return uploadResult, err
		}
	}
	elapsed := time.Since(start)

	uploadResult.Location = result.Location

	uploadResult.Size = totalSize
	uploadResult.Elapsed = elapsed
	return uploadResult, nil
}

func (u Uploader) String() string {
	size := humanize.Bytes(uint64(u.Size))
	seconds := u.Elapsed.Seconds()
	elapsed := utils.HumanizeDuration(u.Elapsed)
	bandwidth := float64(u.Size) / seconds / 1024.0 / 1024.0
	return fmt.Sprintf("Location     : %s\nSize         : %s\nElapsed time : %s\nBandwidth    :% 4.0f MBytes/sec\n",
		u.Location, size, elapsed, bandwidth)
}
