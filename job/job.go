package job

import (
	"fmt"
	"log"

	"sync"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/iandri/snowball/cloud"
	"gopkg.in/cheggaaa/pb.v1"
)

type WorkRequest struct {
	S3SVC    *s3.S3
	Bucket   string
	PartSize int64
	Threads  int
	Src      string
	Dst      string
	WG       *sync.WaitGroup
	PB       *pb.ProgressBar
}

type Worker struct {
	ID          int
	Work        chan WorkRequest
	WorkerQueue chan chan WorkRequest
	QuitChan    chan bool
}

var WorkerQueue chan chan WorkRequest

// A buffered channel that we can send work requests on.
var WorkQueue = make(chan WorkRequest, 100)

func StartDispather(nWorkers int) {
	// First, initialize the channel
	WorkerQueue = make(chan chan WorkRequest, nWorkers)

	// Now create all of our workers.
	for i := 0; i < nWorkers; i++ {
		//fmt.Println("Starting worker", i+1)
		worker := NewWorker(i+1, WorkerQueue)
		worker.Start()
	}

	go func() {
		for {
			select {
			case work := <-WorkQueue:
				//fmt.Println("Received work request.")
				go func() {
					worker := <-WorkerQueue
					//fmt.Println("Dispatching work request")
					worker <- work
				}()
			}
		}
	}()
}

func NewWorker(id int, workerQueue chan chan WorkRequest) Worker {
	worker := Worker{
		ID:          id,
		Work:        make(chan WorkRequest),
		WorkerQueue: workerQueue,
		QuitChan:    make(chan bool),
	}
	return worker
}

func (w *Worker) Start() {
	go func() {
		for {
			w.WorkerQueue <- w.Work

			select {
			case work := <-w.Work:
				//fmt.Printf("worker%d: Received work request, processing file %s\n",
				//	w.ID, work.Src)

				//time.Sleep(work.Delay)
				_, err := cloud.MultiUploadObject(work.PB, work.WG, work.S3SVC, work.Bucket,
					work.PartSize, work.Threads, work.Src, work.Dst)
				if err != nil {
					log.Println(err)
				}

				//fmt.Printf("worker%d: Hello, %s!\n", w.ID, work.Name)

			case <-w.QuitChan:
				// We have been asked to stop.
				fmt.Printf("worker%d stopping\n", w.ID)
			}
		}
	}()
}

// Stop the worker.
func (w *Worker) Stop() {
	go func() {
		w.QuitChan <- true
	}()
}

func Collector(pb *pb.ProgressBar, wg *sync.WaitGroup, s3SVC *s3.S3, bucket string,
	partSize int64, threads int, src, dst string) {
	work := WorkRequest{PB: pb, WG: wg, S3SVC: s3SVC, Bucket: bucket, PartSize: partSize,
		Threads: threads, Src: src, Dst: dst}
	WorkQueue <- work
	//fmt.Println("Work request queued")
}
