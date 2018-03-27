package cmd

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v1"
	"gopkg.in/urfave/cli.v1/altsrc"
)

// App main cli
func App() *cli.App {
	app := cli.NewApp()
	app.Name = "snowball"
	app.Version = "0.0.1"
	app.Description = "AWS snowball manager"
	app.EnableBashCompletion = true
	app.BashComplete = func(c *cli.Context) {
		fmt.Fprintf(c.App.Writer, "list\nupload\ndelete\n")
	}
	app.Authors = []cli.Author{
		{
			Name:  "Ionut Andriescu",
			Email: "ionut.andriescu@gameloft.com",
		},
	}
	app.Commands = commands()
	flags := flags()

	if _, err := os.Stat("snowball.conf"); err == nil {
		app.Before = altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("cfg"))
	}
	app.Flags = flags
	return app
}

func flags() []cli.Flag {
	flags := []cli.Flag{
		altsrc.NewStringFlag(cli.StringFlag{
			Name: "aws_id",
		}),
		altsrc.NewStringFlag(cli.StringFlag{
			Name: "aws_key",
		}),
		altsrc.NewStringFlag(cli.StringFlag{
			Name: "aws_endpoint",
		}),
		altsrc.NewStringFlag(cli.StringFlag{
			Name: "aws_region",
		}),
		cli.StringFlag{
			Name:  "cfg",
			Value: "snowball.conf",
		},
	}
	return flags
}

func commands() []cli.Command {
	cmds := []cli.Command{
		{
			Name:  "list",
			Usage: "list objects",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "bucket, b",
					Usage: "source bucket",
					Value: "test-cbbackup",
				},
				cli.StringFlag{
					Name:  "prefix, p",
					Usage: "prefix to filter results",
				},
				cli.BoolFlag{
					Name:  "group, g",
					Usage: "group objects by name",
				},
				cli.BoolFlag{
					Name:  "verbose, v",
					Usage: "debug enabled",
				},
			},
			Action: commandListObjects,
			// Action: commandDebugObjects,
		},
		{
			Name:  "delete",
			Usage: "delete object(s)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "bucket, b",
					Usage: "source bucket",
					Value: "test-cbbackup",
				},
				cli.StringSliceFlag{
					Name:  "keys, k",
					Usage: "key(s) to delete",
				},
				cli.StringFlag{
					Name:  "prefix, p",
					Usage: "prefix to filter results",
				},
				cli.BoolFlag{
					Name:  "verbose, v",
					Usage: "debug enabled",
				},
			},
			Action: commandDeleteObjects,
		},
		{
			Name:  "upload",
			Usage: "upload object(s)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "bucket, b",
					Usage: "source bucket",
					Value: "test-cbbackup",
				},

				cli.StringFlag{
					Name:  "src, s",
					Usage: "source file to upload",
				},
				cli.StringFlag{
					Name:  "dst, d",
					Usage: "destination object in s3",
				},
				cli.Int64Flag{
					Name:  "part, p",
					Usage: "chunk part size in MB",
					Value: 32,
				},
				cli.IntFlag{
					Name:  "threads, t",
					Usage: "number of threads to upload",
					Value: 3,
				},
				cli.BoolFlag{
					Name:  "verbose, v",
					Usage: "debug enabled",
				},
			},
			Action: commandUploadObjects,
		},
		{
			Name:  "sync",
			Usage: "sync source directory to snowball",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "bucket, b",
					Usage: "source bucket",
					Value: "test-cbbackup",
				},
				cli.StringFlag{
					Name:  "src, s",
					Usage: "source directory",
				},
				cli.StringFlag{
					Name:  "filter, f",
					Usage: "regex to filter",
					Value: "",
				},
				cli.StringFlag{
					Name:  "prefix, x",
					Usage: "s3 object path starts with this prefix",
					Value: "",
				},
				cli.Int64Flag{
					Name:  "part, p",
					Usage: "chunk part size in MB",
					Value: 32,
				},
				cli.IntFlag{
					Name:  "threads, t",
					Usage: "number of threads to upload chunks in parallel",
					Value: 3,
				},
				cli.IntFlag{
					Name:  "forks, ff",
					Usage: "number of files to be processed in parallel",
					Value: 32,
				},
				cli.BoolFlag{
					Name:  "dry, d",
					Usage: "dry-run, does not upload",
				},
				cli.BoolFlag{
					Name:  "verbose, v",
					Usage: "debug enabled",
				},
			},
			Action: commandSyncDirectory,
		},
	}
	return cmds
}
