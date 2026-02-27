

# Kratos × Zerolog 适配器说明（按日 + 按大小滚动，gzip，双路输出）

本适配器将 **zerolog** 封装为 **Kratos `log.Logger`**，满足以下需求：

- **双路输出**：文件（JSON）+ 控制台（pretty）
- **滚动策略**：**每天至少滚动一次**（本地时区午夜），且支持**按大小**滚动（保证单文件不超过上限）
- **命名规则**：
    - 当天首个：`region.log.YYYYMMDD`
    - 当天因大小触发的第 *n* 个：`region.log.YYYYMMDD.n`（第二个是 `.2`）
    - 压缩后：`region.log.YYYYMMDD.gz`、`region.log.20250728.2.gz`
- **压缩**：滚动后自动 **gzip**
- **保留数量**：仅统计 `.gz`，超出 `MaxBackups` 删除最旧
- **软链**：始终创建 `BaseFilename` → “当前活动文件”的**软链**
- **控制台**：人类可读的 **pretty** 样式

> 注：当 `BaseFilename` 为空时，不写文件，仅输出到控制台（pretty），此时不涉及滚动/压缩/保留数量/软链。

---

## 配置项（Options）与默认值

| 选项名 | 类型/示例 | 默认值 | 说明 |
|---|---|---|---|
| `BaseFilename` | `/var/log/region.log` 或空字符串 | 空字符串（仅控制台） | 文件输出基础名（含路径与主体名，不含日期/序号/后缀）。非空即启用文件输出与滚动。空字符串则只输出控制台。 |
| `MaxSizeBytes` | `104857600`（100MB） | `100MB` | 单个**活动文件**的最大大小；写入将超限时执行“**同日按大小**”滚动，序号 `n` 自增（从 `2` 开始）。 |
| `MaxBackups` | `7` | `7` | 仅对**已压缩**的 `.gz` 文件计数；超过该数量时清理最旧的 `.gz`。`<=0` 表示不限制。 |
| `Compress` | `true/false` | `true` | 滚动后是否对旧文件进行 gzip 压缩。 |
| `ForceDailyRollover` | `true/false` | `true` | 本地时区午夜（00:00:00）**强制按日滚动**，即使无写入。 |
| `Location` | `time.Local` 或指定时区 | 本地时区 | 用于确定“午夜”与日期格式化的时区。 |
| `ConsolePretty` | `true/false` | `true` | 控制台输出采用 pretty（人类可读）。 |
| `ConsoleToStderr` | `true/false` | `false` | 控制台输出目标：`false` 为 `stdout`，`true` 为 `stderr`。 |
| `TimeFieldFormat` | `"2006-01-02 15:04:05"` | `"2006-01-02 15:04:05"` | 时间字段格式；**文件 JSON 与控制台**统一使用该格式。 |

> 软链：当 `BaseFilename` 非空时，始终创建软链 `BaseFilename -> BaseFilename.YYYYMMDD[.n]`，以便 `tail -f` 始终跟随当前活动文件。若系统不支持软链（例如某些 Windows 环境）或权限不足，将忽略创建失败而不影响日志写入。

---

## 命名规则与示例

- **当天首个文件**：`<BaseFilename>.YYYYMMDD`  
  例：`/var/log/region.log.20250728`
- **同日按大小滚动的第 n 个文件**：`<BaseFilename>.YYYYMMDD.n`（`n` 从 2 开始）  
  例：`/var/log/region.log.20250728.2`
- **滚动后压缩产物**：追加 `.gz`  
  例：`/var/log/region.log.20250728.gz`、`/var/log/region.log.20250728.2.gz`
- **软链（始终创建）**：`/var/log/region.log -> /var/log/region.log.20250728[.n]`

**一天内的文件演化（示意）**：

```bash
/var/log/
├─ region.log -> 软链，指向当前活动文件
├─ region.log.20250728 # 当天首个（无 .n）
├─ region.log.20250728.2 # 同日因大小滚动产生的第二个
├─ region.log.20250728.gz # 首个被后台压缩后的产物
├─ region.log.20250728.2.gz # 第二个被后台压缩后的产物
└─ region.log.20250729 # 午夜后新的一天，成为新的活动文件
```


---

## 滚动行为

### 按日滚动（保证每天至少一次）
- **触发时机**：本地时区午夜（00:00:00），即使当日**没有写入**也会切换到新文件。
- **效果**：当天活动文件关闭，后台进行 `.gz` 压缩与超量清理；新的一天从 `YYYYMMDD`（无 `.n`）开始写入。

### 按大小滚动（限制单文件体积）
- **触发条件**：即将写入使活动文件大小 **超过** `MaxSizeBytes`。
- **效果**：在**同一天**创建新的 `YYYYMMDD.n`（`n` 从 `2` 开始递增），旧文件后台压缩并参与保留数量清理。

### 清理策略（MaxBackups）
- 仅对 **`.gz` 压缩后的文件** 计数。
- 一旦 `.gz` 数量**超过** `MaxBackups`，将按**最旧优先**删除多余文件（基于文件名序与时间戳）。

---

## Demo 1：文件（JSON）+ 控制台（pretty）

