package klas

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
)

// Clients restored from the same saved session share the upstream course context.
// Stripes bound memory without retaining session credentials. Hash collisions only
// serialize unrelated sessions; they cannot mix their responses.
var courseLocks [256]chan struct{}
var courseLocksOnce sync.Once

func (c *Client) lockCourse(ctx context.Context) (func(), error) {
	courseLocksOnce.Do(func() {
		for i := range courseLocks {
			courseLocks[i] = make(chan struct{}, 1)
		}
	})
	key := c.baseURL.String()
	cookies := map[string]string{}
	if c.jar != nil {
		cookies = c.cookiesMap()
	}
	if session := cookies["SESSION"]; session != "" {
		key += session
	} else if session := cookies["WMONID"]; session != "" {
		key += session
	} else {
		key += fmt.Sprintf("%p", c)
	}
	sum := sha256.Sum256([]byte(key))
	lock := courseLocks[int(sum[0])]
	select {
	case lock <- struct{}{}:
		release, err := lockCourseProcess(ctx, sum)
		if err != nil {
			<-lock
			return nil, err
		}
		return func() { release(); <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
