package clickhouse

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/ClickHouse/clickhouse-go"
	_ "github.com/ClickHouse/clickhouse-go"
	"github.com/docker/go-connections/nat"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"
	apiPb "github.com/squzy/squzy_generated/generated/proto/v1"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"os"
	"sort"
	"squzy/internal/logger"
	"testing"
	"time"
)

var (
	db, _       = sql.Open("clickhouse", "tcp://user:password@lkl:00/debug=true&clicks?read_timeout=10&write_timeout=10")
	clickhWrong = &Clickhouse{
		db,
	}
	clickh        *Clickhouse
	testContainer testcontainers.Container
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	err := setup(ctx)
	if err != nil {
		logger.Fatalf("could not start test: %s", err)
	}
	code := m.Run()
	err = shutdown(ctx)
	if err != nil {
		logger.Fatalf("could not stop test: %s", err)
	}
	os.Exit(code)
}

func shutdown(ctx context.Context) error {
	err := testContainer.Terminate(ctx)
	if err != nil {
		return err
	}
	return nil
}

func setup(ctx context.Context) error {
	var err error
	req := testcontainers.ContainerRequest{
		Image:        "yandex/clickhouse-server",
		ExposedPorts: []string{"9000/tcp"},
		WaitingFor:   wait.ForListeningPort(nat.Port("9000/tcp")),
	}
	testContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}

	ip, err := testContainer.Host(ctx)
	if err != nil {
		return err
	}
	port, err := testContainer.MappedPort(ctx, "9000")
	if err != nil {
		return err
	}
	db, err = sql.Open("clickhouse", fmt.Sprintf("tcp://%s:%s?debug=true", ip, port.Port()))
	if err != nil {
		return err
	}
	clickh = &Clickhouse{
		db,
	}

	err = clickh.Migrate()
	if err != nil {
		return err
	}
	return nil
}

func TestInsert(t *testing.T) {
	lo := &apiPb.Incident{
		Id:     "insert",
		Status: 0,
		RuleId: "433",
		Histories: []*apiPb.Incident_HistoryItem{&apiPb.Incident_HistoryItem{
			Status: 0,
			Timestamp: &timestamp.Timestamp{
				Seconds: 3324,
				Nanos:   0,
			},
		}},
	}
	err := clickh.InsertIncident(lo)
	if err != nil {
		assert.Fail(t, err.Error())
	}
}

func TestGetIncidentById(t *testing.T) {
	lo := &apiPb.Incident{
		Id:     "select",
		Status: 1,
		RuleId: "433",
		Histories: []*apiPb.Incident_HistoryItem{
			&apiPb.Incident_HistoryItem{
				Status: 0,
				Timestamp: &timestamp.Timestamp{
					Seconds: 3324,
					Nanos:   0,
				},
			},
		},
	}
	err := clickh.InsertIncident(lo)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	inc, err := clickh.GetIncidentById(lo.Id)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	sort.Slice(lo.Histories, func(i, j int) bool {
		return lo.Histories[i].Status < lo.Histories[j].Status
	})
	assert.NotNil(t, inc)
	assert.Equal(t, lo.Id, inc.Id)
	assert.Equal(t, lo.Status, inc.Status)
	assert.Equal(t, lo.RuleId, inc.RuleId)
	assert.Equal(t, lo.Histories[0].Status, inc.Histories[0].Status)
	assert.Equal(t, lo.Histories[0].Timestamp, inc.Histories[0].Timestamp)
	assert.NotNil(t, inc)
}

func TestUpdateIncidentStatus(t *testing.T) {
	lo := &apiPb.Incident{
		Id:     "update",
		Status: 1,
		RuleId: "433",
		Histories: []*apiPb.Incident_HistoryItem{&apiPb.Incident_HistoryItem{
			Status: 0,
			Timestamp: &timestamp.Timestamp{
				Seconds: 3324,
				Nanos:   0,
			},
		}},
	}
	err := clickh.InsertIncident(lo)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	inc, err := clickh.UpdateIncidentStatus(lo.Id, apiPb.IncidentStatus(2))
	if err != nil {
		assert.Fail(t, err.Error())
	}
	assert.NotNil(t, inc)
	assert.Equal(t, lo.Id, inc.Id)
	assert.Equal(t, apiPb.IncidentStatus(2), inc.Status)
	assert.Equal(t, lo.RuleId, inc.RuleId)
	assert.Equal(t, lo.Histories[0].Status, inc.Histories[0].Status)
	assert.Equal(t, lo.Histories[0].Timestamp, inc.Histories[0].Timestamp)
}

