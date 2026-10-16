package baozimh

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

type fakeCDPServer struct {
	*httptest.Server

	mu         sync.Mutex
	lastNavURL string
}

func (s *fakeCDPServer) navigatedURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.lastNavURL
}

func newFakeCDPServer(t *testing.T, pageHTML string) *fakeCDPServer {
	t.Helper()

	const (
		targetID  = "T1"
		sessionID = "S1"
		frameID   = "F1"
		nodeID    = 2
	)

	serv := &fakeCDPServer{}

	upgrader := websocket.Upgrader{}

	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"webSocketDebuggerUrl": "ws://%s/devtools/browser/FAKE"}`, r.Host)
	})
	mux.HandleFunc("/devtools/browser/FAKE", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var msg struct {
				ID        int64           `json:"id"`
				SessionID string          `json:"sessionId"`
				Method    string          `json:"method"`
				Params    json.RawMessage `json:"params"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				return
			}

			var result any
			switch msg.Method {
			case "Target.createTarget":
				result = map[string]any{"targetId": targetID}
			case "Target.attachToTarget":
				result = map[string]any{"sessionId": sessionID}
			case "Runtime.evaluate":
				result = map[string]any{"result": map[string]any{"className": "Window", "type": "object"}}
			case "Page.navigate":
				result = map[string]any{"frameId": frameID, "loaderId": "L1"}
			case "DOM.getDocument":
				result = map[string]any{"root": map[string]any{
					"nodeId":         1,
					"backendNodeId":  1,
					"nodeType":       9,
					"nodeName":       "#document",
					"localName":      "",
					"nodeValue":      "",
					"childNodeCount": 1,
					"children": []map[string]any{{
						"nodeId":        nodeID,
						"backendNodeId": nodeID,
						"nodeType":      1,
						"nodeName":      "html",
						"localName":     "html",
						"nodeValue":     "",
					}},
				}}
			case "DOM.querySelector":
				result = map[string]any{"nodeId": nodeID}
			case "DOM.resolveNode":
				result = map[string]any{"object": map[string]any{"objectId": "O1", "type": "object"}}
			case "Runtime.callFunctionOn":
				result = map[string]any{"result": map[string]any{"type": "string", "value": pageHTML}}
			default:
				result = map[string]any{}
			}

			resp := map[string]any{"id": msg.ID, "result": result}
			if msg.SessionID != "" {
				resp["sessionId"] = msg.SessionID
			}
			if err := conn.WriteJSON(resp); err != nil {
				return
			}

			if msg.Method == "Page.navigate" {
				var params struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal(msg.Params, &params); err != nil {
					return
				}

				serv.mu.Lock()
				serv.lastNavURL = params.URL
				serv.mu.Unlock()

				sendEvent := func(method string, params any) {
					if err := conn.WriteJSON(map[string]any{
						"method":    method,
						"params":    params,
						"sessionId": sessionID,
					}); err != nil {
						return
					}
				}

				sendEvent("Page.frameNavigated", map[string]any{
					"frame": map[string]any{"id": frameID, "loaderId": "L1", "url": params.URL},
				})
				sendEvent("Runtime.executionContextCreated", map[string]any{
					"context": map[string]any{
						"id":      1,
						"auxData": map[string]any{"frameId": frameID},
					},
				})
				sendEvent("DOM.documentUpdated", map[string]any{})
				sendEvent("Page.navigatedWithinDocument", map[string]any{
					"frameId": frameID,
					"url":     params.URL,
				})
			}
		}
	})

	serv.Server = httptest.NewServer(mux)
	t.Cleanup(serv.Close)

	return serv
}
