package gateway

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type Gateway struct {
	UpstreamURL string
	Client      *http.Client
}

func NewGateway(upstream string) *Gateway {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConnsPerHost = 200
	t.MaxIdleConns = 200
	return &Gateway{
		UpstreamURL: upstream,
		Client: &http.Client{
			Transport: t,
			Timeout:   0, // Streaming needs no global timeout
		},
	}
}

func (g *Gateway) ProxySSE(w http.ResponseWriter, r *http.Request) (overhead time.Duration, upstream time.Duration, err error) {
	t2Start := time.Now()

	req, err := http.NewRequestWithContext(r.Context(), r.Method, g.UpstreamURL, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create request: %v", err), http.StatusServiceUnavailable)
		overhead += time.Since(t2Start)
		return overhead, 0, err
	}
	
	for k, vv := range r.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Accept", "text/event-stream")

	overhead += time.Since(t2Start)
	
	beforeDo := time.Now()
	resp, err := g.Client.Do(req)
	afterDo := time.Now()
	
	upstream += afterDo.Sub(beforeDo)

	t2Start = time.Now()
	if err != nil {
		http.Error(w, fmt.Sprintf("upstream unavailable: %v", err), http.StatusServiceUnavailable)
		overhead += time.Since(t2Start)
		return overhead, upstream, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("upstream returned %d", resp.StatusCode), http.StatusBadGateway)
		overhead += time.Since(t2Start)
		return overhead, upstream, fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		overhead += time.Since(t2Start)
		return overhead, upstream, fmt.Errorf("streaming unsupported")
	}
	overhead += time.Since(t2Start)

	buf := make([]byte, 4096)

	for {
		readStart := time.Now()
		n, readErr := resp.Body.Read(buf)
		upstream += time.Since(readStart)

		t2Start = time.Now()
		if n > 0 {
			_, err = w.Write(buf[:n])
			if err != nil {
				overhead += time.Since(t2Start)
				return overhead, upstream, err
			}
			flusher.Flush()
		}

		if readErr != nil {
			if readErr == io.EOF {
				overhead += time.Since(t2Start)
				break
			}
			overhead += time.Since(t2Start)
			return overhead, upstream, readErr
		}
		overhead += time.Since(t2Start)
	}

	return overhead, upstream, nil
}
