package transform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type wasmHostKey struct{}

func withWasmHost(ctx context.Context, host *wasmHost) context.Context {
	return context.WithValue(ctx, wasmHostKey{}, host)
}

func wasmHostFromContext(ctx context.Context) (*wasmHost, bool) {
	host, ok := ctx.Value(wasmHostKey{}).(*wasmHost)
	return host, ok
}

type wasmModule struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	cfg      wasmConfig
	path     string
}

type wasmHost struct {
	in       Input
	resPath  []byte
	cdnPath  []byte
	platform int32
}

func newWasmModule(baseDir string, cfg wasmConfig) (*wasmModule, error) {
	cfg = cfg.withDefaults()
	modulePath := resolvePath(baseDir, cfg.Module)

	data, err := os.ReadFile(modulePath)
	if err != nil {
		return nil, fmt.Errorf("read wasm module %q: %w", modulePath, err)
	}

	ctx := context.Background()
	runtimeCfg := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	if cfg.MaxMemoryBytes > 0 {
		pages := uint32((cfg.MaxMemoryBytes + 65535) / 65536)
		if pages == 0 {
			pages = 1
		}
		runtimeCfg = runtimeCfg.WithMemoryLimitPages(pages)
	}
	r := wazero.NewRuntimeWithConfig(ctx, runtimeCfg)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("instantiate wasi: %w", err)
	}

	if err := instantiateWasmHost(ctx, r); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}

	compiled, err := r.CompileModule(ctx, data)
	if err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("compile wasm module %q: %w", modulePath, err)
	}

	return &wasmModule{
		runtime:  r,
		compiled: compiled,
		cfg:      cfg,
		path:     modulePath,
	}, nil
}

func instantiateWasmHost(ctx context.Context, r wazero.Runtime) error {
	builder := r.NewHostModuleBuilder(wasmModuleEnv)
	builder.NewFunctionBuilder().
		WithFunc(getResPathHost).
		Export("get_res_path")
	builder.NewFunctionBuilder().
		WithFunc(getCDNPathHost).
		Export("get_cdn_path")
	builder.NewFunctionBuilder().
		WithFunc(getPlatformHost).
		Export("get_platform")
	_, err := builder.Instantiate(ctx)
	if err != nil {
		return fmt.Errorf("instantiate host module: %w", err)
	}
	return nil
}

func (m *wasmModule) Close(ctx context.Context) error {
	if m == nil || m.runtime == nil {
		return nil
	}
	return m.runtime.Close(ctx)
}

func (m *wasmModule) transform(ctx context.Context, in Input) ([]byte, error) {
	host := &wasmHost{
		in:       in,
		resPath:  []byte(in.ResPath),
		cdnPath:  []byte(in.CDNPath),
		platform: platformCode(in.Platform),
	}

	runCtx, cancel := context.WithTimeout(ctx, wasmExecutionTimeout(m.cfg.Fuel))
	defer cancel()
	runCtx = withWasmHost(runCtx, host)

	instanceName := fmt.Sprintf("%s-%p", filepath.Base(m.path), host)
	modCtx := wazero.NewModuleConfig().
		WithName(instanceName)

	module, err := m.runtime.InstantiateModule(runCtx, m.compiled, modCtx)
	if err != nil {
		return nil, fmt.Errorf("instantiate wasm module: %w", err)
	}
	defer func() { _ = module.Close(runCtx) }()

	fn := module.ExportedFunction(m.cfg.Export)
	if fn == nil {
		return nil, fmt.Errorf("wasm module missing export %q", m.cfg.Export)
	}

	mem := module.Memory()
	if mem == nil {
		return nil, fmt.Errorf("wasm module has no memory export")
	}

	inLen := uint32(len(in.Data))
	outMax := uint32(m.cfg.MaxOutputBytes)
	needed := inLen + outMax + 1024
	if _, ok := mem.Grow(ceilPages(needed)); !ok {
		return nil, fmt.Errorf("grow wasm memory")
	}

	inPtr := uint32(0)
	outPtr := inLen
	if inLen > 0 {
		if ok := mem.Write(inPtr, in.Data); !ok {
			return nil, fmt.Errorf("write input to wasm memory")
		}
	}

	results, err := fn.Call(runCtx, uint64(inPtr), uint64(inLen), uint64(outPtr), uint64(outMax))
	if err != nil {
		return nil, fmt.Errorf("call transform: %w", err)
	}
	if len(results) != 1 {
		return nil, fmt.Errorf("transform returned %d results, want 1", len(results))
	}

	outLen := int32(results[0])
	switch {
	case outLen == wasmErrOutputTooLarge:
		return nil, fmt.Errorf("transform output exceeds max_output_bytes (%d)", m.cfg.MaxOutputBytes)
	case outLen == wasmErrInvalidOutput:
		return nil, fmt.Errorf("transform returned invalid output")
	case outLen < 0:
		return nil, fmt.Errorf("transform returned error code %d", outLen)
	case outLen == 0:
		return []byte{}, nil
	case int(outLen) > m.cfg.MaxOutputBytes:
		return nil, fmt.Errorf("transform output exceeds max_output_bytes (%d > %d)", outLen, m.cfg.MaxOutputBytes)
	}

	out, ok := mem.Read(outPtr, uint32(outLen))
	if !ok {
		return nil, fmt.Errorf("read transform output from wasm memory")
	}

	return append([]byte(nil), out...), nil
}

