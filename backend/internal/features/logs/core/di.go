package logs_core

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"logbull/internal/config"
	projects_services "logbull/internal/features/projects/services"
	"logbull/internal/util/logger"
)

var env = config.GetEnv()

var logCoreRepository = &LogCoreRepository{
	client: &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,              // Total idle connections across all hosts
			MaxIdleConnsPerHost: 10,               // Idle connections per host
			MaxConnsPerHost:     50,               // Max connections per host
			IdleConnTimeout:     90 * time.Second, // How long idle connections stay open
			DisableKeepAlives:   false,            // Enable connection reuse
			ForceAttemptHTTP2:   false,            // Stick to HTTP/1.1 for OpenSearch
		},
	},
	baseURL:      strings.TrimRight(fmt.Sprintf("%s:%s", env.OpenSearchURL, env.OpenSearchAPIPort), "/"),
	indexPattern: "logs-*",
	indexPrefix:  "logs-",
	timeout:      5 * time.Minute,
	logger:       logger.GetLogger(),
	queryBuilder: &QueryBuilder{logger.GetLogger()},
}

var logQueryBuilder = &QueryBuilder{
	logger.GetLogger(),
}

var logCoreService = &LogCoreService{
	logCoreRepository,
}

func GetLogCoreRepository() *LogCoreRepository {
	return logCoreRepository
}

func GetUnavailableLogCoreRepository() *LogCoreRepository {
	return &LogCoreRepository{
		client:  &http.Client{},
		baseURL: "http://localhost:8080",
		timeout: 30 * time.Second,
		logger:  logger.GetLogger(),
	}
}

func GetLogQueryBuilder() *QueryBuilder {
	return logQueryBuilder
}

func SetupDependencies() {
	projects_services.GetProjectService().AddProjectDeletionListener(logCoreService)
}
