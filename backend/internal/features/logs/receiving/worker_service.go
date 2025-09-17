package logs_receiving

import (
	"context"
	"encoding/json"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"logbull/internal/config"
	logs_core "logbull/internal/features/logs/core"
	cache_utils "logbull/internal/util/cache"

	"github.com/google/uuid"
)

// LogWorkerService provides high-performance log processing optimized from 10 RPS to 30k RPS capacity.
//
// PERFORMANCE CHARACTERISTICS:
// - Ready for 10 RPS in any scenario (mainly in single node scenario)
// - Designed for 10k RPS on very performant VPS in single node scenario
// - Designed for 30k RPS on multiple API nodes and single worker node scenario
// - Batch-only operations with Valkey pipelines for maximum throughput
// - CPU-adaptive configuration optimized for reliability and performance
// - Direct processing from Valkey to log storage without worker buffers
//
// ARCHITECTURE:
// - Multi-worker direct processing with CPU-based worker pool for log storage (25% of CPU cores, min 1)
// - Sharded accumulation buffers for incoming logs (write TO Valkey)
// - CPU-based dedicated flush workers processing shards to Valkey in parallel (25% of CPU cores, min 1)
// - Direct processing from Valkey to log storage (no worker buffers)
// - Separate background workers for maintenance (quotas, retention, stats)
//
// LOAD HANDLING:
// - Queue capacity: unlimited (Valkey-based distributed queue)
// - Batch-only operations: All logs processed in batches for maximum efficiency
// - Fixed batch size: 1,000 logs per batch for dequeue from Valkey
// - Sharded accumulation: CPU-based parallel flush workers eliminate single-point bottleneck
// - Worker pool: CPU-based workers processing directly to log storage
//
// MULTI-INSTANCE DEPLOYMENT:
// This service is ready for writing logs from many application instances via the shared Valkey queue,
// but StartWorkers should only be called on ONE instance to avoid duplicate log processing.
// Other instances can use QueueLog() to add logs to the distributed queue. It is possible that
// both API and worker will be on one very performant VPS. It is possible that API will be on many VPS
// and worker on single node (always single node).
type LogWorkerService struct {
	logRepository *logs_core.LogCoreRepository
	queueService  *cache_utils.ValkeyQueueService
	logger        *slog.Logger

	// Worker control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Sharded accumulation buffers to reduce mutex contention at high RPS.
	// Each shard has its own buffer and mutex, processed by dedicated flush workers.
	accumulatedLogShards [][]*logs_core.LogItem
	accumulationMutexes  []sync.RWMutex
	flushTickers         []*time.Ticker
}

const (
	batchProcessingInterval = 1 * time.Second // Base processing interval optimized for 10 RPS expected load

	// Fixed settings optimized for 10k RPS capacity
	cacheToLogsStorageWritingBatchSize = 1_000 // Fixed batch size for dequeuing from Valkey

	logQueueKey = "logbull:logs:queue" // Valkey queue key for log items

	// Internal accumulation settings - sharded for high RPS
	ramToValkeyQueueAccumulationFlushInterval = 1 * time.Second
)

var (
	queueToLogsStorageWriterWorkersCount = max(runtime.NumCPU()/4, 1) // 25% of CPUs
	accumulationFlushWorkersCount        = max(runtime.NumCPU()/4, 1) // 25% of CPUs
)

func NewLogWorkerService(
	logRepository *logs_core.LogCoreRepository,
	logger *slog.Logger,
) *LogWorkerService {
	service := &LogWorkerService{
		logRepository: logRepository,
		queueService:  cache_utils.NewValkeyQueueService(),
		logger:        logger,

		// Worker control - will be initialized when StartWorkers() is called
		ctx:    nil,
		cancel: nil,
		wg:     sync.WaitGroup{},
	}

	// Initialize sharded accumulation buffers
	service.accumulatedLogShards = make([][]*logs_core.LogItem, accumulationFlushWorkersCount)
	service.accumulationMutexes = make([]sync.RWMutex, accumulationFlushWorkersCount)
	service.flushTickers = make([]*time.Ticker, accumulationFlushWorkersCount)

	for i := range accumulationFlushWorkersCount {
		service.accumulatedLogShards[i] = make(
			[]*logs_core.LogItem,
			0,
			cacheToLogsStorageWritingBatchSize/accumulationFlushWorkersCount,
		)
	}

	return service
}

