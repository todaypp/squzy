package clickhouse

import (
	"fmt"
	"github.com/ClickHouse/clickhouse-go"
	uuid "github.com/satori/go.uuid"
	apiPb "github.com/squzy/squzy_generated/generated/proto/v1"
	"squzy/internal/logger"
	"time"
)

type Incident struct {
	Model      Model
	IncidentId string
	Status     int32
	RuleId     string
	StartTime  int64
	EndTime    int64
	Histories  []*IncidentHistory
}

type IncidentHistory struct {
	Model      Model
	IncidentID string
	Status     int32
	Timestamp  int64
}

const (
	dbIncidentCollection = "incidents"
	descPrefix           = " DESC"
)

var (
	incidentFields          = "id, created_at, updated_at, incident_id, status, rule_id, start_time, end_time"
	incidentHistoriesFields = "id, created_at, incident_id, status, timestamp"
	incidentIdString        = fmt.Sprintf(`"incident_id" = ?`)
	incidentRuleIdString    = fmt.Sprintf(`"rule_id" = ?`)
	incidentStatusString    = fmt.Sprintf(`"status" = ?`)
	incidentEndTimeString   = fmt.Sprintf(`"end_time" = ?`)
	incidentCreatedAtString = fmt.Sprintf(`"created_at" = ?`)
	incidentUpdatedAtString = fmt.Sprintf(`"updated_at" = ?`)
	incidentDeletedAtString = fmt.Sprintf(`"deleted_at" = ?`)
	incidentHistoriesString = fmt.Sprintf(`"histories.status" = ? , "histories.timestamp" = ?`)

	incidentOrderMap = map[apiPb.SortIncidentList]string{
		apiPb.SortIncidentList_SORT_INCIDENT_LIST_UNSPECIFIED: "startTime",
		apiPb.SortIncidentList_INCIDENT_LIST_BY_START_TIME:    "startTime",
		apiPb.SortIncidentList_INCIDENT_LIST_BY_END_TIME:      "endTime",
	}
)

func (c *Clickhouse) InsertIncident(data *apiPb.Incident) error {
	now := time.Now()

	incident := convertToIncident(data)

	err := c.insertIncident(now, incident)
	if err != nil {
		logger.Error(err.Error())
		return errorDataBase
	}

	for _, history := range incident.Histories {
		err = c.insertIncidentHistory(incident.IncidentId, now, history.Status, history.Timestamp)
		if err != nil {
			logger.Error(err.Error())
			return errorDataBase
		}
	}

	return nil
}

