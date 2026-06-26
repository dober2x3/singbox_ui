package clashapi

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/coder/websocket"
)

func writeSSE(c *gin.Context, data []byte) {
	_, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	if err != nil {
		return
	}
	c.Writer.Flush()
}

func ProxyWS(c *gin.Context, clashPort int, wsPath string) {
	ctx := c.Request.Context()

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d%s", clashPort, wsPath)

	wsConn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		c.JSON(502, gin.H{"error": fmt.Sprintf("clash ws dial: %v", err)})
		return
	}
	defer wsConn.Close(websocket.StatusInternalError, "closing")

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(200)

	for {
		_, msg, err := wsConn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil || err == io.EOF {
				return
			}
			log.Printf("clash ws read error: %v", err)
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
			writeSSE(c, msg)
		}
	}
}

func ProxyWSBinary(c *gin.Context, clashPort int, wsPath string) {
	ctx := c.Request.Context()

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d%s", clashPort, wsPath)

	wsConn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		c.JSON(502, gin.H{"error": fmt.Sprintf("clash ws dial: %v", err)})
		return
	}
	defer wsConn.Close(websocket.StatusInternalError, "closing")

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(200)

	for {
		_, msg, err := wsConn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil || err == io.EOF {
				return
			}
			log.Printf("clash ws read error: %v", err)
			return
		}

		var t TrafficMessage
		if err := t.UnmarshalBinary(msg); err != nil {
			log.Printf("traffic unmarshal error: %v", err)
			continue
		}
		jsonData := fmt.Sprintf(`{"up":%d,"down":%d}`, t.Up, t.Down)

		select {
		case <-ctx.Done():
			return
		default:
			writeSSE(c, []byte(jsonData))
		}
	}
}

func (c *Client) ProxyTrafficSSE(ginCtx *gin.Context) {
	ProxyWSBinary(ginCtx, c.portFromURL(), "/traffic")
}

func (c *Client) ProxyMemorySSE(ginCtx *gin.Context) {
	ProxyWS(ginCtx, c.portFromURL(), "/memory")
}

func (c *Client) ProxyLogsSSE(ginCtx *gin.Context) {
	ProxyWS(ginCtx, c.portFromURL(), "/logs")
}

func (c *Client) portFromURL() int {
	port := 9090
	if idx := strings.LastIndex(c.baseURL, ":"); idx >= 0 {
		if p, err := strconv.Atoi(c.baseURL[idx+1:]); err == nil {
			port = p
		}
	}
	return port
}
