package pyritego

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type PrtPackage struct {
	Session    string
	Identifier string
	sequence   int
	Body       string
}

var (
	ErrInvalidResponse = errors.New("invalid response")
)

func CastToPrtpackage(raw []byte) (*PrtPackage, error) {
	str := string(raw)
	splits := strings.SplitN(str, "\n", 5)
	if len(splits) < 5 {
		return nil, ErrInvalidResponse
	}

	sequence, err := strconv.Atoi(splits[2])
	if err != nil {
		return nil, ErrInvalidResponse
	}

	return &PrtPackage{
		Session:    splits[0],
		Identifier: splits[1],
		sequence:   sequence,
		Body:       splits[4],
	}, nil
}

func (r PrtPackage) ToBytes() []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%d\n\n%s", r.Session, r.Identifier, r.sequence, r.Body))
}
