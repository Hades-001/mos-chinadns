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

package domainlist

import (
	"bufio"
	"fmt"
	"github.com/IrineSistiana/mos-chinadns/dispatcher/logger"
	"io"
	"os"
	"strings"

	"github.com/miekg/dns"
)

func NewListFormFile(file string, continueOnInvalidString bool) (*List, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return NewListFormReader(f, continueOnInvalidString)
}

func NewListFormReader(r io.Reader, continueOnInvalidString bool) (*List, error) {
	l := New()

	lineCounter := 0
	s := bufio.NewScanner(r)
	for s.Scan() {
		lineCounter++
		line := strings.TrimSpace(s.Text())

		//ignore lines begin with # and empty lines
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		fqdn := dns.Fqdn(line)
		if _, ok := dns.IsDomainName(fqdn); !ok {
			if continueOnInvalidString {
				logger.GetStd().Warnf("NewListFormReader: invalid domain [%s] at line %d", line, lineCounter)
			} else {
				return nil, fmt.Errorf("invalid domain [%s] at line %d", line, lineCounter)
			}
		}
		l.Add(fqdn)

	}

	return l, nil
}
