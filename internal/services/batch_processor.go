package services

import (
	"context"
	"sync"
	"time"

	"github.com/lspecian/ovncp/internal/models"
	"go.uber.org/zap"
)

// BatchProcessor handles batching of OVN operations for improved performance
type BatchProcessor struct {
	service        OVNServiceInterface
	logger         *zap.Logger
	batchSize      int
	batchTimeout   time.Duration
	maxConcurrent  int
	
	// Channels for different operation types
	createSwitchCh chan *batchItem
	updateSwitchCh chan *batchItem
	deleteSwitchCh chan *batchItem
	createPortCh   chan *batchItem
	updatePortCh   chan *batchItem
	deletePortCh   chan *batchItem
	
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// batchItem represents a single operation in a batch
type batchItem struct {
	ctx      context.Context
	data     interface{}
	resultCh chan batchResult
}

// batchResult represents the result of a batch operation
type batchResult struct {
	data interface{}
	err  error
}

// BatchProcessorConfig holds configuration for batch processor
type BatchProcessorConfig struct {
	BatchSize     int
	BatchTimeout  time.Duration
	MaxConcurrent int
}

// DefaultBatchProcessorConfig returns default configuration
func DefaultBatchProcessorConfig() *BatchProcessorConfig {
	return &BatchProcessorConfig{
		BatchSize:     100,
		BatchTimeout:  100 * time.Millisecond,
		MaxConcurrent: 4,
	}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(service OVNServiceInterface, cfg *BatchProcessorConfig, logger *zap.Logger) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	bp := &BatchProcessor{
		service:        service,
		logger:         logger,
		batchSize:      cfg.BatchSize,
		batchTimeout:   cfg.BatchTimeout,
		maxConcurrent:  cfg.MaxConcurrent,
		createSwitchCh: make(chan *batchItem, cfg.BatchSize),
		updateSwitchCh: make(chan *batchItem, cfg.BatchSize),
		deleteSwitchCh: make(chan *batchItem, cfg.BatchSize),
		createPortCh:   make(chan *batchItem, cfg.BatchSize),
		updatePortCh:   make(chan *batchItem, cfg.BatchSize),
		deletePortCh:   make(chan *batchItem, cfg.BatchSize),
		ctx:            ctx,
		cancel:         cancel,
	}
	
	// Start batch workers
	bp.start()
	
	return bp
}

// start initializes all batch workers
func (bp *BatchProcessor) start() {
	// Switch workers
	bp.wg.Add(3)
	go bp.batchWorker("create_switch", bp.createSwitchCh, bp.processCreateSwitchBatch)
	go bp.batchWorker("update_switch", bp.updateSwitchCh, bp.processUpdateSwitchBatch)
	go bp.batchWorker("delete_switch", bp.deleteSwitchCh, bp.processDeleteSwitchBatch)
	
	// Port workers
	bp.wg.Add(3)
	go bp.batchWorker("create_port", bp.createPortCh, bp.processCreatePortBatch)
	go bp.batchWorker("update_port", bp.updatePortCh, bp.processUpdatePortBatch)
	go bp.batchWorker("delete_port", bp.deletePortCh, bp.processDeletePortBatch)
}

// Stop gracefully stops the batch processor
func (bp *BatchProcessor) Stop() {
	bp.cancel()
	
	// Close all channels
	close(bp.createSwitchCh)
	close(bp.updateSwitchCh)
	close(bp.deleteSwitchCh)
	close(bp.createPortCh)
	close(bp.updatePortCh)
	close(bp.deletePortCh)
	
	// Wait for all workers to finish
	bp.wg.Wait()
}

// batchWorker is a generic batch processing worker
func (bp *BatchProcessor) batchWorker(name string, ch chan *batchItem, processFn func([]batchItem)) {
	defer bp.wg.Done()
	
	ticker := time.NewTicker(bp.batchTimeout)
	defer ticker.Stop()
	
	batch := make([]batchItem, 0, bp.batchSize)
	
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				// Channel closed, process remaining batch
				if len(batch) > 0 {
					processFn(batch)
				}
				return
			}
			
			batch = append(batch, *item)
			
			// Process batch if it's full
			if len(batch) >= bp.batchSize {
				processFn(batch)
				batch = batch[:0]
				ticker.Reset(bp.batchTimeout)
			}
			
		case <-ticker.C:
			// Process batch on timeout
			if len(batch) > 0 {
				processFn(batch)
				batch = batch[:0]
			}
			
		case <-bp.ctx.Done():
			// Shutdown requested, process remaining batch
			if len(batch) > 0 {
				processFn(batch)
			}
			return
		}
	}
}

