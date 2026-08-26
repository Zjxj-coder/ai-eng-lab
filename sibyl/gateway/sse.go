package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func HandleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	// 模拟结论事件与图表事件的生成过程
	go generateEvents(ctx, w, flusher)

	// 等待客户端断开连接或正常结束
	<-ctx.Done()
}

func generateEvents(ctx context.Context, w http.ResponseWriter, flusher http.Flusher) {
	events := []struct {
		Type string
		Data interface{}
	}{
		{
			Type: "conclusion",
			Data: "Based on the data, DAU increased by 5%.",
		},
		{
			Type: "chart",
			Data: NewDataPoint(
				map[string]interface{}{"date": "2023-01-01", "dau": 1000},
				"SELECT dt, COUNT(DISTINCT user_id) as dau FROM fact_login GROUP BY dt",
				"DAU: Daily Active Users",
			),
		},
	}

	for _, ev := range events {
		select {
		case <-ctx.Done():
			// 客户端已断开
			return
		case <-time.After(10 * time.Millisecond): // 模拟延迟
			dataBytes, _ := json.Marshal(ev.Data)
			fmt.Fprintf(w, "event: %s\n", ev.Type)
			fmt.Fprintf(w, "data: %s\n\n", string(dataBytes))
			flusher.Flush()
		}
	}
}
