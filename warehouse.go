package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var (
	warehouseTotalCreditsUsed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "credits_total",
		Subsystem: "warehouse",
		Namespace: "snowflake",
		Help:      "Total credits consumed for the past hour by the particular warehouse",
	}, []string{"warehouse"})

	warehouseCloudCreditsUsed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "credits_cloud",
		Subsystem: "warehouse",
		Namespace: "snowflake",
		Help:      "Total cloud credits consumed by the warehouse in the past hour",
	}, []string{"warehouse"})

	warehouseComputeCreditsUsed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "credits_compute",
		Subsystem: "warehouse",
		Namespace: "snowflake",
		Help:      "Total compute credits used in the last timeframe",
	}, []string{"warehouse"})
)

type warehouseBilling struct {
	Warehouse          string  `db:"WAREHOUSE_NAME"`
	CreditsUsed        float64 `db:"CREDITS_USED"`
	CreditsUsedCompute float64 `db:"CREDITS_USED_COMPUTE"`
	CreditsUsedCloud   float64 `db:"CREDITS_USED_CLOUD_SERVICES"`
}

// Need to specify the list of warehouses to monitor
func gatherWarehouseUsageMetrics(db *sql.DB, start chan time.Time, done chan bool) {
	for rangeStart := range start {
		if !dry {
			query := fmt.Sprintf("select * from table(information_schema.warehouse_metering_history(DATE_RANGE_START => to_timestamp_ltz('%s'), DATE_RANGE_END => current_timestamp()));", rangeStart.Format(time.RFC3339))
			log.Debugf("[WarehouseUsage] Query: %s", query)
			rows, err := runQuery(query, db)
			if err != nil {
				log.Errorf("[WarehouseUsage] Failed to gather warehouse metrics: %+v\n", err)
				done <- true
				continue
			}

			done <- true

			warehouse := &warehouseBilling{}
			for rows.Next() {
				rows.StructScan(warehouse)

				log.Debugf("[WarehouseUsage] row: %+v", warehouse)

				warehouseCloudCreditsUsed.WithLabelValues(warehouse.Warehouse).Add(warehouse.CreditsUsedCloud)
				warehouseComputeCreditsUsed.WithLabelValues(warehouse.Warehouse).Add(warehouse.CreditsUsedCompute)
				warehouseTotalCreditsUsed.WithLabelValues(warehouse.Warehouse).Add(warehouse.CreditsUsed)
			}

			rows.Close()
		} else {
			log.Info("[WarehouseUsage] Skipping query execution due to presence of dry-run flag")
			done <- true
		}
	}
}
