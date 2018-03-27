package main

import (
	"log"
	"os"

	"github.com/iandri/snowball/cmd"
)

// /opt/data/gluster/backups/weekly/cad/2017-11-18/cad_profile/108-111/2017-11-18T101424Z/2017-11-18T101424Z-full/bucket-cad_profile/node-gis-couchbase-bkg014.mdc.gameloft.org%3A8091/data-0000.cbb.gz

type cfg struct {
	awsID       string `yaml:"aws_id"`
	awsKey      string `yaml:"aws_key"`
	awsEndpoint string `yaml:"aws_endpoint"`
	awsRegion   string `yaml:"aws_region"`
	awsBucket   string `yaml:"aws_bucket"`
}

func main() {
	cliApp := cmd.App()
	if err := cliApp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
