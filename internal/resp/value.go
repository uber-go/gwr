// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package resp

import (
	"fmt"
	"strconv"
)

// RedisValue implements a scalar value from the redis protocol (string or int)
type RedisValue struct {
	isNum bool
	num   int
	buf   []byte
}

var NilRedisValue = RedisValue{}

func NewIntRedisValue(num int) RedisValue {
	return RedisValue{
		isNum: true,
		num:   num,
	}
}

func NewBytesRedisValue(buf []byte) RedisValue {
	return RedisValue{
		isNum: false,
		buf:   buf,
	}
}

func NewStringRedisValue(str string) RedisValue {
	buf := []byte(str)
	return RedisValue{
		isNum: false,
		buf:   buf,
	}
}

func (rv RedisValue) IsNumber() bool {
	return rv.isNum
}

func (rv RedisValue) IsNull() bool {
	if rv.isNum {
		return false
	}
	return rv.buf == nil
}

func (rv RedisValue) GetNumber() (int, bool) {
	if !rv.isNum {
		return 0, false
	}
	return rv.num, true
}

func (rv RedisValue) GetBytes() ([]byte, bool) {
	if rv.isNum {
		return nil, false
	}
	if rv.buf == nil {
		return nil, false
	}
	return rv.buf, true
}

func (rv RedisValue) GetString() (string, bool) {
	if rv.isNum {
		return "", false
	}
	if rv.buf == nil {
		return "", false
	}
	return string(rv.buf), true
}

func (rv RedisValue) WriteTo(rconn *RedisConnection) error {
	if rv.isNum {
		return rconn.WriteInteger(rv.num)
	}
	if rv.buf != nil {
		return rconn.WriteBulkBytes(rv.buf)
	}
	return rconn.WriteNull()
}

func (rv RedisValue) String() string {
	if rv.isNum {
		return strconv.Itoa(rv.num)
	} else if rv.buf != nil {
		return fmt.Sprintf("%#v", string(rv.buf))
	} else {
		return "null"
	}
}

type RedisArray []RedisValue

func (ra RedisArray) WriteTo(rconn *RedisConnection) error {
	if err := rconn.WriteArrayHeader(len(ra)); err != nil {
		return err
	}
	for _, val := range ra {
		if err := val.WriteTo(rconn); err != nil {
			return err
		}
	}
	return nil
}
