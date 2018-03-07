package v1

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/rand"
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

func TestMysql(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()
	db, err := create(ctx)
	require.Nil(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO hot (service, at, name, data) VALUES (?, ?, ?, ?)`, "prod-nginx", time.Now(), "syslog+udp://localhost:9001:1", `{"response_time":0.02, "response_code": 200}`)
	require.Nil(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO hot (service, at, name, data) VALUES (?, ?, ?, ?)`, "prod-nginx", time.Now(), "syslog+udp://localhost:9001:2", `{"response_time":0.011, "response_code": 202}`)
	require.Nil(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO hot (service, at, name, data) VALUES (?, ?, ?, ?)`, "prod-nginx", time.Now(), "syslog+udp://localhost:9001:3", `{"response_time":0.07, "response_code": 200}`)
	require.Nil(t, err)
	m := &MySQL{
		db: db,
	}
	err = m.Heatmap(
		ctx,
		"prod-nginx",
		"$.response_time",
		0.5,
		Window{
			Start: time.Now().Add(-time.Minute * 5),
			End:   time.Now().Add(time.Minute),
		},
		[]Filter{},
		[]Field{
			{
				JSONPath: "$.response_code",
			},
		},
		[]float64{
			0.01,
			0.02,
			0.04,
			0.08,
		},
	)
	require.Nil(t, err)
	assert.True(t, false)
}