// Logical Switch batch operations

// CreateLogicalSwitchBatch creates multiple switches in a batch
func (bp *BatchProcessor) CreateLogicalSwitchBatch(ctx context.Context, switches []*models.LogicalSwitch) error {
	resultChs := make([]chan batchResult, len(switches))
	
	// Queue all items
	for i, sw := range switches {
		resultCh := make(chan batchResult, 1)
		resultChs[i] = resultCh
		
		select {
		case bp.createSwitchCh <- &batchItem{
			ctx:      ctx,
			data:     sw,
			resultCh: resultCh,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	// Wait for all results
	var firstErr error
	for i, resultCh := range resultChs {
		select {
		case result := <-resultCh:
			if result.err != nil && firstErr == nil {
				firstErr = result.err
			}
			// Update the switch with any returned data (like UUID)
			if result.data != nil {
				*switches[i] = *result.data.(*models.LogicalSwitch)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	return firstErr
}

// processCreateSwitchBatch processes a batch of switch creations
func (bp *BatchProcessor) processCreateSwitchBatch(items []batchItem) {
	bp.logger.Debug("Processing switch creation batch", zap.Int("size", len(items)))
	
	// Group operations by transaction
	ops := make([]TransactionOp, 0, len(items))
	for _, item := range items {
		sw := item.data.(*models.LogicalSwitch)
		ops = append(ops, TransactionOp{
			Operation:    "create",
			ResourceType: "logical_switch",
			Data:         sw,
		})
	}
	
	// Execute as a single transaction
	err := bp.service.ExecuteTransaction(context.Background(), ops)
	
	// Send results back
	for i, item := range items {
		result := batchResult{err: err}
		if err == nil {
			result.data = ops[i].Data
		}
		item.resultCh <- result
		close(item.resultCh)
	}
}

// UpdateLogicalSwitchBatch updates multiple switches in a batch
func (bp *BatchProcessor) UpdateLogicalSwitchBatch(ctx context.Context, switches []*models.LogicalSwitch) error {
	resultChs := make([]chan batchResult, len(switches))
	
	// Queue all items
	for i, sw := range switches {
		resultCh := make(chan batchResult, 1)
		resultChs[i] = resultCh
		
		select {
		case bp.updateSwitchCh <- &batchItem{
			ctx:      ctx,
			data:     sw,
			resultCh: resultCh,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	// Wait for all results
	var firstErr error
	for _, resultCh := range resultChs {
		select {
		case result := <-resultCh:
			if result.err != nil && firstErr == nil {
				firstErr = result.err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	return firstErr
}

// processUpdateSwitchBatch processes a batch of switch updates
func (bp *BatchProcessor) processUpdateSwitchBatch(items []batchItem) {
	bp.logger.Debug("Processing switch update batch", zap.Int("size", len(items)))
	
	// Group operations by transaction
	ops := make([]TransactionOp, 0, len(items))
	for _, item := range items {
		sw := item.data.(*models.LogicalSwitch)
		ops = append(ops, TransactionOp{
			Operation:    "update",
			ResourceType: "logical_switch",
			ResourceID:   sw.UUID,
			Data:         sw,
		})
	}
	
	// Execute as a single transaction
	err := bp.service.ExecuteTransaction(context.Background(), ops)
	
	// Send results back
	for _, item := range items {
		item.resultCh <- batchResult{err: err}
		close(item.resultCh)
	}
}

// DeleteLogicalSwitchBatch deletes multiple switches in a batch
func (bp *BatchProcessor) DeleteLogicalSwitchBatch(ctx context.Context, ids []string) error {
	resultChs := make([]chan batchResult, len(ids))
	
	// Queue all items
	for i, id := range ids {
		resultCh := make(chan batchResult, 1)
		resultChs[i] = resultCh
		
		select {
		case bp.deleteSwitchCh <- &batchItem{
			ctx:      ctx,
			data:     id,
			resultCh: resultCh,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	// Wait for all results
	var firstErr error
	for _, resultCh := range resultChs {
		select {
		case result := <-resultCh:
			if result.err != nil && firstErr == nil {
				firstErr = result.err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	return firstErr
}

// processDeleteSwitchBatch processes a batch of switch deletions
func (bp *BatchProcessor) processDeleteSwitchBatch(items []batchItem) {
	bp.logger.Debug("Processing switch deletion batch", zap.Int("size", len(items)))
	
	// Group operations by transaction
	ops := make([]TransactionOp, 0, len(items))
	for _, item := range items {
		id := item.data.(string)
		ops = append(ops, TransactionOp{
			Operation:    "delete",
			ResourceType: "logical_switch",
			ResourceID:   id,
		})
	}
	
	// Execute as a single transaction
	err := bp.service.ExecuteTransaction(context.Background(), ops)
	
	// Send results back
	for _, item := range items {
		item.resultCh <- batchResult{err: err}
		close(item.resultCh)
	}
}

// Port batch operations

// CreatePortBatch creates multiple ports in a batch
func (bp *BatchProcessor) CreatePortBatch(ctx context.Context, ports []*models.LogicalSwitchPort) error {
	resultChs := make([]chan batchResult, len(ports))
	
	// Queue all items
	for i, port := range ports {
		resultCh := make(chan batchResult, 1)
		resultChs[i] = resultCh
		
		select {
		case bp.createPortCh <- &batchItem{
			ctx:      ctx,
			data:     port,
			resultCh: resultCh,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	// Wait for all results
	var firstErr error
	for i, resultCh := range resultChs {
		select {
		case result := <-resultCh:
			if result.err != nil && firstErr == nil {
				firstErr = result.err
			}
			// Update the port with any returned data
			if result.data != nil {
				*ports[i] = *result.data.(*models.LogicalSwitchPort)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	return firstErr
}

// processCreatePortBatch processes a batch of port creations
func (bp *BatchProcessor) processCreatePortBatch(items []batchItem) {
	bp.logger.Debug("Processing port creation batch", zap.Int("size", len(items)))
	
	// Group operations by transaction
	ops := make([]TransactionOp, 0, len(items))
	for _, item := range items {
		port := item.data.(*models.LogicalSwitchPort)
		ops = append(ops, TransactionOp{
			Operation:    "create",
			ResourceType: "logical_port",
			Data:         port,
		})
	}
	
	// Execute as a single transaction
	err := bp.service.ExecuteTransaction(context.Background(), ops)
	
	// Send results back
	for i, item := range items {
		result := batchResult{err: err}
		if err == nil {
			result.data = ops[i].Data
		}
		item.resultCh <- result
		close(item.resultCh)
	}
}

// processUpdatePortBatch processes a batch of port updates
func (bp *BatchProcessor) processUpdatePortBatch(items []batchItem) {
	bp.logger.Debug("Processing port update batch", zap.Int("size", len(items)))
	
	// Group operations by transaction
	ops := make([]TransactionOp, 0, len(items))
	for _, item := range items {
		port := item.data.(*models.LogicalSwitchPort)
		ops = append(ops, TransactionOp{
			Operation:    "update",
			ResourceType: "logical_port",
			ResourceID:   port.UUID,
			Data:         port,
		})
	}
	
	// Execute as a single transaction
	err := bp.service.ExecuteTransaction(context.Background(), ops)
	
	// Send results back
	for _, item := range items {
		item.resultCh <- batchResult{err: err}
		close(item.resultCh)
	}
}

// processDeletePortBatch processes a batch of port deletions
func (bp *BatchProcessor) processDeletePortBatch(items []batchItem) {
	bp.logger.Debug("Processing port deletion batch", zap.Int("size", len(items)))
	
	// Group operations by transaction
	ops := make([]TransactionOp, 0, len(items))
	for _, item := range items {
		id := item.data.(string)
		ops = append(ops, TransactionOp{
			Operation:    "delete",
			ResourceType: "logical_port",
			ResourceID:   id,
		})
	}
	
	// Execute as a single transaction
	err := bp.service.ExecuteTransaction(context.Background(), ops)
	
	// Send results back
	for _, item := range items {
		item.resultCh <- batchResult{err: err}
		close(item.resultCh)
	}
}

// Utility functions

// BatchedService wraps a service with batch processing capabilities
type BatchedService struct {
	OVNServiceInterface
	processor *BatchProcessor
}

// NewBatchedService creates a new batched service wrapper
func NewBatchedService(service OVNServiceInterface, cfg *BatchProcessorConfig, logger *zap.Logger) *BatchedService {
	return &BatchedService{
		OVNServiceInterface: service,
		processor:          NewBatchProcessor(service, cfg, logger),
	}
}

// CreateLogicalSwitches creates multiple switches efficiently
func (bs *BatchedService) CreateLogicalSwitches(ctx context.Context, switches []*models.LogicalSwitch) error {
	if len(switches) <= 1 {
		// For single item, use regular method
		if len(switches) == 1 {
			_, err := bs.CreateLogicalSwitch(ctx, switches[0])
			return err
		}
		return nil
	}
	
	return bs.processor.CreateLogicalSwitchBatch(ctx, switches)
}

// UpdateLogicalSwitches updates multiple switches efficiently
func (bs *BatchedService) UpdateLogicalSwitches(ctx context.Context, switches []*models.LogicalSwitch) error {
	if len(switches) <= 1 {
		// For single item, use regular method
		if len(switches) == 1 {
			_, err := bs.UpdateLogicalSwitch(ctx, switches[0].UUID, switches[0])
			return err
		}
		return nil
	}
	
	return bs.processor.UpdateLogicalSwitchBatch(ctx, switches)
}

// DeleteLogicalSwitches deletes multiple switches efficiently
func (bs *BatchedService) DeleteLogicalSwitches(ctx context.Context, ids []string) error {
	if len(ids) <= 1 {
		// For single item, use regular method
		if len(ids) == 1 {
			return bs.DeleteLogicalSwitch(ctx, ids[0])
		}
		return nil
	}
	
	return bs.processor.DeleteLogicalSwitchBatch(ctx, ids)
}

// CreatePorts creates multiple ports efficiently
func (bs *BatchedService) CreatePorts(ctx context.Context, ports []*models.LogicalSwitchPort) error {
	if len(ports) <= 1 {
		// For single item, use regular method
		if len(ports) == 1 {
			// Use SwitchID if ParentUUID is not set
			switchID := ports[0].ParentUUID
			if switchID == "" {
				switchID = ports[0].SwitchID
			}
			_, err := bs.CreatePort(ctx, switchID, ports[0])
			return err
		}
		return nil
	}
	
	return bs.processor.CreatePortBatch(ctx, ports)
}

// Stop stops the batch processor
func (bs *BatchedService) Stop() {
	bs.processor.Stop()
}

// GetBatchStats returns batch processing statistics
func (bs *BatchedService) GetBatchStats() map[string]interface{} {
	// This could be enhanced to track actual statistics
	return map[string]interface{}{
		"batch_size":     bs.processor.batchSize,
		"batch_timeout":  bs.processor.batchTimeout,
		"max_concurrent": bs.processor.maxConcurrent,
	}
}