package pyritego

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Response struct {
	Session    string
	Identifier string
	Sequence   int
	Body       string
}

var (
	ErrInvalidResponse = errors.New("invalid response")
)

func CastToResponse(raw []byte) (*Response, error) {
	str := string(raw)
	splits := strings.SplitN(str, "\n", 5)
	if len(splits) < 5 {
		return nil, ErrInvalidResponse
	}

	sequence, err := strconv.Atoi(splits[2])
	if err != nil {
		return nil, ErrInvalidResponse
	}

	return &Response{
		Session:    splits[0],
		Identifier: splits[1],
		Sequence:   sequence,
		Body:       splits[4],
	}, nil
}

func (r Response) ToBytes() []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%d\n\n%s", r.Session, r.Identifier, r.Sequence, r.Body))
}
