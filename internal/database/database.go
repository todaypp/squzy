package database

import (
	"database/sql"
	"github.com/squzy/squzy/internal/database/clickhouse"
	apiPb "github.com/squzy/squzy_generated/generated/github.com/squzy/squzy_proto"
)

type Database interface {
	InsertSnapshot(data *apiPb.SchedulerResponse) error
	GetSnapshots(request *apiPb.GetSchedulerInformationRequest) ([]*apiPb.SchedulerSnapshot, int32, error)
	GetSnapshotsUptime(request *apiPb.GetSchedulerUptimeRequest) (*apiPb.GetSchedulerUptimeResponse, error)
	InsertStatRequest(data *apiPb.Metric) error
	GetStatRequest(id string, pagination *apiPb.Pagination, filter *apiPb.TimeFilter) ([]*apiPb.GetAgentInformationResponse_Statistic, int32, error)
	GetCPUInfo(id string, pagination *apiPb.Pagination, filter *apiPb.TimeFilter) ([]*apiPb.GetAgentInformationResponse_Statistic, int32, error)
	GetMemoryInfo(id string, pagination *apiPb.Pagination, filter *apiPb.TimeFilter) ([]*apiPb.GetAgentInformationResponse_Statistic, int32, error)
	GetDiskInfo(id string, pagination *apiPb.Pagination, filter *apiPb.TimeFilter) ([]*apiPb.GetAgentInformationResponse_Statistic, int32, error)
	GetNetInfo(id string, pagination *apiPb.Pagination, filter *apiPb.TimeFilter) ([]*apiPb.GetAgentInformationResponse_Statistic, int32, error)
	InsertTransactionInfo(data *apiPb.TransactionInfo) error
	GetTransactionInfo(request *apiPb.GetTransactionsRequest) ([]*apiPb.TransactionInfo, int64, error)
	GetTransactionByID(request *apiPb.GetTransactionByIdRequest) (*apiPb.TransactionInfo, []*apiPb.TransactionInfo, error)
	GetTransactionGroup(request *apiPb.GetTransactionGroupRequest) (map[string]*apiPb.TransactionGroup, error)
	InsertIncident(*apiPb.Incident) error
	GetIncidentById(id string) (*apiPb.Incident, error)
	GetActiveIncidentByRuleId(ruleId string) (*apiPb.Incident, error)
	UpdateIncidentStatus(id string, status apiPb.IncidentStatus) (*apiPb.Incident, error)
	GetIncidents(request *apiPb.GetIncidentsListRequest) ([]*apiPb.Incident, int64, error)
	Migrate() error
}

func New(chDb *sql.DB) Database {
	return &clickhouse.Clickhouse{
		Db: chDb,
	}
}
