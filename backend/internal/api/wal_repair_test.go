package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/alkaid/miniprometheus/internal/api"
	"github.com/alkaid/miniprometheus/internal/config"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/remote"
	"github.com/alkaid/miniprometheus/internal/wal"
	"github.com/stretchr/testify/require"
)

const storageHelperEnv = "MINIPROM_STORAGE_HELPER"

func TestStorageProcess(t *testing.T) {
	if os.Getenv(storageHelperEnv) != "1" {
		return
	}
	require.NoError(t, api.RunStorage(config.Load()))
}

func TestWALRepairKeepsSubsequentWritesReplayable(t *testing.T) {
	dir := t.TempDir()
	const firstTimestamp = int64(1_700_000_000_000)
	const secondTimestamp = firstTimestamp + 1_000

	first := startStorage(t, dir)
	writeSample(t, first.baseURL, firstTimestamp, 1)
	first.stop(t)

	segments, err := wal.ListSegments(filepath.Join(dir, "wal"))
	require.NoError(t, err)
	require.Len(t, segments, 1)
	raw, err := os.ReadFile(segments[0])
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	raw[len(raw)-1] ^= 0xff
	require.NoError(t, os.WriteFile(segments[0], raw, 0o600))

	recovered := startStorage(t, dir)
	writeSample(t, recovered.baseURL, secondTimestamp, 2)
	beforeRestart := queryAt(t, recovered.baseURL, secondTimestamp)
	require.Len(t, beforeRestart, 1)
	recovered.stop(t)

	reopened := startStorage(t, dir)
	afterRestart := queryAt(t, reopened.baseURL, secondTimestamp)
	require.Equal(t, beforeRestart, afterRestart)
	reopened.stop(t)
}

type storageProcess struct {
	cmd     *exec.Cmd
	logs    bytes.Buffer
	baseURL string
}

func startStorage(t *testing.T, dir string) *storageProcess {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	exe, err := os.Executable()
	require.NoError(t, err)
	p := &storageProcess{
		cmd:     exec.Command(exe, "-test.run=^TestStorageProcess$", "-test.timeout=15s"),
		baseURL: "http://" + addr,
	}
	p.cmd.Env = append(os.Environ(),
		storageHelperEnv+"=1",
		"MP_ROLE=storage",
		"MP_HTTP_ADDR="+addr,
		"MP_DATA_DIR="+dir,
		"MP_LOG_LEVEL=error",
		"MP_HEAD_BLOCK_MIN=60",
	)
	p.cmd.Stdout = &p.logs
	p.cmd.Stderr = &p.logs
	require.NoError(t, p.cmd.Start())
	t.Cleanup(func() {
		if p.cmd.ProcessState == nil {
			_ = p.cmd.Process.Kill()
			_ = p.cmd.Wait()
		}
	})

	client := &http.Client{Timeout: time.Second}
	require.Eventually(t, func() bool {
		resp, err := client.Get(p.baseURL + "/health")
		if err != nil {
			return false
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 10*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	return p
}

func (p *storageProcess) stop(t *testing.T) {
	t.Helper()
	require.NoError(t, p.cmd.Process.Signal(os.Interrupt))
	require.NoError(t, p.cmd.Wait(), p.logs.String())
}

func writeSample(t *testing.T, baseURL string, timestamp int64, value float64) {
	t.Helper()
	payload := remote.WriteRequest{Series: []remote.TimeSeries{{
		Labels:  model.FromMap("requests_total", map[string]string{"job": "api"}),
		Samples: []model.Sample{{T: timestamp, V: value}},
	}}}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/write", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: time.Second}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

type querySeries struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
}

func queryAt(t *testing.T, baseURL string, timestamp int64) []querySeries {
	t.Helper()
	q := url.Values{
		"query": {"requests_total"},
		"time":  {strconv.FormatInt(timestamp, 10)},
	}
	resp, err := (&http.Client{Timeout: time.Second}).Get(baseURL + "/api/v1/query?" + q.Encode())
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result struct {
		Status string `json:"status"`
		Data   struct {
			Result []querySeries `json:"result"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Equal(t, "success", result.Status)
	return result.Data.Result
}
