package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	CertDir        string
	CertFiles      []string
	ScrapeInterval time.Duration
	ListenAddress  string
	NodeName       string
}

var (
	certDaysRemaining = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "microk8s_cert_days_remaining",
			Help: "Days remaining before certificate expiration",
		},
		[]string{"node", "cert"},
	)

	certNotAfter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "microk8s_cert_not_after_timestamp",
			Help: "Certificate expiration unix timestamp",
		},
		[]string{"node", "cert"},
	)

	certExpired = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "microk8s_cert_expired",
			Help: "1 if certificate expired, otherwise 0",
		},
		[]string{"node", "cert"},
	)

	lastScrapeSuccess = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "microk8s_cert_exporter_last_scrape_success",
			Help: "1 if last certificate scan succeeded",
		},
	)

	certsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "microk8s_cert_exporter_certs_total",
			Help: "Number of configured certificates",
		},
	)

	certsFailed = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "microk8s_cert_exporter_certs_failed",
			Help: "Number of certificates that failed to load",
		},
	)
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadConfig() Config {

	interval, err := time.ParseDuration(
		getEnv("SCRAPE_INTERVAL", "5m"),
	)
	if err != nil {
		log.Printf(
			"invalid SCRAPE_INTERVAL, using default 5m: %v",
			err,
		)
		interval = 5 * time.Minute
	}

	files := []string{}
	for _, file := range strings.Split(
		getEnv(
			"CERT_FILES",
			"server.crt,front-proxy-client.crt",
		),
		",",
	) {

		file = strings.TrimSpace(file)

		if file != "" {
			files = append(files, file)
		}
	}

	return Config{
		CertDir:        getEnv("CERT_DIR", "/host-certs"),
		CertFiles:      files,
		ScrapeInterval: interval,
		ListenAddress:  getEnv("LISTEN_ADDRESS", ":9101"),
		NodeName:       getEnv("NODE_NAME", "unknown"),
	}
}

func loadCertificate(path string) (*x509.Certificate, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, x509.CertificateInvalidError{}
	}

	return x509.ParseCertificate(block.Bytes)
}
func scanCertificates(cfg Config) error {

	now := time.Now()

	total := 0
	failed := 0

	for _, certFile := range cfg.CertFiles {

		certFile = strings.TrimSpace(certFile)

		if certFile == "" {
			continue
		}

		total++

		path := filepath.Join(
			cfg.CertDir,
			certFile,
		)

		cert, err := loadCertificate(path)
		if err != nil {

			failed++

			log.Printf(
				"failed loading certificate %s: %v",
				certFile,
				err,
			)

			continue
		}

		certName := strings.TrimSuffix(
			filepath.Base(certFile),
			".crt",
		)

		daysRemaining :=
			cert.NotAfter.Sub(now).Hours() / 24

		expired := 0.0

		if cert.NotAfter.Before(now) {
			expired = 1
		}

		certDaysRemaining.
			WithLabelValues(
				cfg.NodeName,
				certName,
			).
			Set(daysRemaining)

		certNotAfter.
			WithLabelValues(
				cfg.NodeName,
				certName,
			).
			Set(float64(cert.NotAfter.Unix()))

		certExpired.
			WithLabelValues(
				cfg.NodeName,
				certName,
			).
			Set(expired)
	}

	certsTotal.Set(float64(total))
	certsFailed.Set(float64(failed))

	if failed > 0 {
		return fmt.Errorf(
			"%d of %d certificates failed to load",
			failed,
			total,
		)
	}

	return nil
}

func refreshLoop(cfg Config) {

	for {

		if err := scanCertificates(cfg); err != nil {
			log.Printf(
				"certificate scan failed: %v",
				err,
			)

			lastScrapeSuccess.Set(0)

		} else {

			lastScrapeSuccess.Set(1)
		}

		time.Sleep(cfg.ScrapeInterval)
	}
}
func main() {

	cfg := loadConfig()

	prometheus.MustRegister(
		certDaysRemaining,
		certNotAfter,
		certExpired,
		lastScrapeSuccess,
		certsTotal,
		certsFailed,
	)

	if err := scanCertificates(cfg); err != nil {

		log.Printf(
			"initial certificate scan failed: %v",
			err,
		)

		lastScrapeSuccess.Set(0)

	} else {

		lastScrapeSuccess.Set(1)
	}

	go refreshLoop(cfg)

	http.Handle(
		"/metrics",
		promhttp.Handler(),
	)

	log.Printf(
		"starting microk8s-cert-exporter on %s",
		cfg.ListenAddress,
	)

	log.Printf(
		"monitoring certificates: %s",
		strings.Join(
			cfg.CertFiles,
			", ",
		),
	)

	log.Fatal(
		http.ListenAndServe(
			cfg.ListenAddress,
			nil,
		),
	)
}
