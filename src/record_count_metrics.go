package pbprometheus

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func SetupRecordCountStats(app core.App, reg *prometheus.Registry, config PBPrometheusConfig) error {

	recordStats := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Namespace: config.Namespace,
		Name:      "record_count",
		Help:      "Count of records in collections",
	}, []string{"collection"})

	app.Cron().MustAdd("prometheus_record_stats", "* * * * *", func() {

		collectionNames, error := app.FindAllCollections()
		if error != nil {
			app.Logger().Error(fmt.Sprintf(" Failed to get collections for prometheus record stats: %v", error))
			return
		}
		for _, collection := range collectionNames {

			count, err := app.CountRecords(collection.Name)
			if err != nil {
				app.Logger().Error(fmt.Sprintf("Failed to get record count for collection %s: %v", collection.Name, err))
				continue
			}
			recordStats.WithLabelValues(collection.Name).Set(float64(count))
		}

	})

	return nil

}
