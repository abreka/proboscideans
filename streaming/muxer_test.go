package streaming

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/mattn/go-mastodon"
	"github.com/stretchr/testify/require"
)

func Test_streamPublicSafely(t *testing.T) {
	type testCase struct {
		hits       int
		statusCode int
	}

	currentTest := &testCase{hits: 0, statusCode: http.StatusNotFound}

	// Start an httptest server that returns a 200 OK response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentTest.hits++
		w.WriteHeader(currentTest.statusCode)
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testCases := []testCase{
		{hits: 0, statusCode: http.StatusBadGateway},
		{hits: 0, statusCode: http.StatusNotFound},
		{hits: 0, statusCode: http.StatusServiceUnavailable},
		{hits: 0, statusCode: http.StatusGatewayTimeout},
		{hits: 0, statusCode: http.StatusTooManyRequests},
		{hits: 0, statusCode: http.StatusInternalServerError},
		{hits: 0, statusCode: http.StatusBadRequest},
	}

	for _, testCase := range testCases {
		*currentTest = testCase

		// Create a mastodon client that connects to this test server.
		client := mastodon.NewClient(&mastodon.Config{
			Server:   "http://" + addr,
			ClientID: "client-id",
		})

		// Make two channels to receive events and errors.
		ch := make(chan mastodon.Event)
		errCh := make(chan *StreamError)

		// Start a goroutine that streams public events.
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			defer close(ch)
			defer close(errCh)

			// Only give CI 1 seconds
			ctx, timeout := context.WithTimeout(ctx, time.Second*2)
			defer timeout()
			streamPublicSafely(ctx, "localhost", client, true, ch, errCh)
		}()

		// Consume all events.
		eventsReceived := 0
		go func() {
			defer wg.Done()
			for range ch {
				eventsReceived++
			}
		}()

		// Consume all errors.
		errorsReceived := 0
		go func() {
			defer wg.Done()
			for range errCh {
				errorsReceived++
			}
		}()

		wg.Wait()

		// Not sure why this is 2, but it is for some.
		require.GreaterOrEqual(t, currentTest.hits, 1)
		require.LessOrEqual(t, currentTest.hits, 2)
		require.Equal(t, 0, eventsReceived)
		require.Equal(t, 1, errorsReceived)
	}
}