func TestGetActiveIncidentByRuleId(t *testing.T) {
	lo := &apiPb.Incident{
		Id:     "active",
		Status: 1,
		RuleId: "some rule",
		Histories: []*apiPb.Incident_HistoryItem{&apiPb.Incident_HistoryItem{
			Status: 0,
			Timestamp: &timestamp.Timestamp{
				Seconds: 3324,
				Nanos:   0,
			},
		}},
	}
	err := clickh.InsertIncident(lo)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	inc, err := clickh.GetActiveIncidentByRuleId(lo.RuleId)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	assert.NotNil(t, inc)
	assert.Equal(t, lo.Id, inc.Id)
	assert.Equal(t, lo.Status, inc.Status)
	assert.Equal(t, lo.RuleId, inc.RuleId)
	assert.Equal(t, lo.Histories[0].Status, inc.Histories[0].Status)
	assert.Equal(t, lo.Histories[0].Timestamp, inc.Histories[0].Timestamp)
}

func TestGetIncidents(t *testing.T) {
	lo := &apiPb.Incident{
		Id:     "incidents",
		Status: 1,
		RuleId: "433",
		Histories: []*apiPb.Incident_HistoryItem{&apiPb.Incident_HistoryItem{
			Status: 0,
			Timestamp: &timestamp.Timestamp{
				Seconds: 3324,
				Nanos:   0,
			},
		}},
	}
	lo2 := &apiPb.Incident{
		Id:     "incidents2",
		Status: 1,
		RuleId: "433",
		Histories: []*apiPb.Incident_HistoryItem{&apiPb.Incident_HistoryItem{
			Status: 0,
			Timestamp: &timestamp.Timestamp{
				Seconds: 3424,
				Nanos:   0,
			},
		}},
	}
	lo3 := &apiPb.Incident{
		Id:     "incidents3",
		Status: 3,
		RuleId: "433",
		Histories: []*apiPb.Incident_HistoryItem{&apiPb.Incident_HistoryItem{
			Status: 0,
			Timestamp: &timestamp.Timestamp{
				Seconds: 3324,
				Nanos:   0,
			},
		}},
	}

	err := clickh.InsertIncident(lo)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = clickh.InsertIncident(lo2)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = clickh.InsertIncident(lo3)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	incs, count, err := clickh.GetIncidents(&apiPb.GetIncidentsListRequest{
		Status:               lo.Status,
		RuleId:               &wrappers.StringValue{Value:lo.RuleId},
		Pagination:           &apiPb.Pagination{
			Page:                 2,
			Limit:                2,
		},
		TimeRange:            &apiPb.TimeFilter{
			From:                 lo.Histories[0].Timestamp,
			To:                   lo2.Histories[0].Timestamp,
		},
		Sort:                 &apiPb.SortingIncidentList{
			SortBy:               0,
			Direction:            0,
		},
	})
	if err != nil {
		assert.Fail(t, err.Error())
	}
	assert.NotNil(t, incs)
	assert.Equal(t, 2, count)
	assert.Equal(t, 1, incs[0].Status)
	assert.Equal(t, lo.RuleId, incs[0].RuleId)
	assert.Equal(t, 1, incs[1].Status)
	assert.Equal(t, lo.RuleId, incs[1].RuleId)

}

func TestClickhouse_Migrate_error(t *testing.T) {
	t.Run("Should: return error", func(t *testing.T) {
		err := clickhWrong.Migrate()
		assert.Error(t, err)
	})
}

type CustomConverter struct{}

func (s CustomConverter) ConvertValue(v interface{}) (driver.Value, error) {
	switch v.(type) {
	case clickhouse.UUID:
		return v.(clickhouse.UUID), nil
	case string:
		return v.(string), nil
	case []uint32:
		return v.([]uint32), nil
	case []int64:
		return v.([]int64), nil
	case int:
		return v.(int), nil
	case int32:
		return v.(int32), nil
	case int64:
		return v.(int64), nil
	case time.Time:
		return v.(time.Time), nil
	default:
		return nil, errors.New(fmt.Sprintf("cannot convert %T with value %v", v, v))
	}
}