func wasmExecutionTimeout(fuel uint64) time.Duration {
	if fuel == 0 {
		return 10 * time.Second
	}
	// Map fuel units to a wall-clock budget; capped to avoid unbounded waits.
	timeout := time.Duration(fuel/1_000_000) * time.Millisecond
	if timeout < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	if timeout > 30*time.Second {
		return 30 * time.Second
	}
	return timeout
}

func ceilPages(bytes uint32) uint32 {
	pages := bytes / 65536
	if bytes%65536 != 0 {
		pages++
	}
	if pages == 0 {
		return 1
	}
	return pages
}

func getResPathHost(ctx context.Context, mod api.Module, bufPtr, maxLen uint32) uint32 {
	host, ok := wasmHostFromContext(ctx)
	if !ok {
		return uint32(^uint32(0) >> 1)
	}
	return writeGuestString(mod, bufPtr, maxLen, host.resPath)
}

func getCDNPathHost(ctx context.Context, mod api.Module, bufPtr, maxLen uint32) uint32 {
	host, ok := wasmHostFromContext(ctx)
	if !ok {
		return uint32(^uint32(0) >> 1)
	}
	return writeGuestString(mod, bufPtr, maxLen, host.cdnPath)
}

func getPlatformHost(ctx context.Context, _ api.Module) uint32 {
	host, ok := wasmHostFromContext(ctx)
	if !ok {
		return uint32(^uint32(0) >> 1)
	}
	return uint32(host.platform)
}

func writeGuestString(mod api.Module, bufPtr, maxLen uint32, value []byte) uint32 {
	if len(value) == 0 {
		return 0
	}
	if bufPtr == 0 || maxLen < uint32(len(value)) {
		return uint32(^uint32(0) >> 1)
	}
	mem := mod.Memory()
	if mem == nil {
		return uint32(^uint32(0) >> 1)
	}
	if ok := mem.Write(bufPtr, value); !ok {
		return uint32(^uint32(0) >> 1)
	}
	return uint32(len(value))
}

type wasmRunner struct {
	module *wasmModule
	mu     sync.Mutex
}

func newWasmRunner(baseDir string, cfg wasmConfig) (*wasmRunner, error) {
	module, err := newWasmModule(baseDir, cfg)
	if err != nil {
		return nil, err
	}
	return &wasmRunner{module: module}, nil
}

func (r *wasmRunner) transform(ctx context.Context, in Input) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.module.transform(ctx, in)
}

func (r *wasmRunner) close(ctx context.Context) error {
	if r == nil || r.module == nil {
		return nil
	}
	return r.module.Close(ctx)
}