func (c *Clickhouse) insertIncident(now time.Time, incident *Incident) error {
	tx, err := c.Db.Begin()
	if err != nil {
		return err
	}

	q := fmt.Sprintf(`INSERT INTO incidents (%s) VALUES ('$0', '$1', '$2', '$3', '$4', '$5', '$6', '$7')`, incidentFields)
	_, err = tx.Exec(q,
		clickhouse.UUID(uuid.NewV4().String()),
		now,
		now,
		incident.IncidentId,
		incident.Status,
		incident.RuleId,
		incident.StartTime,
		incident.EndTime,
	)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (c *Clickhouse) insertIncidentHistory(incidentId string, now time.Time, status int32, timestamp int64) error {
	tx, err := c.Db.Begin()
	if err != nil {
		return err
	}

	q := fmt.Sprintf(`INSERT INTO incidents_history (%s) VALUES ('$0', '$1', '$2', '$3', '$4')`, incidentHistoriesFields)
	_, err = tx.Exec(q,
		clickhouse.UUID(uuid.NewV4().String()),
		now,
		incidentId,
		status,
		timestamp,
	)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (c *Clickhouse) UpdateIncidentStatus(id string, status apiPb.IncidentStatus) (*apiPb.Incident, error) {
	now := time.Now()
	uNow := now.UnixNano()

	err := c.updateIncident(id, status, now, uNow)
	if err != nil {
		logger.Error(err.Error())
		return nil, errorDataBase
	}

	err = c.insertIncidentHistory(id, now, int32(status), uNow)
	if err != nil {
		logger.Error(err.Error())
		return nil, errorDataBase
	}

	incident, err := c.GetIncidentById(id)
	if err != nil {
		logger.Error(err.Error())
		return nil, errorDataBase
	}
	return incident, nil
}

func (c *Clickhouse) updateIncident(id string, status apiPb.IncidentStatus, tNow time.Time, tUNow int64) error {
	tx, err := c.Db.Begin()
	if err != nil {
		return err
	}

	q := fmt.Sprintf(`ALTER TABLE incidents UPDATE %s , %s , %s WHERE %s`,
		incidentStatusString,
		incidentUpdatedAtString,
		incidentEndTimeString,
		incidentIdString)
	_, err = tx.Exec(q,
		int32(status),
		tNow,
		tUNow,
		id,
	)

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (c *Clickhouse) GetIncidentById(id string) (*apiPb.Incident, error) {
	inc, err := c.getIncident(id)
	if err != nil {
		logger.Error(err.Error())
		return nil, errorDataBase
	}

	histories, err := c.getIncidentHistories(id)
	if err != nil {
		logger.Error(err.Error())
		return nil, errorDataBase
	}

	inc.Histories = histories

	return convertFromIncident(inc), nil
}

func (c *Clickhouse) getIncident(id string) (*Incident, error) {
	rows, err := c.Db.Query(fmt.Sprintf(`SELECT %s FROM incidents WHERE %s LIMIT 1`, incidentFields, incidentIdString), id)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Error(err.Error())
		}
	}()

	inc := &Incident{}

	if ok := rows.Next(); !ok {
		return nil, err
	}

	if err := rows.Scan(&inc.Model.ID, &inc.Model.CreatedAt, &inc.Model.UpdatedAt,
		&inc.IncidentId, &inc.Status, &inc.RuleId, &inc.StartTime, &inc.EndTime); err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	if err := rows.Err(); err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	return inc, nil
}

func (c *Clickhouse) getIncidentHistories(id string) ([]*IncidentHistory, error) {
	var incs []*IncidentHistory

	rows, err := c.Db.Query(fmt.Sprintf(`SELECT %s FROM incidents_history WHERE %s`, incidentHistoriesFields, incidentIdString), id)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Error(err.Error())
		}
	}()

	if next := rows.Next(); next {
		inc := &IncidentHistory{}
		if err := rows.Scan(&inc.Model.ID, &inc.Model.CreatedAt, &inc.IncidentID, &inc.Status, &inc.Timestamp); err != nil {
			logger.Error(err.Error())
			return nil, err
		}
		incs = append(incs, inc)
	}

	if err := rows.Err(); err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	return incs, nil
}

func (c *Clickhouse) GetActiveIncidentByRuleId(ruleId string) (*apiPb.Incident, error) {
	inc, err := c.getActiveIncident(ruleId)
	if err != nil {
		logger.Error(err.Error())
		return nil, errorDataBase
	}

	histories, err := c.getIncidentHistories(inc.IncidentId)
	if err != nil {
		logger.Error(err.Error())
		return nil, errorDataBase
	}

	inc.Histories = histories

	return convertFromIncident(inc), nil
}

func (c *Clickhouse) getActiveIncident(ruleId string) (*Incident, error) {
	inc := &Incident{}

	rows, err := c.Db.Query(fmt.Sprintf(`SELECT %s FROM incidents WHERE (%s) AND (%s OR %s OR %s) LIMIT 1`,
		incidentFields,
		getIncidentRuleString(ruleId),
		getIncidentStatusString(apiPb.IncidentStatus_INCIDENT_STATUS_OPENED),
		getIncidentStatusString(apiPb.IncidentStatus_INCIDENT_STATUS_CAN_BE_CLOSED),
		getIncidentStatusString(apiPb.IncidentStatus_INCIDENT_STATUS_STUDIED),
	))
	if err != nil {
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Error(err.Error())
		}
	}()

	if ok := rows.Next(); !ok {
		return nil, err
	}

	if err := rows.Scan(&inc.Model.ID, &inc.Model.CreatedAt, &inc.Model.UpdatedAt,
		&inc.IncidentId, &inc.Status, &inc.RuleId, &inc.StartTime, &inc.EndTime); err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	if err := rows.Err(); err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	return inc, nil
}

