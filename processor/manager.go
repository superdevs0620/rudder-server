package processor

import (
	"context"
	"sync"

	"github.com/rudderlabs/rudder-go-kit/config"
	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/rudderlabs/rudder-go-kit/stats"
	"github.com/rudderlabs/rudder-go-kit/stats/metric"

	transformationdebugger "github.com/rudderlabs/rudder-server/services/debugger/transformation"

	destinationdebugger "github.com/rudderlabs/rudder-server/services/debugger/destination"

	backendconfig "github.com/rudderlabs/rudder-server/backend-config"
	"github.com/rudderlabs/rudder-server/jobsdb"
	"github.com/rudderlabs/rudder-server/processor/transformer"
	"github.com/rudderlabs/rudder-server/services/fileuploader"
	"github.com/rudderlabs/rudder-server/services/rsources"
	"github.com/rudderlabs/rudder-server/services/transientsource"
	"github.com/rudderlabs/rudder-server/utils/types"
)

type LifecycleManager struct {
	Handle           *Handle
	mainCtx          context.Context
	currentCancel    context.CancelFunc
	waitGroup        interface{ Wait() }
	gatewayDB        *jobsdb.Handle
	routerDB         *jobsdb.Handle
	batchRouterDB    *jobsdb.Handle
	readErrDB        *jobsdb.Handle
	writeErrDB       *jobsdb.Handle
	esDB             *jobsdb.Handle
	arcDB            *jobsdb.Handle
	clearDB          *bool
	ReportingI       types.Reporting // need not initialize again
	BackendConfig    backendconfig.BackendConfig
	Transformer      transformer.Transformer
	transientSources transientsource.Service
	fileuploader     fileuploader.Provider
	rsourcesService  rsources.JobService
	destDebugger     destinationdebugger.DestinationDebugger
	transDebugger    transformationdebugger.TransformationDebugger
}

// Start starts a processor, this is not a blocking call.
// If the processor is not completely started and the data started coming then also it will not be problematic as we
// are assuming that the DBs will be up.
func (proc *LifecycleManager) Start() error {
	if proc.Transformer != nil {
		proc.Handle.transformer = proc.Transformer
	}

	proc.Handle.Setup(
		proc.BackendConfig, proc.gatewayDB, proc.routerDB, proc.batchRouterDB, proc.readErrDB, proc.writeErrDB, proc.esDB, proc.arcDB,
		proc.ReportingI, proc.transientSources, proc.fileuploader, proc.rsourcesService, proc.destDebugger, proc.transDebugger,
	)

	currentCtx, cancel := context.WithCancel(context.Background())
	proc.currentCancel = cancel

	var wg sync.WaitGroup
	proc.waitGroup = &wg
	metric.Instance.Reset()
	if err := proc.Handle.countPendingEvents(currentCtx); err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := proc.Handle.Start(currentCtx); err != nil {
			proc.Handle.logger.Errorf("Error starting processor: %v", err)
		}
	}()
	return nil
}

// Stop stops the processor, this is a blocking call.
func (proc *LifecycleManager) Stop() {
	proc.currentCancel()
	proc.waitGroup.Wait()
	proc.Handle.Shutdown()
}

func WithFeaturesRetryMaxAttempts(maxAttempts int) func(l *LifecycleManager) {
	return func(l *LifecycleManager) {
		l.Handle.config.featuresRetryMaxAttempts = maxAttempts
	}
}

// New creates a new Processor instance
func New(ctx context.Context, clearDb *bool, gwDb, rtDb, brtDb, errDbForRead, errDBForWrite, esDB, arcDB *jobsdb.Handle,
	reporting types.Reporting, transientSources transientsource.Service, fileuploader fileuploader.Provider,
	rsourcesService rsources.JobService, destDebugger destinationdebugger.DestinationDebugger, transDebugger transformationdebugger.TransformationDebugger,
	opts ...Opts,
) *LifecycleManager {
	proc := &LifecycleManager{
		Handle:           NewHandle(transformer.NewTransformer(config.Default, logger.NewLogger().Child("processor"), stats.Default)),
		mainCtx:          ctx,
		gatewayDB:        gwDb,
		routerDB:         rtDb,
		batchRouterDB:    brtDb,
		readErrDB:        errDbForRead,
		writeErrDB:       errDBForWrite,
		esDB:             esDB,
		arcDB:            arcDB,
		clearDB:          clearDb,
		BackendConfig:    backendconfig.DefaultBackendConfig,
		ReportingI:       reporting,
		transientSources: transientSources,
		fileuploader:     fileuploader,
		rsourcesService:  rsourcesService,
		destDebugger:     destDebugger,
		transDebugger:    transDebugger,
	}
	for _, opt := range opts {
		opt(proc)
	}
	return proc
}

type Opts func(l *LifecycleManager)

func WithAdaptiveLimit(adaptiveLimitFunction func(int64) int64) Opts {
	return func(l *LifecycleManager) {
		l.Handle.adaptiveLimit = adaptiveLimitFunction
	}
}