**目标**
- 同时输出到**文件（JSON）**与**控制台（pretty）**。
- **每天至少滚动一次**，并在**超过单文件大小**时“同日序号”滚动。
- 滚动后**自动 `.gz` 压缩**，并按 `.gz` **保留数量**清理。
- 始终创建**软链**：`BaseFilename -> 当前活动文件`。

**推荐设置**
- `BaseFilename: /var/log/region.log`
- `MaxSizeBytes: 100MB`（默认即可，或按需调整）
- `MaxBackups: 7`（默认即可，或按需调整）
- `Compress: true`（默认）
- `ForceDailyRollover: true`（默认）
- `Location: 本地时区`（默认）
- `ConsolePretty: true`（默认）
- `ConsoleToStderr: 视需要`
- `TimeFieldFormat: "2006-01-02 15:04:05"`（默认）

**操作便捷性**
- **追踪当前活动文件**：`tail -f /var/log/region.log`（通过软链自动跟随）
- **按日筛查**：`ls /var/log/region.log.20250728*`

```golang
func Example_FileAndConsole() {
    logger, closeFn, err := New(Options{
    BaseFilename:       "/var/log/region.log", // 非空 => 写文件(JSON) + 控制台(pretty)
    MaxSizeBytes:       0,                     // 使用默认 100MB
    MaxBackups:         0,                     // 使用默认 7
    Compress:           true,                  // 默认 true，可省略
    ForceDailyRollover: true,                  // 默认 true，可省略
    // Location:         nil,  // 默认本地时区
    // ConsolePretty:    true, // 默认
    // TimeFieldFormat:  "2006-01-02 15:04:05", // 默认
    })
    if err != nil {
        panic(err)
    }
    klog.SetLogger(logger)
    klog.Info("msg", "hello")
    _ = closeFn(context.Background())
}
```

---

## Demo 2：仅控制台（pretty）

**目标**
- **不写文件**，仅在控制台打印**pretty**日志。
- 不涉及滚动/压缩/保留数量/软链。

**推荐设置**
- `BaseFilename: ""`（空字符串）
- `ConsolePretty: true`（默认）
- `ConsoleToStderr: 视需要`
- `TimeFieldFormat: "2006-01-02 15:04:05"`（默认）
- 其他与文件相关的选项（如 `MaxSizeBytes/MaxBackups/Compress/ForceDailyRollover/Location`）对“仅控制台”模式**无效**，可忽略。

**适用场景**
- 开发环境、本地调试、容器标准输出（由外部日志系统采集）。

```golang
func Example_ConsoleOnly() {
	logger, closeFn, err := New(Options{
		BaseFilename: "", // 为空 => 仅控制台 pretty
	})
	if err != nil {
		panic(err)
	}
	klog.SetLogger(logger)
	klog.Info("msg", "console only")
	_ = closeFn(context.Background())
}
```
---

## 常见问题（FAQ）

**Q1：为何不用 `lumberjack`？**  
A：`lumberjack` 的滚动是**按大小**，无法严格保证**每天**至少滚动一次；本适配器要求“按日 + 按大小”的复合策略及自定义命名与软链，故采用自实现以满足精确需求。

**Q2：`MaxBackups` 如何计算？**  
A：仅统计**压缩后的 `.gz`** 文件。活动文件（未压缩）不计入；当发生新滚动且压缩完成后，清理逻辑会删除最旧的超量 `.gz`。

**Q3：软链在 Windows 上可能失败怎么办？**  
A：创建软链需要权限与文件系统支持。在不满足条件时，创建失败会被**忽略**，不影响日志写入；你仍可直接 `tail -f <BaseFilename>.<YYYYMMDD>[.n]`。

**Q4：进程退出时的刷盘/压缩？**  
A：关闭时会 `sync + close` 当前活动文件，但不会强制将活动文件立即压缩（通常留待下一次滚动后再压缩）。若你需要“退出即压缩”，可在退出流程中**先触发一次滚动**（例如日期或大小条件），或按需扩展实现。

**Q5：时间格式统一吗？**  
A：是。通过 `TimeFieldFormat`（默认 `"2006-01-02 15:04:05"`）对**文件 JSON 与控制台**统一格式，便于对齐排查。

---

## 行为速查表

| 场景 | 触发 | 新文件命名 | 旧文件处理 | 软链 |
|---|---|---|---|---|
| 午夜到达（本地时区） | `ForceDailyRollover=true` | `Base.YYYYMMDD`（新的一天） | 异步 `.gz` 压缩 + 超量清理 | 指向新活动文件 |
| 写入将超出 `MaxSizeBytes` | 同日按大小滚动 | `Base.YYYYMMDD.n`（`n=2,3,...`） | 异步 `.gz` 压缩 + 超量清理 | 指向新活动文件 |
| 仅控制台模式 | `BaseFilename=""` | 无文件 | 无 | 无 |

---

## 兼容性与限制

- **权限**：确保日志目录具备写入权限；创建软链需要相应权限与文件系统支持。
- **高并发写入**：当前为同步写入（线程安全）；如需极高吞吐与抖动隔离，可在外层加入缓冲异步落盘（需权衡丢失风险与关闭时 drain）。
- **分卷写入**：当单次日志行超大接近阈值时，本适配器采用“滚动后再整体写入”的策略，不拆分单次写入。
