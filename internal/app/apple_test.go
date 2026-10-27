package app

import "testing"

func TestAppleAdaptersDoNotMutateOriginalOrShareRequestSession(t *testing.T) {
	original := &Service{requestSession: &requestSession{}}
	copy := original.WithAppleAdapters(nil, nil, nil)
	if copy == original || copy.requestSession != nil || original.requestSession == nil {
		t.Fatal("platform adapter copy changed the original service or retained request state")
	}
}
