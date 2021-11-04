package resource_test

import (
	"net/http"
	"testing"
	"time"

	resource "lighthouse.uni-kiel.de/lighthouse-server/resource/broker" // <- change resource implementation here
)

const maxLatency time.Duration = 1 * time.Millisecond

func TestGet(t *testing.T) {
	testResource := resource.Create([]string{})
	v, resp := testResource.Get()
	if resp.Err != nil {
		t.Fatalf("Get failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	if v != nil {
		t.Fatalf("Expected nil, got %v", v)
	}
	testResource.Close()
}

func TestPut(t *testing.T) {
	testResource := resource.Create([]string{})
	resp := testResource.Put("test")
	if resp.Err != nil {
		t.Fatalf("Put failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	testResource.Close()
}

func TestStream(t *testing.T) {
	testResource := resource.Create([]string{})
	stream, resp := testResource.Stream()
	if resp.Err != nil {
		t.Fatalf("Stream failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	select {
	case v := <-stream:
		t.Fatalf("Stream channel returned %v, but should not return anything", v)
	case <-time.After(maxLatency):
		// pass
	}
	testResource.Close()
}

func TestStopStream(t *testing.T) {
	testResource := resource.Create([]string{})
	stream, resp := testResource.Stream()
	if resp.Err != nil {
		t.Fatalf("Stream failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	resp = testResource.StopStream(stream)
	if resp.Err != nil {
		t.Fatalf("StopStream failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	testResource.Close()
}

func TestLink(t *testing.T) {
	testResource := resource.Create([]string{})
	testResource2 := resource.Create([]string{})
	resp := testResource.Link(testResource2)
	if resp.Err != nil {
		t.Fatalf("Link failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	testResource.Close()
	testResource2.Close()
}

func TestUnLink(t *testing.T) {
	testResource := resource.Create([]string{})
	testResource2 := resource.Create([]string{})
	resp := testResource.Link(testResource2)
	if resp.Err != nil {
		t.Fatalf("Link failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	resp = testResource.UnLink(testResource2)
	if resp.Err != nil {
		t.Fatalf("UnLink failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	testResource.Close()
	testResource2.Close()
}

func TestPutGet(t *testing.T) {
	testResource := resource.Create([]string{})
	s1 := "test"
	testResource.Put(s1)
	time.Sleep(maxLatency)
	s2, resp := testResource.Get()
	if resp.Err != nil {
		t.Fatalf("Get failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	if s1 != s2 {
		t.Fatalf("Expected %v, got %v", s1, s2)
	}
	testResource.Close()
}

func TestStreamPut(t *testing.T) {
	testResource := resource.Create([]string{})
	stream, resp := testResource.Stream()
	if resp.Err != nil {
		t.Fatalf("Stream failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	s1 := "test"
	resp = testResource.Put(s1)
	if resp.Err != nil {
		t.Fatalf("Put failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	select {
	case s2 := <-stream:
		if s1 != s2 {
			t.Fatalf("Expected %v, got %v", s1, s2)
		}
	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	testResource.Close()
}

func TestStreamPutStopStreamPut(t *testing.T) {
	testResource := resource.Create([]string{})
	stream, resp := testResource.Stream()
	if resp.Err != nil {
		t.Fatalf("Stream failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	s1 := "test"
	resp = testResource.Put(s1)
	if resp.Err != nil {
		t.Fatalf("Put failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	select {
	case s2 := <-stream:
		if s1 != s2 {
			t.Fatalf("Expected %v, got %v", s1, s2)
		}
	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	resp = testResource.StopStream(stream)
	if resp.Err != nil {
		t.Fatalf("StopStream failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	resp = testResource.Put(s1)
	if resp.Err != nil {
		t.Fatalf("Put failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	select {
	case s2 := <-stream:
		if s2 != nil {
			t.Fatalf("Expected nil, got %v", s2)
		}
	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	testResource.Close()
}

func TestLinkPutGet(t *testing.T) {
	testResource := resource.Create([]string{})
	testResource2 := resource.Create([]string{})
	resp := testResource.Link(testResource2)
	if resp.Err != nil {
		t.Fatalf("Linking failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	s1 := "test"
	resp = testResource2.Put(s1)
	if resp.Err != nil {
		t.Fatalf("Put failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	time.Sleep(maxLatency)
	s2, resp := testResource.Get()
	if resp.Err != nil {
		t.Fatalf("Get failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	if s1 != s2 {
		t.Fatalf("Expected %v, got %v", s1, s2)
	}
	testResource.Close()
	testResource2.Close()
}

func TestLinkStreamPut(t *testing.T) {
	testResource := resource.Create([]string{})
	testResource2 := resource.Create([]string{})
	resp := testResource.Link(testResource2)
	if resp.Err != nil {
		t.Fatalf("Link failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	s1 := "test"
	stream, resp := testResource.Stream()
	if resp.Err != nil {
		t.Fatalf("Stream failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	resp = testResource2.Put(s1)
	if resp.Err != nil {
		t.Fatalf("Put failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	select {
	case s2 := <-stream:
		if s1 != s2 {
			t.Fatalf("Expected %v, got %v", s1, s2)
		}

	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	testResource.Close()
	testResource2.Close()
}

func TestLinkUnLinkPutGet(t *testing.T) {
	testResource := resource.Create([]string{})
	testResource2 := resource.Create([]string{})
	resp := testResource.Link(testResource2)
	if resp.Err != nil {
		t.Fatalf("Link failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	s1 := "test"
	testResource2.Put(s1)
	time.Sleep(maxLatency)
	s2, resp := testResource.Get()
	if resp.Err != nil {
		t.Fatalf("Get failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	if s1 != s2 {
		t.Fatalf("Expected %v, got %v", s1, s2)
	}

	resp = testResource.UnLink(testResource2)
	if resp.Err != nil {
		t.Fatalf("UnLink failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	s3 := "test2"
	resp = testResource2.Put(s3)
	if resp.Err != nil {
		t.Fatalf("Put failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	s2, resp = testResource.Get()
	if resp.Err != nil {
		t.Fatalf("Get failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	if s2 == s3 || s2 != s1 {
		t.Fatalf("Expected %v, got %v", s1, s2)
	}
	testResource.Close()
	testResource2.Close()
}

func TestLinkUnLinkStreamPut(t *testing.T) {
	testResource := resource.Create([]string{})
	testResource2 := resource.Create([]string{})
	resp := testResource.Link(testResource2)
	if resp.Err != nil {
		t.Fatalf("Link failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	s1 := "test"
	stream, resp := testResource.Stream()
	if resp.Err != nil {
		t.Fatalf("Stream failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	resp = testResource2.Put(s1)
	if resp.Err != nil {
		t.Fatalf("Put failed with code %d: %s", resp.Code, resp.Err.Error())
	}

	select {
	case s2 := <-stream:
		if s1 != s2 {
			t.Fatalf("Expected %v, got %v", s1, s2)
		}

	case <-time.After(maxLatency):
		t.Fatalf("Timeout after %v", maxLatency)
	}
	resp = testResource.UnLink(testResource2)
	if resp.Err != nil {
		t.Fatalf("UnLink failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	time.Sleep(maxLatency)
	s3 := "test2"
	resp = testResource2.Put(s3)
	if resp.Err != nil {
		t.Fatalf("Put failed with code %d: %s", resp.Code, resp.Err.Error())
	}
	select {
	case s2 := <-stream:
		t.Fatalf("Expected nothing, got %v", s2)
	case <-time.After(maxLatency):
		// pass
	}
	testResource.Close()
	testResource2.Close()
}

func TestStopStreamInvalid(t *testing.T) {
	testResource := resource.Create([]string{})
	stream := make(chan interface{})
	resp := testResource.StopStream(stream)
	if resp.Err == nil {
		t.Fatalf("Expected error, got nil")
	}
	if resp.Code != http.StatusNotFound {
		t.Fatalf("Expected %d, got %d", http.StatusNotFound, resp.Code)
	}
	testResource.Close()
}
