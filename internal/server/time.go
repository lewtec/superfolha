package server

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

type Time time.Time

func (t Time) MarshalGQL(w io.Writer) {
	tyme := time.Time(t)
	_, _ = w.Write([]byte(strconv.Quote(tyme.Format(time.RFC3339))))
}

func (t *Time) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("Time must be a string")
	}

	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	*t = Time(parsed)
	return nil
}

func MarshalTime(t time.Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = w.Write([]byte(strconv.Quote(t.Format(time.RFC3339))))
	})
}

func UnmarshalTime(v interface{}) (time.Time, error) {
	str, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("Time must be a string")
	}
	return time.Parse(time.RFC3339, str)
}

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
