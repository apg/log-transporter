package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	S3Region = "us-west-2"
)

type Line []byte

// Batch holds an ordered set of lines.
//  TODO: we want some sort of limits in terms of lines or total size.
type Batch struct {
	name  string
	lines []Line

	// read cursor
	cursor     int
	lineCursor int

	// size in bytes for the whole buffer
	size int64
}

func NewBatch(name string, c int) Batch {
	// this should be an input to batch. But, perhaps the name should be passed
	// in explicitly
	return Batch{
		name:  name,
		lines: make([]Line, 0, c),
	}
}

// Count returns the number of lines in the batch
func (b *Batch) Count() int {
	return len(b.lines)
}

func (b *Batch) Size() int64 {
	return b.size
}

func (b *Batch) Name() string {
	return b.name
}

// WriteLine returns false if the batch is full.
// If line does not have a terminated newline, a single one will be added.
func (b *Batch) WriteLine(l []byte) bool {
	if len(b.lines) == cap(b.lines) {
		return false
	}
	l = bytes.TrimRight(l, "\n")
	b.lines = append(b.lines, l)
	b.size += int64(len(l) + 1)
	return true
}

// Read populates p with as many lines as can be included, writing partial lines when full lines will not fit. Returns io.EOF when no lines are left.
func (b *Batch) Read(p []byte) (n int, err error) {

	var next byte
	for ; n < cap(p); n++ {
		if b.cursor < len(b.lines) {
			if b.lineCursor < len(b.lines[b.cursor]) {
				next = b.lines[b.cursor][b.lineCursor]
				b.lineCursor = b.lineCursor + 1
			} else {
				next = '\n'
				b.lineCursor = 0
				b.cursor = b.cursor + 1
			}
		} else {
			return n, io.EOF
		}
		p[n] = next
	}

	return n, nil
}

func readBatches(ctx context.Context, batches chan<- Batch) {
	newName := func() string {
		ts := time.Now().UTC()
		return fmt.Sprintf("/vault/audit/%s.log", ts.Format("2006/01/02/15/0405.000000"))
	}

	b := NewBatch(newName(), 10)
	r := bufio.NewReader(os.Stdin)
	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			log.Printf("Sending batch of %d lines.\n", b.Count())
			return
		} else if err != nil {
			log.Println(err)
			continue
		}

		if !b.WriteLine(line) {
			log.Printf("Sending batch of %d lines.\n", b.Count())
			batches <- b
			b = NewBatch(newName(), 10)
		}
	}
}

func writeBatches(ctx context.Context, batches chan Batch, bucket string, s *session.Session) error {

	uploader := s3manager.NewUploader(s)

	// Upload
	for {
		select {
		case b := <-batches:
			fmt.Println("got a batch to upload")
			result, err := uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(b.Name()),
				Body:   &b,
			})
			fmt.Println("HELLO", result, err)
			if err != nil {
				return fmt.Errorf("upload %s: %w", b.Name(), err)
			}
			log.Println(result, err)

		case <-ctx.Done():
			return nil
		}
	}

	return nil
}

var creds = flag.String("credentials", "", "path to credentials file")
var bucket = flag.String("bucket", "", "name of bucket to write to")

func main() {
	flag.Parse()
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt)

	batches := make(chan Batch, 10)
	ctx, cancel := context.WithCancel(context.Background())

	cfg := &aws.Config{
		Region: aws.String(S3Region),
	}
	var (
		s   *session.Session
		err error
	)

	if *creds == "" {
		s, err = session.NewSession(cfg)
	} else {
		s, err = session.NewSessionWithOptions(session.Options{
			Config:            *cfg,
			Profile:           "default",
			SharedConfigFiles: []string{*creds},
		})
	}
	if err != nil {
		log.Fatalf("new session: %s", err)
	}

	go readBatches(ctx, batches)
	go writeBatches(ctx, batches, *bucket, s)

	<-signalC
	cancel()
}
