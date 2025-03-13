package resource_test

import (
	"testing"
	"time"

	// TODO: test all implementations of a resource

	resourceImpl "github.com/ProjectLighthouseCAU/beacon/resource/brokerless" // <- change resource implementation here
)

const (
	maxLatency time.Duration = 1 * time.Millisecond
	expected                 = "test"  // test value
	expected2                = "test2" // different test value
)

func TestGet(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, expected)
	got := testResource.Get()
	if got != expected {
		t.Fatalf("Get expected %s, but got %s", expected, got)
	}
	testResource.Close()
}

func TestPut(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	err := testResource.Put(expected)
	if err != nil { // only StreamSkipped -> should not happen without open streams
		t.Fatalf("Put failed with error: %s", err)
	}
	got := testResource.Get()
	if got != expected {
		t.Fatalf("Get after Put expected %s, but got %s", expected, got)
	}
	testResource.Close()
}

func TestStream(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	stream := testResource.Stream()
	select {
	case v := <-stream:
		t.Fatalf("Stream channel returned %v, but should not return anything", v)
	case <-time.After(maxLatency):
		// pass
	}
	testResource.Close()
}

func TestStopStream(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	stream := testResource.Stream()
	err := testResource.StopStream(stream)
	if err != nil {
		t.Fatalf("StopStream failed: %s", err)
	}
	testResource.Close()
}

func TestLink(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	testResource2 := resourceImpl.Create[any]([]string{}, nil)
	err := testResource.Link(testResource2)
	if err != nil {
		t.Fatalf("Link failed: %s", err)
	}
	testResource.Close()
	testResource2.Close()
}

func TestUnLink(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	testResource2 := resourceImpl.Create[any]([]string{}, nil)
	err := testResource.Link(testResource2)
	if err != nil {
		t.Fatalf("Link failed: %s", err)
	}
	err = testResource.UnLink(testResource2)
	if err != nil {
		t.Fatalf("UnLink failed: %s", err)
	}
	testResource.Close()
	testResource2.Close()
}

func TestPutGet(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	err := testResource.Put(expected)
	if err != nil { // StreamSkipped -> should not happen
		t.Fatalf("Put failed: %s", err)
	}
	time.Sleep(maxLatency)
	got := testResource.Get()
	if got != expected {
		t.Fatalf("Expected %v, got %v", expected, got)
	}
	testResource.Close()
}

func TestStreamPut(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	stream := testResource.Stream()
	err := testResource.Put(expected)
	if err != nil {
		t.Fatalf("Put failed: %s", err)
	}
	select {
	case got := <-stream:
		if got != expected {
			t.Fatalf("Expected %v, got %v", expected, got)
		}
	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	testResource.Close()
}

func TestStreamPutStopStreamPut(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	stream := testResource.Stream()
	err := testResource.Put(expected)
	if err != nil {
		t.Fatalf("Put failed: %s", err)
	}
	select {
	case got := <-stream:
		if got != expected {
			t.Fatalf("Expected %v, got %v", expected, got)
		}
	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	err = testResource.StopStream(stream)
	if err != nil {
		t.Fatalf("StopStream failed: %s", err)
	}
	err = testResource.Put(expected)
	if err != nil {
		t.Fatalf("Put failed: %s", err)
	}
	select {
	case got := <-stream:
		if got != nil { // chan returns zero value when closed
			t.Fatalf("Expected nil, got %v", got)
		}
	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	testResource.Close()
}

func TestLinkPutGet(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	testResource2 := resourceImpl.Create[any]([]string{}, nil)
	err := testResource.Link(testResource2)
	if err != nil {
		t.Fatalf("Link failed: %s", err)
	}
	err = testResource2.Put(expected)
	if err != nil {
		t.Fatalf("Put failed: %s", err)
	}
	time.Sleep(maxLatency)
	got := testResource.Get()

	if got != expected {
		t.Fatalf("Expected %v, got %v", expected, got)
	}
	testResource.Close()
	testResource2.Close()
}

func TestLinkStreamPut(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	testResource2 := resourceImpl.Create[any]([]string{}, nil)
	err := testResource.Link(testResource2)
	if err != nil {
		t.Fatalf("Link failed: %s", err)
	}
	stream := testResource.Stream()
	err = testResource2.Put(expected)
	if err != nil {
		t.Fatalf("Put failed: %s", err)
	}
	select {
	case got := <-stream:
		if got != expected {
			t.Fatalf("Expected %v, got %v", expected, got)
		}
	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	testResource.Close()
	testResource2.Close()
}

func TestLinkUnLinkPutGet(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	testResource2 := resourceImpl.Create[any]([]string{}, nil)
	err := testResource.Link(testResource2)
	if err != nil {
		t.Fatalf("Link failed: %s", err)
	}
	testResource2.Put(expected)
	time.Sleep(maxLatency)
	got := testResource.Get()
	if got != expected {
		t.Fatalf("Expected %v, got %v", expected, got)
	}
	err = testResource.UnLink(testResource2)
	if err != nil {
		t.Fatalf("UnLink failed: %s", err)
	}
	err = testResource2.Put(expected2)
	if err != nil {
		t.Fatalf("Put failed: %s", err)
	}
	got = testResource.Get()
	if got == expected2 || got != expected {
		t.Fatalf("Expected %v, got %v", expected, got)
	}
	testResource.Close()
	testResource2.Close()
}

func TestLinkUnLinkStreamPut(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	testResource2 := resourceImpl.Create[any]([]string{}, nil)
	err := testResource.Link(testResource2)
	if err != nil {
		t.Fatalf("Link failed: %s", err)
	}
	stream := testResource.Stream()
	err = testResource2.Put(expected)
	if err != nil {
		t.Fatalf("Put failed: %s", err)
	}
	select {
	case got := <-stream:
		if got != expected {
			t.Fatalf("Expected %v, got %v", expected, got)
		}
	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	err = testResource.UnLink(testResource2)
	if err != nil {
		t.Fatalf("UnLink failed: %s", err)
	}
	time.Sleep(maxLatency)
	err = testResource2.Put(expected2)
	if err != nil {
		t.Fatalf("Put failed: %s", err)
	}
	select {
	case got := <-stream:
		t.Fatalf("Expected nothing, got %v", got)
	case <-time.After(maxLatency):
		// pass
	}
	testResource.Close()
	testResource2.Close()
}

func TestStopStreamInvalid(t *testing.T) {
	testResource := resourceImpl.Create[any]([]string{}, nil)
	stream := make(chan any)
	err := testResource.StopStream(stream)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	testResource.Close()
}
