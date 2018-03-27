package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/iandri/snowball/cloud"
	"github.com/iandri/snowball/job"
	"github.com/pkg/errors"
	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/urfave/cli.v1"
)

var s3SVC *s3.S3

func initialize(awsID, awsKey, awsEndpoint, awsRegion string, debug bool) error {
	var err error
	var timeout time.Duration
	timeout = 10

	ctx := context.Background()
	var cancelFn func()
	if timeout > 0 {
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
	}
	defer cancelFn()

	s3SVC, err = svcNew(awsID, awsKey, awsEndpoint, awsRegion, debug)
	if err != nil {
		return err
	}
	return nil
}

func svcNew(awsID, awsKey, awsEndpoint, awsRegion string, debug bool) (*s3.S3, error) {
	credsUp := credentials.NewStaticCredentials(awsID, awsKey, "")
	_, err := credsUp.Get()
	if err != nil {
		return &s3.S3{}, errors.WithStack(err)
	}

	snowConfig := &aws.Config{
		Credentials:             credsUp,
		Endpoint:                aws.String(awsEndpoint),
		Region:                  aws.String(awsRegion),
		DisableSSL:              aws.Bool(true),
		S3ForcePathStyle:        aws.Bool(true),
		S3Disable100Continue:    aws.Bool(true),
		DisableComputeChecksums: aws.Bool(true),
	}

	sessUp, err := session.NewSession(snowConfig)
	if err != nil {
		return &s3.S3{}, errors.WithStack(err)
	}

	if debug {
		sessUp.Config.WithLogLevel(
			// aws.LogDebugWithHTTPBody |
			aws.LogDebugWithRequestErrors |
				aws.LogDebugWithRequestRetries |
				aws.LogDebug)
	}
	s3Svc := s3.New(sessUp, snowConfig)
	return s3Svc, nil
}

func commandDebugObjects(c *cli.Context) error {
	fmt.Printf("aws_id: %s\n", c.GlobalString("aws_id"))
	fmt.Printf("aws_key: %s\n", c.GlobalString("aws_key"))
	fmt.Printf("aws_endpoint: %s\n", c.GlobalString("aws_endpoint"))
	fmt.Printf("aws_region: %s\n", c.GlobalString("aws_region"))
	return nil
}

func checkFlags(c *cli.Context) error {
	app := App()
	help := []string{"", "--help"}
	if c.GlobalString("aws_id") == "" {
		app.Run(help)
		return fmt.Errorf("aws_id is missing")
	}
	if c.GlobalString("aws_key") == "" {
		return fmt.Errorf("aws_key is missing")
	}
	if c.GlobalString("aws_endpoint") == "" {
		return fmt.Errorf("aws_endpoint is missing")
	}
	if c.GlobalString("aws_region") == "" {
		return fmt.Errorf("aws_region is missing")
	}
	if c.String("bucket") == "" {
		return fmt.Errorf("bucket is missing")
	}
	return nil
}

func commandListObjects(c *cli.Context) error {
	var content s3Obj
	var err error
	if err := checkFlags(c); err != nil {
		return err
	}
	initialize(c.GlobalString("aws_id"), c.GlobalString("aws_key"), c.GlobalString("aws_endpoint"),
		c.GlobalString("aws_region"), c.Bool("verbose"))
	content, err = cloud.ListObjectsAll(s3SVC, c.String("bucket"), c.String("prefix"))
	if err != nil {
		// log.Fatalln(err)
		return err
	}
	if c.Bool("group") {
		sort.Sort(content)
	} else {
		sort.Slice(content, func(i, j int) bool {
			return content[i].LastModified.Before(*content[j].LastModified)
		})
	}
	for _, v := range content {
		tme := v.LastModified.Format(time.RFC3339)
		fmt.Printf("Key: %15s, Modified: %s, Size: %d\n", *v.Key, tme, *v.Size)
	}
	return nil
}

