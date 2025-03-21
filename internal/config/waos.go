// 版权 @2022 凹语言 作者。保留所有权利。

package config

// wa 代码:
// xxx.wa
// xxx_$OS.wa
// xxx_$ARCH.wa
// xxx_$OS_$ARCH.wa

// wz 中文代码:
// xxx.wz
// xxx_$OS.wz
// xxx_$ARCH.wz
// xxx_$OS_$ARCH.wz

// ws 汇编代码:
// xxx.$BACKEND.ws
// xxx_$OS.$BACKEND.ws
// xxx_$ARCH.$BACKEND.ws
// xxx_$OS_$ARCH.$BACKEND.ws

// 编译器后端类型
const (
	WaBackend_Default = WaBackend_wat // 默认

	WaBackend_clang = "clang" // 输出 c
	WaBackend_llvm  = "llvm"  // 输出 llir
	WaBackend_wat   = "wat"   // 输出 wat
)

// 目标平台类型, 可管理后缀名
const (
	WaOS_Default = WaOS_wasi // 默认

	WaOS_arduino = "arduino" // Arduino 平台
	WaOS_chrome  = "chrome"  // Chrome 浏览器
	WaOS_wasi    = "wasi"    // WASI 接口
	WaOS_mvp     = "mvp"     // MVP 接口, 最小可用
)

// 体系结构类型
const (
	WaArch_Default = WaArch_wasm // 默认

	WaArch_386     = "386"     // 386 平台
	WaArch_amd64   = "amd64"   // amd64 平台
	WaArch_arm64   = "arm64"   // arm64 平台
	WaArch_riscv64 = "riscv64" // riscv64 平台
	WaArch_wasm    = "wasm"    // wasm 平台
)

// 后端列表
var WaBackend_List = []string{
	WaBackend_clang,
	WaBackend_llvm,
	WaBackend_wat,
}

// OS 列表
var WaOS_List = []string{
	WaOS_arduino,
	WaOS_chrome,
	WaOS_wasi,
	WaOS_mvp,
}

// CPU 列表
var WaArch_List = []string{
	WaArch_386,
	WaArch_amd64,
	WaArch_arm64,
	WaArch_riscv64,
	WaArch_wasm,
}

// 检查 OS 值是否 OK
func CheckWaOS(os string) bool {
	for _, x := range WaOS_List {
		if x == os {
			return true
		}
	}
	return false
}
