package m

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"
)

type column struct {
	Index []int
	Value []interface{}
}

type Memory struct {
	at      []time.Time
	columns map[string]*column
}

func (m *Memory) UnmarshalJSON(b []byte) error {
	m.at = []time.Time{}
	m.columns = map[string]*column{}
	d := json.NewDecoder(bytes.NewReader(b))
	var r map[string]interface{}
	for {
		r = map[string]interface{}{}
		err := d.Decode(&r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		at, ok := r["at"].(float64)
		if !ok {
			return errors.New("expected at field to be numeric")
		}
		t := time.Unix(0, int64(at*1e9))
		m.at = append(m.at, t)
		delete(r, "at")
		for k, v := range r {
			cols := m.columns[k]
			if cols == nil {
				cols = &column{
					Index: []int{},
					Value: []interface{}{},
				}
				m.columns[k] = cols
			}
			cols.Index = append(cols.Index, len(m.at)-1)
			cols.Value = append(cols.Value, v)
		}
	}
	return nil
}
