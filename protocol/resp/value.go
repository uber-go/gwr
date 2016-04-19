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
