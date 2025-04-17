// Copyright 2025 AUTHORS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package websocketutil

import (
	"context"
	"io"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type ByteHandler func([]byte) bool

type ConnReadWriter struct {
	DoneCh       chan error
	doneOnce     sync.Once
	ctx          context.Context
	conn         *websocket.Conn
	cancel       context.CancelFunc
	readCh       chan []byte
	handlerMu    sync.Mutex
	readHandler  ByteHandler
	writeHandler ByteHandler
}

func NewConnReadWriteCloser(ctx context.Context, conn *websocket.Conn) *ConnReadWriter {
	ctx, cancel := context.WithCancel(ctx)
	readWriter := &ConnReadWriter{
		ctx:    ctx,
		conn:   conn,
		cancel: cancel,
		DoneCh: make(chan error, 2),
		readCh: make(chan []byte, 16),
	}
	go readWriter.handleReads()
	return readWriter
}

func (rw *ConnReadWriter) SetReadHandler(handler ByteHandler) {
	rw.handlerMu.Lock()
	defer rw.handlerMu.Unlock()
	rw.readHandler = handler
}

func (rw *ConnReadWriter) SetWriteHandler(handler ByteHandler) {
	rw.handlerMu.Lock()
	defer rw.handlerMu.Unlock()
	rw.writeHandler = handler
}

func (rw *ConnReadWriter) Close() error {
	err := rw.conn.Close()
	close(rw.readCh)
	return err
}

func (rw *ConnReadWriter) Write(data []byte) (n int, err error) {
	select {
	case <-rw.ctx.Done():
		return 0, rw.ctx.Err()
	default:
	}

	if rw.writeHandler != nil && rw.writeHandler(data) {
		return len(data), nil
	}
	if err := rw.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		rw.doneOnce.Do(func() {
			select {
			case rw.DoneCh <- err:
			default:
			}
			close(rw.DoneCh)
		})
		return 0, err
	}
	return len(data), nil
}

func (rw *ConnReadWriter) Read(dst []byte) (n int, err error) {
	select {
	case <-rw.ctx.Done():
		log.Print("Early done, sending EOF")
		return 0, io.EOF
	default:
	}
	select {
	case <-rw.ctx.Done():
		return 0, io.EOF
	case bs := <-rw.readCh:
		if len(dst) < len(bs) {
			return 0, io.ErrShortBuffer
		}
		return copy(dst, bs), nil
	}
}

func (rw *ConnReadWriter) handleReads() {
	defer rw.cancel()
	for {
		_, data, err := rw.conn.ReadMessage()
		if err != nil {
			rw.doneOnce.Do(func() {
				select {
				case rw.DoneCh <- err:
				default:
				}
				close(rw.DoneCh)
			})
			return
		}
		if rw.readHandler != nil && rw.readHandler(data) {
			continue
		}
		select {
		case rw.readCh <- data:
		case <-rw.ctx.Done():
		}
	}
}
