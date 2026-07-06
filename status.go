package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

// mcStatus is the live Minecraft server status served at /gamers/status.
type mcStatus struct {
	Up      bool     `json:"up"`
	Online  int      `json:"online"`
	Max     int      `json:"max"`
	Players []string `json:"players"`
}

// slpResponse is the subset of the Server List Ping JSON we care about.
type slpResponse struct {
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
		} `json:"sample"`
	} `json:"players"`
}

func putVarInt(buf *bytes.Buffer, v int) {
	uv := uint32(v)
	for {
		b := byte(uv & 0x7f)
		uv >>= 7
		if uv != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if uv == 0 {
			return
		}
	}
}

func readVarInt(r io.ByteReader) (int, error) {
	var res uint32
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		res |= uint32(b&0x7f) << shift
		if b&0x80 == 0 {
			return int(res), nil
		}
		shift += 7
		if shift >= 35 {
			return 0, io.ErrUnexpectedEOF
		}
	}
}

func putString(buf *bytes.Buffer, s string) {
	putVarInt(buf, len(s))
	buf.WriteString(s)
}

func writeFramed(conn net.Conn, body []byte) error {
	var frame bytes.Buffer
	putVarInt(&frame, len(body))
	frame.Write(body)
	_, err := conn.Write(frame.Bytes())
	return err
}

// queryMCStatus performs a Minecraft Server List Ping (status handshake) against
// addr ("host:port") and returns the online/max counts plus the player sample.
func queryMCStatus(addr string, timeout time.Duration) (mcStatus, error) {
	var st mcStatus
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return st, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return st, err
	}
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return st, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	// Handshake packet: id 0x00, protocol -1 (status query), host, port, next-state 1.
	var hs bytes.Buffer
	putVarInt(&hs, 0x00)
	putVarInt(&hs, -1)
	putString(&hs, host)
	_ = binary.Write(&hs, binary.BigEndian, uint16(port))
	putVarInt(&hs, 1)
	if err := writeFramed(conn, hs.Bytes()); err != nil {
		return st, err
	}
	// Status request packet: id 0x00, empty.
	var req bytes.Buffer
	putVarInt(&req, 0x00)
	if err := writeFramed(conn, req.Bytes()); err != nil {
		return st, err
	}

	// Response: <len><packetID 0x00><jsonLen><json>.
	r := bufio.NewReader(conn)
	if _, err := readVarInt(r); err != nil { // total frame length
		return st, err
	}
	if _, err := readVarInt(r); err != nil { // packet id
		return st, err
	}
	jsonLen, err := readVarInt(r)
	if err != nil {
		return st, err
	}
	if jsonLen <= 0 || jsonLen > 1<<20 {
		return st, io.ErrUnexpectedEOF
	}
	body := make([]byte, jsonLen)
	if _, err := io.ReadFull(r, body); err != nil {
		return st, err
	}
	var resp slpResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return st, err
	}
	st.Up = true
	st.Online = resp.Players.Online
	st.Max = resp.Players.Max
	for _, p := range resp.Players.Sample {
		if p.Name != "" {
			st.Players = append(st.Players, p.Name)
		}
	}
	return st, nil
}

// statusCache polls the Minecraft server in the background and hands out the last
// result, so page loads never block on (or hammer) the game server.
type statusCache struct {
	mu   sync.RWMutex
	addr string
	val  mcStatus
}

func newStatusCache(addr string) *statusCache {
	c := &statusCache{addr: addr}
	if addr != "" {
		go c.loop()
	}
	return c
}

func (c *statusCache) loop() {
	c.refresh()
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for range t.C {
		c.refresh()
	}
}

func (c *statusCache) refresh() {
	st, err := queryMCStatus(c.addr, 4*time.Second)
	c.mu.Lock()
	if err != nil {
		c.val = mcStatus{Up: false}
	} else {
		c.val = st
	}
	c.mu.Unlock()
}

func (c *statusCache) get() mcStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.val
}
