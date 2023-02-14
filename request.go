package pyritego

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Request struct {
	Session    string
	Identifier string
	Sequence   int
	Body       string
}

var (
	ErrInvalidRequest = errors.New("invalid request")
)

func CastToRequest(raw []byte) (*Request, error) {
	str := string(raw)
	splits := strings.SplitN(str, "\n", 5)
	if len(splits) < 5 {
		return nil, ErrInvalidRequest
	}

	sequence, err := strconv.Atoi(splits[2])
	if err != nil {
		return nil, ErrInvalidRequest
	}

	return &Request{
		Session:    splits[0],
		Identifier: splits[1],
		Sequence:   sequence,
		Body:       splits[4],
	}, nil
}

// 此函数不检查长度是否超标
func (r Request) ToBytes() []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%d\n\n%s", r.Session, r.Identifier, r.Sequence, r.Body))
}
