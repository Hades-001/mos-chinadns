//     Copyright (C) 2020, IrineSistiana
//
//     This file is part of mos-chinadns.
//
//     mos-chinadns is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     mos-chinadns is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package utils

import (
	"github.com/miekg/dns"
	"sync"
)

var (
	tcpHeaderBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 2)
		},
	}

	tcpWriteBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 512)
		},
	}
)

func getTCPHeaderBuf() []byte {
	return tcpHeaderBufPool.Get().([]byte)
}

func releaseTCPHeaderBuf(buf []byte) {
	tcpHeaderBufPool.Put(buf)
}

// getTCPWriteBuf returns a 2048-byte slice buf
func getTCPWriteBuf() []byte {
	return tcpWriteBufPool.Get().([]byte)
}

func releaseTCPWriteBuf(buf []byte) {
	tcpWriteBufPool.Put(buf)
}

// packMsgWithBuffer packs by bufpool.GetMsgBufFor(m).
// Caller should release the buf using bufpool.ReleaseMsgBuf(buf) after mRaw is no longer used.
func packMsgWithBuffer(m *dns.Msg) (mRaw, buf []byte, err error) {
	buf, err = GetMsgBufFor(m)
	if err != nil {
		return
	}

	mRaw, err = m.PackBuffer(buf)
	if err != nil {
		ReleaseMsgBuf(buf)
		return
	}
	return
}