func (s *LogWorkerService) StartWorkers() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.logger.Info("Starting log worker services with fixed configuration",
		slog.Duration("batchInterval", batchProcessingInterval),
		slog.Int("batchSize", cacheToLogsStorageWritingBatchSize),
		slog.Int("workerCount", queueToLogsStorageWriterWorkersCount),
		slog.Int("flushWorkersCount", accumulationFlushWorkersCount),
		slog.Duration("accumulationFlushInterval", ramToValkeyQueueAccumulationFlushInterval))

	// Start multiple sharded accumulation flush workers
	for i := range accumulationFlushWorkersCount {
		s.wg.Add(1)
		go s.accumulationFlushWorker(i)
	}

	// Start multiple batch processing workers
	for i := range queueToLogsStorageWriterWorkersCount {
		s.wg.Add(1)
		go s.runCacheToLogStorageWorker(i)
	}

	s.logger.Info("All log workers started successfully")
}

// QueueLog adds a single log item to a sharded accumulation buffer.
// Logs are distributed across shards using project ID hash to balance load.
// Each shard is flushed to Valkey automatically every second by dedicated workers.
func (s *LogWorkerService) QueueLog(log *logs_core.LogItem) error {
	if log == nil {
		return nil
	}

	// Hash project ID to determine shard (distribute load evenly)
	shard := s.hashProjectIDToShard(log.ProjectID)

	s.accumulationMutexes[shard].Lock()
	defer s.accumulationMutexes[shard].Unlock()

	s.accumulatedLogShards[shard] = append(s.accumulatedLogShards[shard], log)

	return nil
}

// ExecuteBackgroundTasksForTest executes log flushing tasks once in a blocking way.
// This method is needed for testing to avoid waiting for workers to execute all tasks.
// When this method is called, log flushing is performed immediately:
// - Flush accumulated logs from all producer shards
// - Process any remaining logs in Valkey queue
func (s *LogWorkerService) ExecuteBackgroundTasksForTest() error {
	// Flush accumulated logs from all producer shards
	for shard := range accumulationFlushWorkersCount {
		s.flushAccumulatedLogsShard(shard)
	}

	// Process any remaining logs in Valkey queue (single worker execution)
	s.processLogsFromValkeyQueueToLogsRepository(0)

	return nil
}

// runCacheToLogStorageWorker runs periodically to process logs directly from Valkey to log storage
func (s *LogWorkerService) runCacheToLogStorageWorker(workerID int) {
	defer s.wg.Done()

	ticker := time.NewTicker(batchProcessingInterval)
	defer ticker.Stop()

	s.logger.Info("Cache to log storage worker started",
		slog.Int("workerID", workerID),
		slog.Duration("interval", batchProcessingInterval))

	for {
		if config.IsShouldShutdown() {
			s.logger.Info(
				"Cache to log storage worker shutting down due to shutdown signal",
				slog.Int("workerID", workerID),
			)
			return
		}

		select {
		case <-s.ctx.Done():
			s.logger.Info("Cache to log storage worker shutting down", slog.Int("workerID", workerID))
			return

		case <-ticker.C:
			// Dequeue and process logs from Valkey directly to log storage
			s.processLogsFromValkeyQueueToLogsRepository(workerID)
		}
	}
}