func commandDeleteObjects(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		os.Exit(1)
	}
	if err := checkFlags(c); err != nil {
		return err
	}
	initialize(c.GlobalString("aws_id"), c.GlobalString("aws_key"), c.GlobalString("aws_endpoint"),
		c.GlobalString("aws_region"), c.Bool("verbose"))
	result, err := cloud.DeleteObjects(s3SVC, c.String("bucket"), c.StringSlice("keys"), c.String("prefix"))
	if err != nil {
		log.Fatalln(err)
	}
	for _, v := range result.Deleted {
		if v.Key != nil {
			fmt.Printf("Key %s deleted.\n", *v.Key)
		}
	}
	return nil
}

func commandUploadObjects(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		os.Exit(1)
	}
	if err := checkFlags(c); err != nil {
		return err
	}
	initialize(c.GlobalString("aws_id"), c.GlobalString("aws_key"), c.GlobalString("aws_endpoint"),
		c.GlobalString("aws_region"), c.Bool("verbose"))
	var dst string
	if c.String("dst") == "" {
		dst = c.String("src")
	} else {
		dst = c.String("dst")
	}
	result, err := cloud.UploadObject(s3SVC, c.String("bucket"), c.Int64("part"), c.Int("threads"),
		c.String("src"), dst)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(result)
	return nil
}

func commandSyncDirectory(c *cli.Context) error {
	if c.NumFlags() == 0 {
		//cli.ShowCommandHelp(c, "delete")
		cli.ShowSubcommandHelp(c)
		os.Exit(1)
	}
	if err := checkFlags(c); err != nil {
		return err
	}
	fullPath, files, err := scanDir(c.String("src"), c.String("filter"), c.String("prefix"))
	if err != nil {
		log.Fatalln(err)
	}

	var wg sync.WaitGroup

	if !c.Bool("dry") {
		initialize(c.GlobalString("aws_id"), c.GlobalString("aws_key"), c.GlobalString("aws_endpoint"),
			c.GlobalString("aws_region"), c.Bool("verbose"))
	}
	filesCount := len(files)
	bar := pb.StartNew(filesCount)
	bar.ShowTimeLeft = false
	job.StartDispather(c.Int("forks"))
	for i, file := range files {
		if c.Bool("dry") {
			wg.Add(1)
			fmt.Printf("uploading %s to s3://%s/%s\n", fullPath[i], c.String("bucket"), file)
			wg.Done()
		} else {
			wg.Add(1)
			job.Collector(bar, &wg, s3SVC, c.String("bucket"), c.Int64("part"), c.Int("threads"),
				fullPath[i], file)
		}
	}
	wg.Wait()
	bar.FinishPrint("Done!")
	return nil
}

func scanDir(searchDir, regex, prefix string) ([]string, []string, error) {
	pathRe := &regexp.Regexp{}
	filterRe := &regexp.Regexp{}

	if regex != "" {
		filterRe = regexp.MustCompile(regex)
	}

	if prefix != "" {
		re := fmt.Sprintf("%s/.*", prefix)
		pathRe = regexp.MustCompile(re)
	}

	fullFileList := make([]string, 0)
	fileList := make([]string, 0)
	e := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		switch {
		case regex != "":
			if filterRe.MatchString(path) {
				if !f.IsDir() {
					fullFileList = append(fullFileList, path)
					fileList = append(fileList, path)
				}
			}
		case prefix != "":
			if pathRe.MatchString(path) {
				if !f.IsDir() {
					fullFileList = append(fullFileList, path)
					path = pathRe.FindString(path)
					fileList = append(fileList, path)
				}
			}
		default:
			if !f.IsDir() {
				fullFileList = append(fullFileList, path)
				fileList = append(fileList, path)
			}
		}
		return err
	})
	if e != nil {
		return nil, nil, e
	}
	return fullFileList, fileList, nil
}

type s3Obj []*s3.Object

func (s s3Obj) Len() int {
	return len(s)
}

func (s s3Obj) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s s3Obj) Less(i, j int) bool {
	return *s[i].Key < *s[j].Key
}
