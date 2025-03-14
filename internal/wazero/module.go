// 版权 @2023 凹语言 作者。保留所有权利。

package wazero

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"wa-lang.org/wa/internal/config"
	"wa-lang.org/wazero"
	"wa-lang.org/wazero/api"
)

// wasm 模块, 可多次执行
type Module struct {
	cfg *config.Config

	wasmName  string
	wasmBytes []byte
	wasmArgs  []string

	stdoutBuffer bytes.Buffer
	stderrBuffer bytes.Buffer

	wazeroOnce          sync.Once
	wazeroCtx           context.Context
	wazeroConf          wazero.ModuleConfig
	wazeroRuntime       wazero.Runtime
	wazeroCompileModule wazero.CompiledModule
	wazeroModule        api.Module
	wazeroInitErr       error
}

// 构建模块(会执行编译)
func BuildModule(
	cfg *config.Config, wasmName string, wasmBytes []byte, wasmArgs ...string,
) (*Module, error) {
	m := &Module{
		cfg:       cfg,
		wasmName:  wasmName,
		wasmBytes: wasmBytes,
		wasmArgs:  wasmArgs,
	}
	if err := m.buildModule(); err != nil {
		return nil, err
	}
	return m, nil
}

// 执行初始化, 仅执行一次
func (p *Module) RunMain() (stdout, stderr []byte, err error) {
	p.wazeroModule, p.wazeroInitErr = p.wazeroRuntime.InstantiateModule(
		p.wazeroCtx, p.wazeroCompileModule, p.wazeroConf,
	)

	stdout = p.stdoutBuffer.Bytes()
	stderr = p.stderrBuffer.Bytes()
	err = p.wazeroInitErr
	return
}

// 执行指定函数(init会被强制执行一次)
func (p *Module) RunFunc(name string, args ...uint64) (result []uint64, stdout, stderr []byte, err error) {
	if p.wazeroModule == nil {
		p.wazeroModule, p.wazeroInitErr = p.wazeroRuntime.InstantiateModule(
			p.wazeroCtx, p.wazeroCompileModule, p.wazeroConf,
		)
	}
	if p.wazeroInitErr != nil {
		stdout = p.stdoutBuffer.Bytes()
		stderr = p.stderrBuffer.Bytes()
		err = p.wazeroInitErr
		return
	}

	p.stdoutBuffer.Reset()
	p.stderrBuffer.Reset()
	fn := p.wazeroModule.ExportedFunction(name)
	if fn == nil {
		err = fmt.Errorf("wazero: func %q not found", name)
		return
	}

	result, err = fn.Call(p.wazeroCtx, args...)
	stdout = p.stdoutBuffer.Bytes()
	stderr = p.stderrBuffer.Bytes()
	return
}

// 关闭模块
func (p *Module) Close() error {
	var err error
	if p.wazeroRuntime != nil {
		err = p.wazeroRuntime.Close(p.wazeroCtx)
		p.wazeroRuntime = nil
	}
	return err
}

func (p *Module) buildModule() error {
	p.wazeroCtx = context.Background()

	p.wazeroConf = wazero.NewModuleConfig().
		WithStdout(&p.stdoutBuffer).
		WithStderr(&p.stderrBuffer).
		WithStdin(os.Stdin).
		WithRandSource(rand.Reader).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithArgs(append([]string{p.wasmName}, p.wasmArgs...)...).
		WithName(p.wasmName)

	// TODO: Windows 可能导致异常, 临时屏蔽
	if runtime.GOOS != "windows" {
		for _, s := range os.Environ() {
			var key, value string
			if kv := strings.Split(s, "="); len(kv) >= 2 {
				key = kv[0]
				value = kv[1]
			} else if len(kv) >= 1 {
				key = kv[0]
			}
			p.wazeroConf = p.wazeroConf.WithEnv(key, value)
		}
	}

	p.wazeroRuntime = wazero.NewRuntime(p.wazeroCtx)

	var err error
	p.wazeroCompileModule, err = p.wazeroRuntime.CompileModule(p.wazeroCtx, p.wasmBytes)
	if err != nil {
		p.wazeroInitErr = err
		return err
	}

	switch p.cfg.WaOS {
	case config.WaOS_arduino:
		if _, err = ArduinoInstantiate(p.wazeroCtx, p.wazeroRuntime); err != nil {
			p.wazeroInitErr = err
			return err
		}
	case config.WaOS_chrome:
		if _, err = ChromeInstantiate(p.wazeroCtx, p.wazeroRuntime); err != nil {
			p.wazeroInitErr = err
			return err
		}
	case config.WaOS_wasi:
		if _, err = WasiInstantiate(p.wazeroCtx, p.wazeroRuntime); err != nil {
			p.wazeroInitErr = err
			return err
		}
	case config.WaOS_mvp:
		if _, err = MvpInstantiate(p.wazeroCtx, p.wazeroRuntime); err != nil {
			p.wazeroInitErr = err
			return err
		}

	default:
		return fmt.Errorf("unknown waos: %q", p.cfg.WaOS)
	}

	return nil
}
