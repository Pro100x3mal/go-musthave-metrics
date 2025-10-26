# Анализ оптимизации памяти сервера метрик

В рамках оптимизации были проведены работы по улучшению управления памятью в сервере метрик. Основная цель — снизить потребление памяти и количество аллокаций при обработке HTTP-запросов и работе с базой данных.


### 1. Оптимизация обработки шаблонов

**Проблема:** При каждом запросе к `/` создавался новый HTML-шаблон, что приводило к излишним аллокациям.

**Решение:** Шаблон был вынесен на уровень структуры `MetricsHandler` и парсится один раз при инициализации:

```go
// internal/server/handlers/serve.go

type MetricsHandler struct {
    // ...
    tmpl *template.Template
}

func NewMetricsHandler(...) *MetricsHandler {
    return &MetricsHandler{
        // ...
        tmpl: template.Must(template.New("metrics").Parse(metricsTemplate)),
    }
}
```

### 2. Оптимизация middleware сжатия

**Проблема:** При каждом запросе с `Content-Encoding: gzip` создавался новый `gzip.Reader`, что приводило к множественным аллокациям внутренних буферов.

**Решение:**
- Добавлен `sync.Pool` для переиспользования `gzip.Reader`
- Изменен уровень сжатия с `gzip.DefaultCompression` на `gzip.BestSpeed` для `gzip.Writer`

```go
// internal/server/middlewares/compress.go

type CompressHandler struct {
    logger     *zap.Logger
    readerPool *sync.Pool
}
// ...
func NewCompressHandler(logger *zap.Logger) *CompressHandler {
    return &CompressHandler{
        logger: logger,
        readerPool: &sync.Pool{
            New: func() interface{} {
                reader, _ := gzip.NewReader(strings.NewReader(""))
                return reader
            },
        },
    }
}
// ...
func newCompressWriter(w http.ResponseWriter) *compressWriter {
    gzw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
    return &compressWriter{
        w:   w,
        gzw: gzw,
    }
}
```


### 3. Оптимизация работы с базой данных

**Проблема:** При формировании SQL-запросов для батчевого обновления метрик слайсы для значений и аргументов объявлялись без предварительной аллокации с фиксированной емкостью.

**Решение:** Добавлена преаллокация слайсов с известной емкостью (отдельно протестировано с помощью benchmark):

```go
// internal/server/repositories/db_storage.go

func (db *DB) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	// ...
	gauges := make([]models.Metrics, 0, len(gaugeMap))
    // ...
    counters := make([]models.Metrics, 0, len(counterMap))
	// ...
}
//...
func (db *DB) updateMetricsChunk(ctx context.Context, gauges, counters []models.Metrics) error {
    if len(gauges) > 0 {
        values := make([]string, 0, len(gauges))
        args := make([]any, 0, len(gauges)*2)
        // ...
    }
// ...
    if len(counters) > 0 {
        values := make([]string, 0, len(counters))
        args := make([]any, 0, len(counters)*2)
        // ... 
    }
//...
}
```

### 4. Оптимизация батчевой обработки метрик при большом количестве элементов

**Проблема:** При большом количестве элементов для батчевого запроса к базе данных (более 1000) скорость обработки запросов снижалась (парсинг большего количества параметров запроса)

**Решение:** Добавлена функция разделения больших батчей на части (чанки): 

```go
func splitMetricsIntoChunks(items []models.Metrics, chunkSize int) [][]models.Metrics {
    if chunkSize <= 0 {
        chunkSize = defaultChunkSize
    }
    if len(items) == 0 {
        return nil
    }
    chunks := make([][]models.Metrics, 0, len(items)/chunkSize+1)
    for i := 0; i < len(items); i += chunkSize {
        end := i + chunkSize
        if end > len(items) {
            end = len(items)
        }
        chunks = append(chunks, items[i:end])
    }
    return chunks
}
```

