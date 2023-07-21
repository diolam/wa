<div align="center">
<h1>凹语言</h1>

[主页](https://wa-lang.org) | [Playground](https://wa-lang.org/playground) | [路线](https://wa-lang.org/smalltalk/st0001.html) | [社区](https://wa-lang.org/community) | [日志](https://wa-lang.org/guide/changelog.html)

</div>

凹语言（凹读音“Wā”）是 针对 WASM 平台设计的通用编程语言，同时支持 Linux、macOS 和 Windows 等主流操作系统和 Chrome 等浏览器环境，同时也支持作为独立 Shell 脚本和被嵌入脚本模式执行。

![](docs/images/logo/logo-animate1.svg)

- 主页: [https://wa-lang.org](https://wa-lang.org)
- 仓库(Gitee): [https://gitee.com/wa-lang/wa](https://gitee.com/wa-lang/wa)
- 仓库(Github): [https://github.com/wa-lang/wa](https://github.com/wa-lang/wa)
- Playground: [https://wa-lang.org/playground](https://wa-lang.org/playground)

> 说明: 除非特别声明，凹语言代码均以 AGPL-v3 开源协议授权, 具体可以参考 LICENSE 文件。

## 如何参与开发

项目尚处于原型开源阶段，如果有共建和PR需求请参考 [如何贡献代码](https://wa-lang.org/community/contribute.html)。我们不再接受针对第三方依赖库修改的 PR。

> 特别注意：向本仓库提交PR视同您认可并接受[凹语言贡献者协议](https://gitee.com/organizations/wa-lang/cla/wca)，但在实际签署之前，您的PR不会被评审或接受。


## Playground 在线预览

[https://wa-lang.org/playground](https://wa-lang.org/playground)

![](https://wa-lang.org/playground-01.png)

## 贪吃蛇游戏

- [https://wa-lang.org/wa/snake/](https://wa-lang.org/wa/snake/)
- [https://wa-lang.org/smalltalk/st0018.html](https://wa-lang.org/smalltalk/st0018.html)

![](https://wa-lang.org/st0018-03.jpg)

## 本地安装和测试:

有 2 种安装方式: 1 是直接从 Github [下载最新的二进制文件](https://github.com/wa-lang/wa/releases); 2 是通过 Go 语言从源码安装.

通过 Go 语言安装需要执行以下命令 (Go >= 1.17):

1. `go install wa-lang.org/wa@latest`
2. `cd waroot`
3. `wa init -name=examples/hi`
4. `wa run examples/hi`

## 例子: 凹语言

打印字符和调用函数：

```wa
import "fmt"

global year: i32 = 2023

func main {
	println("你好，凹语言！")
	println(add(40, 2), year)

	fmt.Println(1+1)
}

func add(a: i32, b: i32) => i32 {
	return a+b
}
```

运行并输出结果:

```
$ go run main.go hello.wa 
你好，凹语言！
42 2023
2
```

## 例子: 打印素数

打印 30 以内的素数:

```wa
// 版权 @2021 凹语言™ 作者。保留所有权利。

func main {
	for n := 2; n <= 30; n = n + 1 {
		isPrime: int = 1
		for i := 2; i*i <= n; i = i + 1 {
			if x := n % i; x == 0 {
				isPrime = 0
			}
		}
		if isPrime != 0 {
			println(n)
		}
	}
}
```

运行并输出结果:

```
$ cd waroot && go run ../main.go run examples/prime
2
3
5
7
11
13
17
19
23
29
```

## 例子：用中文语法打印素数

```wz
引于 "书"

【启】：
  // 打印30以内的素数
  从n=2，到n>30，有n++：
    设素=1
    从i=2，到i*i>n，有i++：
      设x=n%i
      若x==0则：
        素=0
      。
    。
    若素!=0则：
      书·曰：n
    。
  。
。
```

运行的结果和英文语法的示例相同。

更多例子 [waroot/examples](waroot/examples)

## 贡献者名单

|贡献者|贡献点|
| --- | --- |
|柴树杉| 35000|
|丁尔男| 42500|
|史斌  | 29000|
|扈梦明| 14000|
|赵普明| 17000|
|宋汝阳|  2000|
|刘云峰|  1000|
|王湘南|  1000|
|王泽龙|  1000|
|吴烜  |  3000|
|刘斌  |  2500|
|尹贻浩|  2000|

贡献点变更记录见 [waroot/cplog](waroot/cplog) 目录。
