# singularity

[![Go Reference](https://pkg.go.dev/badge/github.com/jeffinity/singularity.svg)](https://pkg.go.dev/github.com/jeffinity/singularity)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jeffinity/singularity)](https://github.com/jeffinity/singularity/blob/main/go.mod)
[![License](https://img.shields.io/github/license/jeffinity/singularity)](https://github.com/jeffinity/singularity/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/jeffinity/singularity)](https://goreportcard.com/report/github.com/jeffinity/singularity)
[![Test Status](https://github.com/jeffinity/singularity/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/jeffinity/singularity/actions/workflows/test.yml?query=branch%3Amain)
[![codecov](https://codecov.io/gh/jeffinity/singularity/branch/main/graph/badge.svg)](https://codecov.io/gh/jeffinity/singularity)

`singularity` 是一个面向 Go + Kratos 微服务的工具库集合，聚焦于服务治理、日志、数据库和常用基础能力封装。

## 特性

- 基于 Nacos 的注册发现与配置接入（`nacosx`）
- Kratos 相关增强能力（`kratosx`）
- Zerolog + Kratos 日志适配与按日/按大小滚动（`logx`）
- Gorm + PostgreSQL 初始化与辅助类型（`pgx`）
- 自动化迁移封装（`migratex`）
- 泛型集合工具（`set`）
- 友好工具函数（`friendly`）
- 启动内置 pprof 监听（`pprof`）
- 构建信息变量注入（`buildinfo`）

## 安装

```bash
go get github.com/jeffinity/singularity@latest
```

## 模块说明

| 包名 | 说明 |
| --- | --- |
| `buildinfo` | 统一管理构建版本、提交信息等变量 |
| `friendly` | 常用友好函数（默认值、时间格式化、资源关闭） |
| `kratosx` | Kratos 生态扩展（连接工厂、endpoint 解析、codec 等） |
| `logx` | Zerolog 适配 Kratos，并支持日志滚动与压缩 |
| `migratex` | 基于 Gorm 的表迁移器封装 |
| `nacosx` | Nacos 命名服务、配置中心和注册发现封装 |
| `pgx` | PostgreSQL 连接初始化、Gorm 日志、JSONB 类型辅助 |
| `pprof` | 后台随机端口启动 pprof 服务 |
| `set` | 泛型 Set 能力与集合运算 |

## 目录结构

```text
.
├── buildinfo
├── friendly
├── kratosx
├── logx
├── migratex
├── nacosx
├── pgx
├── pprof
└── set
```

## 开发

```bash
go test ./...
```

建议 Go 版本与 `go.mod` 保持一致（当前为 `go 1.25.5`）。