### Итоговый результат оптимизаций:
```
File: server
Type: inuse_space
Time: 2025-10-20 01:28:21 MSK
Showing nodes accounting for 2143.14kB, 26.38% of 8123.75kB total
Dropped 5 nodes (cum <= 40.62kB)
      flat  flat%   sum%        cum   cum%
 1026.25kB 12.63% 12.63%  1026.25kB 12.63%  github.com/jackc/pgx/v5/pgxpool.(*connResource).getConn (inline)
 1024.06kB 12.61% 25.24%  1024.06kB 12.61%  github.com/jackc/puddle/v2.(*Pool[go.shape.*uint8]).createNewResource
 -902.59kB 11.11% 14.13% -1447.25kB 17.82%  compress/flate.NewWriter (inline)
 -544.67kB  6.70%  7.42%  -544.67kB  6.70%  compress/flate.(*compressor).initDeflate (inline)
  521.05kB  6.41% 13.84%   521.05kB  6.41%  github.com/lib/pq.map.init.0
 -518.65kB  6.38%  7.45%  -517.67kB  6.37%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories.(*DB).GetAllGauges
 -516.76kB  6.36%  1.09%  -516.76kB  6.36%  runtime.procresize
  516.01kB  6.35%  7.44%   516.01kB  6.35%  github.com/jackc/pgx/v5/internal/iobufpool.init.0.func1
     514kB  6.33% 13.77%      514kB  6.33%  bufio.NewReaderSize (inline)
  512.56kB  6.31% 20.08%   512.56kB  6.31%  runtime.makeProfStackFP (inline)
 -512.22kB  6.31% 13.78%  -512.22kB  6.31%  runtime.malg
 -512.14kB  6.30%  7.47%  -512.14kB  6.30%  github.com/jackc/pgx/v5.(*Conn).getRows
  512.10kB  6.30% 13.77%   512.10kB  6.30%  context.(*cancelCtx).propagateCancel
  512.06kB  6.30% 20.08%   512.06kB  6.30%  net.newFD (inline)
  512.05kB  6.30% 26.38%  1033.11kB 12.72%  runtime.main
  512.03kB  6.30% 32.68%   512.03kB  6.30%  syscall.anyToSockaddr
 -512.02kB  6.30% 26.38%  -512.02kB  6.30%  github.com/jackc/pgx/v5/pgtype.NewMap
         0     0% 26.38%      514kB  6.33%  bufio.NewReader (inline)
         0     0% 26.38%  -544.67kB  6.70%  compress/flate.(*compressor).init
         0     0% 26.38% -1447.25kB 17.82%  compress/gzip.(*Writer).Write
         0     0% 26.38%   512.10kB  6.30%  context.WithCancel
         0     0% 26.38%   512.10kB  6.30%  context.withCancel (inline)
         0     0% 26.38% -1451.80kB 17.87%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers.(*MetricsHandler).ListAllMetricsHandler
         0     0% 26.38%   512.06kB  6.30%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers.(*MetricsHandler).StartServer.func1
         0     0% 26.38% -1451.80kB 17.87%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/middlewares.(*CompressHandler).Middleware-fm.(*CompressHandler).Middleware.func1
         0     0% 26.38% -1451.80kB 17.87%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/middlewares.(*LoggerHandler).Middleware-fm.(*LoggerHandler).Middleware.func1
         0     0% 26.38% -1447.25kB 17.82%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/middlewares.(*compressWriter).Write
         0     0% 26.38%   513.12kB  6.32%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories.(*DB).GetAllCounters
         0     0% 26.38%   513.12kB  6.32%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories/retry.(*RepoWithRetry).GetAllCounters
         0     0% 26.38%   513.12kB  6.32%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories/retry.(*RepoWithRetry).GetAllCounters.func1
         0     0% 26.38%  -517.67kB  6.37%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories/retry.(*RepoWithRetry).GetAllGauges
         0     0% 26.38%  -517.67kB  6.37%  github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories/retry.(*RepoWithRetry).GetAllGauges.func1
         0     0% 26.38% -1451.80kB 17.87%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 26.38% -1451.80kB 17.87%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 26.38%  -512.14kB  6.30%  github.com/jackc/pgx/v5.(*Conn).Query
         0     0% 26.38%   516.02kB  6.35%  github.com/jackc/pgx/v5.ConnectConfig
         0     0% 26.38%   516.02kB  6.35%  github.com/jackc/pgx/v5.connect
         0     0% 26.38%   516.01kB  6.35%  github.com/jackc/pgx/v5/internal/iobufpool.Get
         0     0% 26.38%  1028.04kB 12.65%  github.com/jackc/pgx/v5/pgconn.ConnectConfig
         0     0% 26.38%   516.01kB  6.35%  github.com/jackc/pgx/v5/pgconn.ParseConfigWithOptions.func1
         0     0% 26.38%  1028.04kB 12.65%  github.com/jackc/pgx/v5/pgconn.connectOne
         0     0% 26.38%  1028.04kB 12.65%  github.com/jackc/pgx/v5/pgconn.connectPreferred
         0     0% 26.38%   516.01kB  6.35%  github.com/jackc/pgx/v5/pgproto3.NewFrontend
         0     0% 26.38%   516.01kB  6.35%  github.com/jackc/pgx/v5/pgproto3.newChunkReader (inline)
         0     0% 26.38%  -512.14kB  6.30%  github.com/jackc/pgx/v5/pgxpool.(*Conn).Query
         0     0% 26.38%  1026.25kB 12.63%  github.com/jackc/pgx/v5/pgxpool.(*Pool).Acquire
         0     0% 26.38%   514.11kB  6.33%  github.com/jackc/pgx/v5/pgxpool.(*Pool).Query
         0     0% 26.38%   512.04kB  6.30%  github.com/jackc/pgx/v5/pgxpool.(*Pool).createIdleResources.func1
         0     0% 26.38%   516.02kB  6.35%  github.com/jackc/pgx/v5/pgxpool.NewWithConfig.func1
         0     0% 26.38%   512.04kB  6.30%  github.com/jackc/puddle/v2.(*Pool[go.shape.*uint8]).CreateResource
         0     0% 26.38%  1028.04kB 12.65%  github.com/jackc/puddle/v2.(*Pool[go.shape.*uint8]).initResourceValue.func1
         0     0% 26.38%   521.05kB  6.41%  github.com/lib/pq.init
         0     0% 26.38% -1447.25kB 17.82%  html/template.(*Template).Execute
         0     0% 26.38%   512.03kB  6.30%  net.(*Dialer).DialContext
         0     0% 26.38%   512.06kB  6.30%  net.(*TCPListener).Accept
         0     0% 26.38%   512.06kB  6.30%  net.(*TCPListener).accept
         0     0% 26.38%   512.06kB  6.30%  net.(*netFD).accept
         0     0% 26.38%   512.03kB  6.30%  net.(*netFD).dial
         0     0% 26.38%   512.03kB  6.30%  net.(*sysDialer).dialParallel
         0     0% 26.38%   512.03kB  6.30%  net.(*sysDialer).dialSerial
         0     0% 26.38%   512.03kB  6.30%  net.(*sysDialer).dialSingle
         0     0% 26.38%   512.03kB  6.30%  net.(*sysDialer).dialTCP
         0     0% 26.38%   512.03kB  6.30%  net.(*sysDialer).doDialTCP (inline)
         0     0% 26.38%   512.03kB  6.30%  net.(*sysDialer).doDialTCPProto
         0     0% 26.38%   512.03kB  6.30%  net.internetSocket
         0     0% 26.38%   512.03kB  6.30%  net.socket
         0     0% 26.38%   512.06kB  6.30%  net/http.(*Server).ListenAndServe
         0     0% 26.38%   512.06kB  6.30%  net/http.(*Server).Serve
         0     0% 26.38%   512.10kB  6.30%  net/http.(*conn).readRequest
         0     0% 26.38%  -425.69kB  5.24%  net/http.(*conn).serve
         0     0% 26.38% -1451.80kB 17.87%  net/http.HandlerFunc.ServeHTTP
         0     0% 26.38%      514kB  6.33%  net/http.newBufioReader
         0     0% 26.38% -1451.80kB 17.87%  net/http.serverHandler.ServeHTTP
         0     0% 26.38%   521.05kB  6.41%  runtime.doInit (inline)
         0     0% 26.38%   521.05kB  6.41%  runtime.doInit1
         0     0% 26.38%   512.56kB  6.31%  runtime.mProfStackInit (inline)
         0     0% 26.38%   512.56kB  6.31%  runtime.main.func1
         0     0% 26.38%    -1026kB 12.63%  runtime.mcall
         0     0% 26.38%   512.56kB  6.31%  runtime.mcommoninit
         0     0% 26.38%      513kB  6.31%  runtime.mstart
         0     0% 26.38%      513kB  6.31%  runtime.mstart0
         0     0% 26.38%      513kB  6.31%  runtime.mstart1
         0     0% 26.38%  -512.22kB  6.31%  runtime.newproc.func1
         0     0% 26.38%  -512.22kB  6.31%  runtime.newproc1
         0     0% 26.38%    -1026kB 12.63%  runtime.park_m
         0     0% 26.38%     -513kB  6.31%  runtime.resetspinning
         0     0% 26.38%  -516.76kB  6.36%  runtime.rt0_go
         0     0% 26.38%  -516.76kB  6.36%  runtime.schedinit
         0     0% 26.38%     -513kB  6.31%  runtime.schedule
         0     0% 26.38%     -513kB  6.31%  runtime.startm
         0     0% 26.38%     -513kB  6.31%  runtime.wakep
         0     0% 26.38%   516.01kB  6.35%  sync.(*Pool).Get
         0     0% 26.38%   512.03kB  6.30%  syscall.Getsockname
         0     0% 26.38% -1447.25kB 17.82%  text/template.(*Template).Execute (inline)
         0     0% 26.38% -1447.25kB 17.82%  text/template.(*Template).execute
         0     0% 26.38% -1447.25kB 17.82%  text/template.(*state).walk
```