func (c *Clickhouse) GetIncidents(request *apiPb.GetIncidentsListRequest) ([]*apiPb.Incident, int64, error) {
	timeFrom, timeTo, err := getTimeInt64(request.GetTimeRange())
	if err != nil {
		return nil, -1, err
	}

	count, err := c.countIncidents(request, timeFrom, timeTo)
	if err != nil {
		return nil, -1, err
	}

	offset, limit := getOffsetAndLimit(count, request.GetPagination())

	rows, err := c.Db.Query(fmt.Sprintf(`SELECT %s FROM incidents WHERE (%s AND %s AND %s) ORDER BY %s LIMIT %d OFFSET %d`,
		incidentFields,
		incidentStatusString,
		incidentRuleIdString,
		`start_time >= ? AND start_time <= ?`,
		getIncidentOrder(request.GetSort())+getIncidentDirection(request.GetSort()),
		limit,
		offset),
		request.Status,
		request.RuleId,
		timeFrom,
		timeTo,
	)

	if err != nil {
		logger.Error(err.Error())
		return nil, -1, errorDataBase
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Error(err.Error())
		}
	}()

	var incs []*Incident
	for rows.Next() {
		inc := &Incident{}
		if err := rows.Scan(&inc.Model.ID, &inc.Model.CreatedAt, &inc.Model.UpdatedAt,
			&inc.IncidentId, &inc.Status, &inc.RuleId, &inc.StartTime, &inc.EndTime); err != nil {
			logger.Error(err.Error())
			return nil, -1, err
		}

		histories, err := c.getIncidentHistories(inc.IncidentId)
		if err != nil {
			logger.Error(err.Error())
			return nil, -1, err
		}

		inc.Histories = histories
	}

	if err := rows.Err(); err != nil {
		logger.Error(err.Error())
		return nil, -1, errorDataBase
	}

	return convertFromIncidents(incs), count, nil
}

func (c *Clickhouse) countIncidents(request *apiPb.GetIncidentsListRequest, timeFrom int64, timeTo int64) (int64, error) {
	var count int64
	rows, err := c.Db.Query(fmt.Sprintf(`SELECT count(*) FROM incidents WHERE %s AND %s AND %s`,
		incidentStatusString,
		incidentRuleIdString,
		`start_time >= ? AND start_time <= ?`),
		request.Status,
		request.RuleId.Value,
		timeFrom,
		timeTo)

	if err != nil {
		logger.Error(err.Error())
		return -1, errorDataBase
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Error(err.Error())
		}
	}()

	if ok := rows.Next(); !ok {
		return -1, errorDataBase
	}

	if err := rows.Scan(&count); err != nil {
		logger.Error(err.Error())
		return -1, errorDataBase
	}

	if err := rows.Err(); err != nil {
		logger.Error(err.Error())
		return -1, errorDataBase

	}
	return count, nil
}

func getIncidentStatusString(code apiPb.IncidentStatus) string {
	if code == apiPb.IncidentStatus_INCIDENT_STATUS_UNSPECIFIED {
		return ""
	}
	return fmt.Sprintf(`"status" = '%d'`, code)
}

func getIncidentRuleString(ruleId string) string {
	return fmt.Sprintf(`"rule_id" = '%s'`, ruleId)
}

func getIncidentOrder(request *apiPb.SortingIncidentList) string {
	if request == nil {
		return incidentOrderMap[apiPb.SortIncidentList_SORT_INCIDENT_LIST_UNSPECIFIED]
	}
	if res, ok := incidentOrderMap[request.GetSortBy()]; ok {
		return res
	}
	return incidentOrderMap[apiPb.SortIncidentList_SORT_INCIDENT_LIST_UNSPECIFIED]
}

func getIncidentDirection(request *apiPb.SortingIncidentList) string {
	if request == nil {
		return descPrefix
	}
	if res, ok := directionMap[request.GetDirection()]; ok {
		return res
	}
	return descPrefix
}