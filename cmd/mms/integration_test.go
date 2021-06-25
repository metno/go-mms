// +build integration

package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"encoding/json"
	"github.com/metno/go-mms/pkg/mms"
)

func TestHelpOption(t *testing.T) {
	args := os.Args[0:1]
	args = append(args, "--help")

	output := captureOutput(args, run)
	expected := "USAGE"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected %s; Got %s", expected, output)
	}
}

// TestWithStdOut runs ALL tests that check stdout serially.
// This is done to have predictable captured output from stdout.
func TestWithStdOut(t *testing.T) {
	filteredSubscribe(t)
	subscribeWithCommand(t)
}

func filteredSubscribe(t *testing.T) {
	subscribeArgs := os.Args[0:1]
	subscribeArgs = append(subscribeArgs, "subscribe", "--production-hub", "nats://localhost:4222", "--product", "good")

	go run(subscribeArgs)

	postArgsGood := os.Args[0:1]
	postArgsGood = append(postArgsGood, "post", "--production-hub", "http://localhost:8080", "--product", "good", "--api-key", "97fIjjoKsYxFiJd67EpC1VuZuFPTNUqQv9eTuKEyRXQ=")
	output := captureOutput(postArgsGood, run)

	goodEvent := mms.ProductEvent{}
	err := json.Unmarshal([]byte(output), &goodEvent)
	if err != nil {
		t.Errorf("Expected ok unmarshal; Got error; %s, from output %s", err, output)
		return
	}
	if goodEvent.Product != "good" {
		t.Errorf("Expected event.Product: good; Got %s", goodEvent.Product)
		return
	}

	postArgsBad := os.Args[0:1]
	postArgsBad = append(postArgsBad, "post", "--production-hub", "http://localhost:8080", "--product", "bad", "--api-key", "97fIjjoKsYxFiJd67EpC1VuZuFPTNUqQv9eTuKEyRXQ=")
	output = captureOutput(postArgsBad, run)

	var badEvent mms.ProductEvent
	err = json.Unmarshal([]byte(output), &badEvent)
	if err == nil {
		t.Errorf("Expected empty output from stdout; Got valid json instead: %s", output)
		return
	}
}

// subscribeWithCommand should receive an event like this:
// {"JobName":"vibrations","Product":"good","ProductLocation":"https://best.place.ever","ProductionHub":"http://localhost:8080","Counter":1,"TotalCount":1,"RefTime":"2021-06-25T16:47:05.978454+02:00","CreatedAt":"2021-06-25T16:47:05.978463+02:00","NextEventAt":"2021-06-25T16:47:05.978463+02:00"}
func subscribeWithCommand(t *testing.T) {
	subscribeArgs := os.Args[0:1]
	subscribeArgs = append(subscribeArgs, "subscribe", "--production-hub", "nats://localhost:4222", "--product", "good",
		"--command", "./test_command.sh")

	go run(subscribeArgs)

	postArgsGood := os.Args[0:1]
	postArgsGood = append(postArgsGood, "post", "--production-hub", "http://localhost:8080", "--product", "good",
		"--jobname", "vibrations", "--product-location", "https://best.place.ever", "--api-key", "97fIjjoKsYxFiJd67EpC1VuZuFPTNUqQv9eTuKEyRXQ=")
	output := captureOutput(postArgsGood, run)

	if !strings.Contains(output, "MMS_EVENT={\"JobName\":\"vibrations\",\"Product\":\"good") {
		t.Errorf("Expected command output to include: MMS_EVENT={\"JobName\":\"vibrations\",\"Product\":\"good; Got %s", output)
		return
	}

	if !strings.Contains(output, "product-location=https://best.place.ever") {
		t.Errorf("Expected command output to include: product-location=https://best.place.ever; Got %s", output)
		return
	}
}

// captureOutput captures all output to stdout and stderr after call f with args.
// Waits 100 millseconds and returns a string will all stdout and stderr output.
func captureOutput(args []string, f func([]string) error) string {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
		log.SetOutput(os.Stderr)
	}()
	os.Stdout = writer
	os.Stderr = writer
	log.SetOutput(writer)
	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		var buf bytes.Buffer
		wg.Done()
		io.Copy(&buf, reader)
		out <- buf.String()
	}()
	wg.Wait()
	f(args)
	time.Sleep(100 * time.Millisecond)
	writer.Close()
	return <-out
}
