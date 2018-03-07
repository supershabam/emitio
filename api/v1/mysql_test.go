package v1

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func create(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("mysql", "root:my-secret-pw@tcp(localhost:3306)/")
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(rand.Int63()))
	dbname := "db_" + strings.Replace(base64.RawURLEncoding.EncodeToString(buf), "-", "_", -1)
	_, err = db.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %s`, dbname))
	if err != nil {
		return nil, err
	}
	db, err = sql.Open("mysql", fmt.Sprintf("root:my-secret-pw@tcp(localhost:3306)/%s", dbname))
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE hot(
	service varchar(255),
	at datetime,
	name varchar(255),
	data JSON,
	PRIMARY KEY(service, at, name)
)`)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func populate(ctx context.Context, db *sql.DB, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	type shell struct {
		Time string `json:"time"`
		idx  int
		rest json.RawMessage
		t    time.Time
	}
	batch := make([]shell, 0, 500)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		vals := []interface{}{}
		base := "INSERT INTO hot (service, at, name, data) VALUES "
		for i, b := range batch {
			if i != 0 {
				base = base + ", "
			}
			base = base + "(?,?,?,?)"
			vals = append(vals, []interface{}{"prod-nginx", b.t, fmt.Sprintf("%d", b.idx), b.rest}...)
		}
		_, err := db.ExecContext(ctx, base, vals...)
		if err != nil {
			return err
		}
		batch = make([]shell, 0, 500)
		return nil
	}
	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
		line := scanner.Text()
		var s shell
		err := json.Unmarshal([]byte(line), &s)
		if err != nil {
			return err
		}
		s.rest = json.RawMessage(line)
		// 01/02 03:04:05PM '06 -0700
		// Mon Jan 2 15:04:05 MST 2006
		t, err := time.Parse("02/Jan/2006:15:04:05 -0700", s.Time)
		if err != nil {
			return err
		}
		s.t = t
		s.idx = count
		batch = append(batch, s)
		if len(batch) == cap(batch) {
			err := flush()
			if err != nil {
				return err
			}
		}

	}
	if err := scanner.Err(); err != nil {
		return err
	}
	err = flush()
	if err != nil {
		return err
	}
	return nil
}

func TestMysql(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()
	db, err := create(ctx)
	require.Nil(t, err)
	err = populate(ctx, db, "./testdata/nginx_json_logs.txt")
	require.Nil(t, err)
	m := &MySQL{
		db: db,
	}
	t0 := time.Now()
	err = m.Heatmap(
		ctx,
		"prod-nginx",
		"$.bytes",
		0.5,
		Window{
			Start: time.Now().Add(-time.Hour * 24 * 365 * 5),
			End:   time.Now().Add(time.Minute),
		},
		[]Filter{},
		[]Field{
			{
				JSONPath: "$.response",
			},
		},
		[]float64{
			1,
			2,
			4,
			8,
			16,
			32,
			64,
			128,
			256,
			512,
			1024,
			2048,
			4096,
		},
	)
	require.Nil(t, err)
	fmt.Printf("duration: %s", time.Since(t0))
	assert.True(t, false)
}
