package remote

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

func Push(url string, req WriteRequest) error {
	body, err := Encode(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/x-minip-remote-write")
	httpReq.Header.Set("Content-Encoding", "snappy")
	cli := &http.Client{Timeout: 8 * time.Second}
	resp, err := cli.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("remote write status %d", resp.StatusCode)
	}
	return nil
}