func (s *LogWorkerService) processLogsFromValkeyQueueToLogsRepository(workerID int) {
	// Dequeue batch of logs from Valkey using pipeline for high performance
	// Use non-blocking dequeue to prevent worker from hanging
	serializedLogs, err := s.queueService.DequeueBatch(logQueueKey, cacheToLogsStorageWritingBatchSize, 0)
	if err != nil {
		s.logger.Error("Failed to dequeue logs from Valkey",
			slog.Int("workerID", workerID),
			slog.String("error", err.Error()))
		return
	}

	if len(serializedLogs) == 0 {
		// No logs available, continue
		return
	}

	// Deserialize logs
	var logs []*logs_core.LogItem
	for _, data := range serializedLogs {
		var log logs_core.LogItem

		if err := json.Unmarshal(data, &log); err != nil {
			s.logger.Error("Failed to unmarshal log item from Valkey",
				slog.Int("workerID", workerID),
				slog.String("error", err.Error()))
			continue
		}

		logs = append(logs, &log)
	}

	if len(logs) == 0 {
		return
	}

	// Group logs by project and send directly to log storage
	batch := make(map[uuid.UUID][]*logs_core.LogItem)
	for _, log := range logs {
		batch[log.ProjectID] = append(batch[log.ProjectID], log)
	}

	// Send batch directly to log storage
	startTime := time.Now().UTC()
	err = s.logRepository.StoreLogsBatch(batch)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("Failed to store log batch",
			slog.Int("workerID", workerID),
			slog.Int("totalLogs", len(logs)),
			slog.Int("projects", len(batch)),
			slog.Duration("duration", duration),
			slog.String("error", err.Error()))
	}
}

func (s *LogWorkerService) accumulationFlushWorker(shardID int) {
	defer s.wg.Done()

	s.flushTickers[shardID] = time.NewTicker(ramToValkeyQueueAccumulationFlushInterval)
	defer s.flushTickers[shardID].Stop()

	s.logger.Info("Accumulation flush worker started",
		slog.Int("shardID", shardID),
		slog.Duration("flushInterval", ramToValkeyQueueAccumulationFlushInterval))

	for {
		if config.IsShouldShutdown() {
			s.logger.Info("Accumulation flush worker shutting down due to shutdown signal",
				slog.Int("shardID", shardID))
			s.flushAccumulatedLogsShard(shardID)
			return
		}

		select {
		case <-s.ctx.Done():
			s.logger.Info("Accumulation flush worker shutting down",
				slog.Int("shardID", shardID))
			s.flushAccumulatedLogsShard(shardID)
			return

		case <-s.flushTickers[shardID].C:
			s.flushAccumulatedLogsShard(shardID)
		}
	}
}

func (s *LogWorkerService) flushAccumulatedLogsShard(shardID int) {
	s.accumulationMutexes[shardID].Lock()
	logsToFlush := s.accumulatedLogShards[shardID]
	s.accumulatedLogShards[shardID] = make(
		[]*logs_core.LogItem,
		0,
		cacheToLogsStorageWritingBatchSize/accumulationFlushWorkersCount,
	)
	s.accumulationMutexes[shardID].Unlock()

	if len(logsToFlush) == 0 {
		return
	}

	// Serialize logs to JSON for Valkey storage
	serializedLogs := make([][]byte, 0, len(logsToFlush))

	for _, log := range logsToFlush {
		data, err := json.Marshal(log)
		if err != nil {
			s.logger.Error("Failed to marshal log item during flush",
				slog.Int("shardID", shardID),
				slog.String("logId", log.ID.String()),
				slog.String("error", err.Error()))
			continue
		}
		serializedLogs = append(serializedLogs, data)
	}

	if len(serializedLogs) == 0 {
		return
	}

	// Use batch enqueue with pipeline for maximum performance
	err := s.queueService.EnqueueBatch(logQueueKey, serializedLogs)
	if err != nil {
		s.logger.Error("Failed to flush accumulated logs to Valkey",
			slog.Int("shardID", shardID),
			slog.Int("logsCount", len(serializedLogs)),
			slog.String("error", err.Error()))
		return
	}
}

// hashProjectIDToShard distributes logs across shards using project ID hash.
// This ensures even load distribution and prevents hot-spotting on single shards.
func (s *LogWorkerService) hashProjectIDToShard(projectID uuid.UUID) int {
	// Use a simple hash of the project ID bytes to determine shard
	hash := uint32(0)
	for _, b := range projectID[:] {
		hash = hash*31 + uint32(b)
	}
	return int(hash % uint32(accumulationFlushWorkersCount))
}
