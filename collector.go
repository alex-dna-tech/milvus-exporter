package main

import (
	"context"
	"log"
	"strconv"

	"github.com/milvus-io/milvus-proto/go-api/v2/milvuspb"
	"github.com/prometheus/client_golang/prometheus"
)

type milvusMetricFn func(c milvuspb.MilvusServiceClient) []prometheus.Metric

var defaultMilvusMetrics = []milvusMetricFn{
	func(c milvuspb.MilvusServiceClient) []prometheus.Metric {
		var pm []prometheus.Metric
		ctx := context.Background()
		col, err := c.ShowCollections(ctx, &milvuspb.ShowCollectionsRequest{})
		if err != nil {
			log.Fatalf("ShowCollections err: %v\n", err)
		}

		for i := 0; i < len(col.CollectionNames); i++ {

			is, err := c.GetIndexStatistics(
				ctx,
				&milvuspb.GetIndexStatisticsRequest{
					CollectionName: col.CollectionNames[i],
				},
			)
			if err != nil {
				log.Fatalf("DescribeIndex err: %v\n", err)
			}
			for _, d := range is.GetIndexDescriptions() {
				var index_type, metric_type string
				for _, p := range d.Params {
					switch p.Key {
					case "index_type":
						index_type = p.Value
					case "metric_type":
						metric_type = p.Value
					}
				}
				pm = append(
					pm,
					prometheus.MustNewConstMetric(
						prometheus.NewDesc(
							"milvus_index_progress",
							"showing index build progress in percent",
							[]string{"col_name", "col_id", "index_name", "index_type", "metric_type"},
							nil,
						),
						prometheus.GaugeValue,
						float64(d.IndexedRows)/float64(d.TotalRows)*100,
						col.CollectionNames[i],
						strconv.Itoa(int(col.CollectionIds[i])),
						d.IndexName,
						index_type,
						metric_type,
					),
					prometheus.MustNewConstMetric(
						prometheus.NewDesc(
							"milvus_index_indexed_rows",
							"showing amount of indexed rows in collections",
							[]string{"col_name", "col_id", "index_name", "index_type", "metric_type"},
							nil,
						),
						prometheus.GaugeValue,
						float64(d.IndexedRows),
						col.CollectionNames[i],
						strconv.Itoa(int(col.CollectionIds[i])),
						d.IndexName,
						index_type,
						metric_type,
					),
					prometheus.MustNewConstMetric(
						prometheus.NewDesc(
							"milvus_collection_total_rows",
							"showing total rows in collections",
							[]string{"col_name", "col_id"},
							nil,
						),
						prometheus.GaugeValue,
						float64(d.TotalRows),
						col.CollectionNames[i],
						strconv.Itoa(int(col.CollectionIds[i])),
					),
				)
			}
		}

		return pm
	},
	func(c milvuspb.MilvusServiceClient) []prometheus.Metric {
		var pm []prometheus.Metric
		ctx := context.Background()
		col, err := c.ShowCollections(ctx, &milvuspb.ShowCollectionsRequest{})
		if err != nil {
			log.Fatalf("ShowCollections err: %v\n", err)
		}

		for i := 0; i < len(col.CollectionNames); i++ {
			lp, err := c.GetLoadingProgress(
				ctx, &milvuspb.GetLoadingProgressRequest{
					CollectionName: col.CollectionNames[i],
				},
			)
			if err != nil {
				log.Fatalf("GetLoadingProgress err: %v\n", err)
			}
			pm = append(
				pm,
				prometheus.MustNewConstMetric(
					prometheus.NewDesc(
						"milvus_loading_progress",
						"showing data loadind progerss in percent",
						[]string{"col_name", "col_id"},
						nil,
					),
					prometheus.GaugeValue,
					float64(lp.GetProgress()),
					col.CollectionNames[i],
					strconv.Itoa(int(col.CollectionIds[i])),
				),
			)
		}
		return pm
	},
}

type milvusCollector struct {
	client  milvuspb.MilvusServiceClient
	metrics []milvusMetricFn
}

// NewMilvusCollector initializes every descriptor and returns a pointer to the collector
func NewMilvusCollector(c milvuspb.MilvusServiceClient, m ...milvusMetricFn) *milvusCollector {
	if m == nil {
		m = defaultMilvusMetrics
	}
	mc := &milvusCollector{
		client:  c,
		metrics: m,
	}
	return mc
}

// Describe function essentially writes all descriptors to the prometheus desc channel.
func (collector *milvusCollector) Describe(ch chan<- *prometheus.Desc) {}

// Collect implements required collect function for all promehteus collectors
func (collector *milvusCollector) Collect(ch chan<- prometheus.Metric) {
	for _, fn := range collector.metrics {
		for _, pm := range fn(collector.client) {
			ch <- pm
		}
	}
}
