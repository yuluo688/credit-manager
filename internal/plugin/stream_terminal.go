package plugin

import (
	"bytes"
	"encoding/json"
)

// Host chunks may split an SSE line. Retain a bounded line, never the response
// body. Oversized data lines are skipped; their event header or EOF still ends
// the execution. Mark once so usage-only chunks do not repeatedly write SQLite.
type streamTerminalDetector struct {
	line            []byte
	skipLine        bool
	finished        bool
	expectedChoices int
	finishedChoices map[int]bool
}

func newStreamTerminalDetector(request []byte) streamTerminalDetector {
	var opts struct {
		N int `json:"n"`
	}
	_ = json.Unmarshal(request, &opts)
	if opts.N < 1 {
		opts.N = 1
	}
	return streamTerminalDetector{expectedChoices: opts.N}
}

func (d *streamTerminalDetector) Feed(payload []byte) bool {
	if d.finished {
		return false
	}
	// CLIProxyAPI also returns complete SSE frames with their trailing blank
	// line stripped. Recognize complete data records at the host chunk boundary
	// before trying to join fragments; joining those frames would concatenate
	// the previous JSON body with the next event header.
	for _, line := range bytes.Split(payload, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if bytes.HasPrefix(line, []byte("data:")) && d.terminalSSELine(line) {
			d.finished = true
			d.line = nil
			return true
		}
	}
	for len(payload) > 0 {
		i := bytes.IndexByte(payload, '\n')
		part := payload
		if i >= 0 {
			part = payload[:i]
		}
		if !d.skipLine {
			if len(d.line)+len(part) > 64*1024 {
				d.line = nil
				d.skipLine = true
			} else {
				d.line = append(d.line, part...)
			}
		}
		if i < 0 {
			break
		}
		if !d.skipLine && d.terminalSSELine(bytes.TrimSpace(d.line)) {
			d.finished = true
			d.line = nil
			return true
		}
		d.line = d.line[:0]
		d.skipLine = false
		payload = payload[i+1:]
	}
	return false
}

func (d *streamTerminalDetector) terminalSSELine(line []byte) bool {
	if bytes.HasPrefix(line, []byte("event:")) {
		return terminalEvent(string(bytes.TrimSpace(line[6:])))
	}
	if !bytes.HasPrefix(line, []byte("data:")) {
		return false
	}
	data := bytes.TrimSpace(line[5:])
	if bytes.Equal(data, []byte("[DONE]")) {
		return true
	}
	var event struct {
		Type    string `json:"type"`
		Choices []struct {
			Index        int     `json:"index"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &event) != nil {
		return false
	}
	if terminalEvent(event.Type) {
		return true
	}
	// The host may translate a Chat Completions finish_reason into a Responses
	// completed event before it consumes [DONE]. Release before that translation.
	// For n>1, a single finished choice must not release the other choices.
	for _, c := range event.Choices {
		if c.FinishReason != nil && *c.FinishReason != "" {
			if d.finishedChoices == nil {
				d.finishedChoices = make(map[int]bool)
			}
			d.finishedChoices[c.Index] = true
		}
	}
	want := d.expectedChoices
	if want < 1 {
		want = 1
	}
	return len(d.finishedChoices) >= want
}

func terminalEvent(name string) bool {
	switch name {
	case "response.completed", "response.failed", "response.incomplete", "message_stop", "error":
		return true
	default:
		return false
	}
}
