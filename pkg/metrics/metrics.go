package metrics

import (
	ttlcache "github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
)

// CacheCollector implements a prometheus.Collector for ttlcache.Cache.
type CacheCollector[K comparable, V any] struct {
	cache            *ttlcache.Cache[K, V]
	sizeMetric       *prometheus.Desc
	insertionsMetric *prometheus.Desc
	hitsMetric       *prometheus.Desc
	missesMetric     *prometheus.Desc
	evictionsMetric  *prometheus.Desc
}

// NewCacheCollector creates a new CacheCollector for the specified cache. All
// metrics use the supplied variable and constant labels.
func NewCacheCollector[K comparable, V any](cache *ttlcache.Cache[K, V], prefix string, variableLabels []string, constLabels prometheus.Labels) *CacheCollector[K, V] {
	return &CacheCollector[K, V]{
		cache:            cache,
		sizeMetric:       prometheus.NewDesc(prefix+"cache_size", "Current size of the cache", variableLabels, constLabels),
		insertionsMetric: prometheus.NewDesc(prefix+"cache_insertions", "Number of insertions into the cache", variableLabels, constLabels),
		hitsMetric:       prometheus.NewDesc(prefix+"cache_hits", "Number of cache hits", variableLabels, constLabels),
		missesMetric:     prometheus.NewDesc(prefix+"cache_misses", "Number of cache misses", variableLabels, constLabels),
		evictionsMetric:  prometheus.NewDesc(prefix+"cache_evictions", "Number of cache evictions", variableLabels, constLabels),
	}
}

// Describe implements the Describe method of a prometheus.Collector.
func (collector *CacheCollector[K, V]) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.sizeMetric
	ch <- collector.insertionsMetric
	ch <- collector.hitsMetric
	ch <- collector.missesMetric
	ch <- collector.evictionsMetric
}

// Collect implements the Collect method of a prometheus.Collector.
func (collector *CacheCollector[K, V]) Collect(ch chan<- prometheus.Metric) {
	cache := collector.cache
	metrics := cache.Metrics()
	ch <- prometheus.MustNewConstMetric(collector.sizeMetric, prometheus.GaugeValue, float64(cache.Len()))
	ch <- prometheus.MustNewConstMetric(collector.insertionsMetric, prometheus.GaugeValue, float64(metrics.Insertions))
	ch <- prometheus.MustNewConstMetric(collector.hitsMetric, prometheus.GaugeValue, float64(metrics.Hits))
	ch <- prometheus.MustNewConstMetric(collector.missesMetric, prometheus.GaugeValue, float64(metrics.Misses))
	ch <- prometheus.MustNewConstMetric(collector.evictionsMetric, prometheus.GaugeValue, float64(metrics.Evictions))
}
