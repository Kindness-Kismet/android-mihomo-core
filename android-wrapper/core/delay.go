//go:build android && cgo

package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/metacubex/mihomo/common/utils"
	"github.com/metacubex/mihomo/constant"
)

type TestDelayParams struct {
	ProxyName string `json:"proxy-name"`
	TestURL   string `json:"test-url"`
	Timeout   int64  `json:"timeout"`
}

type Delay struct {
	Url   string `json:"url"`
	Name  string `json:"name"`
	Value int32  `json:"value"`
}

// handleAsyncTestDelay runs a URL test for the specified proxy and returns Delay JSON (Value=-1 on failure).
func handleAsyncTestDelay(paramsString string) string {
	var params TestDelayParams
	if err := json.Unmarshal([]byte(paramsString), &params); err != nil {
		return ""
	}

	testURL := params.TestURL
	if testURL == "" {
		testURL = constant.DefaultTestURL
	}

	timeoutMs := params.Timeout
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	delayData := &Delay{
		Name: params.ProxyName,
		Url:  testURL,
	}

	expectedStatus, err := utils.NewUnsignedRanges[uint16]("")
	if err != nil {
		delayData.Value = -1
		data, _ := json.Marshal(delayData)
		return string(data)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	coreMu.Lock()
	proxy := allProxies()[params.ProxyName]
	coreMu.Unlock()

	if proxy == nil {
		delayData.Value = -1
		data, _ := json.Marshal(delayData)
		return string(data)
	}

	delay, err := proxy.URLTest(ctx, testURL, expectedStatus)
	if err != nil || delay == 0 {
		delayData.Value = -1
	} else {
		delayData.Value = int32(delay)
	}

	data, _ := json.Marshal(delayData)
	return string(data)
}
