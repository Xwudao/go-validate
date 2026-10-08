# go-validate

轻量、无反射、代码优先的 Go 校验库。规则就是普通函数，编译器可以帮你检查；
不依赖任何 Web 框架、数据库或业务服务，任何项目都能直接引用并按需扩展。

> 该库从 `go-hunhepan/internal/validate` 抽取而来，内置规则、错误文案与语义
> 完全保持一致，可直接替换原有用法。

## 安装

```bash
go get github.com/Xwudao/go-validate
```

## 快速开始

```go
import "github.com/Xwudao/go-validate"

type CreateUserParams struct {
    Name  string
    Email string
    Age   int
}

func (p CreateUserParams) Validate() error {
    return validate.Validate(
        validate.Field("name", p.Name,
            validate.Message("名称不能为空", validate.Required()),
            validate.Message("名称不能超过 20 字", validate.MaxLen(20)),
        ),
        validate.Field("email", p.Email,
            validate.Message("邮箱不能为空", validate.Required()),
            validate.Message("邮箱格式不正确", validate.Email()),
        ),
        validate.Field("age", p.Age,
            validate.Min(18),
            validate.Max(120),
        ),
    )
}
```

`Validate` 会逐字段执行规则，每个字段遇到第一条失败的规则就停止，最终返回
`validate.Errors`（实现了 `error`）：

```
name: 名称不能为空; email: 邮箱格式不正确
```

## 核心概念

| 名称 | 说明 |
| --- | --- |
| `Rule[T]` | `func(T) error`，校验一个值，返回 `nil` 表示通过 |
| `Field(name, value, rules...)` | 声明一个字段和它的规则，按顺序执行，遇错即停 |
| `Validate(fields...)` | 汇总每个字段的第一条错误，全部通过返回 `nil` |
| `FieldError` | `{Field, Message}`，单个字段的错误 |
| `Errors` | `[]FieldError`，聚合结果 |

`Errors` 提供了一些便利方法：

```go
err := params.Validate()
var verrs validate.Errors
if errors.As(err, &verrs) {
    verrs.Error()          // "name: 不能为空; age: 不能小于 18"
    verrs.First()          // 第一条错误信息（不含字段名），适合只返回单条文案的老接口
    verrs.For("age")       // *FieldError
    verrs.Has("age")       // bool
    verrs.Fields()         // []string{"name", "age"}
    verrs.Messages()       // []string{"不能为空", "不能小于 18"}
}
```

## 内置规则

### 通用 / 比较

| 规则 | 说明 | 默认文案 |
| --- | --- | --- |
| `Required()` | 字符串非空 | `不能为空` |
| `NotBlank()` | 去除首尾空白后非空 | `不能为空` |
| `NotZero[T]()` | 非零值（含指针 nil） | `不能为空` |
| `Min(v)` / `Max(v)` | 数值上下界 | `不能小于 v` / `不能大于 v` |
| `Between(min, max)` | 闭区间 | `必须在 min 和 max 之间` |
| `Positive[T]()` / `NonNegative[T]()` / `Negative[T]()` / `NonPositive[T]()` | 正负号 | `必须大于 0` 等 |
| `Eq(v)` / `Ne(v)` | 等于 / 不等于 | `必须等于 v` / `不能等于 v` |
| `OneOf(...)` / `NotOneOf(...)` | 枚举 | `值不合法` |

### 字符串

| 规则 | 默认文案 |
| --- | --- |
| `MinLen(n)` / `MaxLen(n)` / `Len(n)` | `长度不能小于 n` 等 |
| `StartsWith(s)` / `EndsWith(s)` | `必须以 s 开头` / `必须以 s 结尾` |
| `Contains(s)` / `NotContains(s)` | `必须包含 s` / `不能包含 s` |
| `Match(re)` / `MatchString(pattern)` | `格式不正确` |
| `Alpha()` / `AlphaNum()` / `ASCII()` | `只能包含字母` 等 |
| `Lowercase()` / `Uppercase()` | `只能包含小写字母` 等 |
| `NoWhitespace()` / `Trimmed()` | `不能包含空白字符` 等 |

### 格式

| 规则 | 默认文案 |
| --- | --- |
| `Email()` / `URL()` | `格式不正确` |
| `IP()` / `IPv4()` / `IPv6()` / `CIDR()` | `IP格式不正确` 等 |
| `Port()` | `端口不合法` |
| `UUID()` | `UUID格式不正确` |
| `Hex()` / `Base64()` / `JSON()` | `只能包含十六进制字符` 等 |
| `Phone()` | `手机号格式不正确` |
| `Hostname()` / `MAC()` | `主机名格式不正确` / `MAC地址格式不正确` |
| `Date()` | `日期格式不正确` |
| `DateTime(layout)` | `时间格式不正确，应为 layout` |
| `Duration()` | `时长格式不正确` |
| `Numeric()` | `只能包含数字` |

### 集合

| 规则 | 默认文案 |
| --- | --- |
| `MinItems(n)` / `MaxItems(n)` / `LenItems(n)` | `数量不能小于 n` 等 |
| `NonEmptyMap[K, V]()` | `不能为空` |
| `Unique[T]()` | `不能包含重复项 v` |
| `Each[T](rules...)` | `第 N 项<规则文案>` |

### 组合器

| 组合器 | 说明 |
| --- | --- |
| `Message(msg, rule)` | 替换规则的错误文案，保留校验逻辑 |
| `When(cond, rule)` | 条件为真时才校验 |
| `Optional(rule)` | `*T` 为 nil 时跳过，否则校验解引用后的值 |
| `All(rules...)` | 组合成一条规则，返回第一个错误 |
| `Any(rules...)` | 任一通过即通过 |
| `Not(rule, msg)` | 取反 |
| `Func(fn)` | 把普通函数转成 `Rule[T]` |

## 自定义规则（重要）

`Rule[T]` 是导出类型，**任何依赖该库的项目都可以直接新增规则**，无需修改本库、
无需提 PR。推荐在业务项目里放一个 `internal/validatex` 之类的包：

```go
package validatex

import (
    "errors"
    "github.com/Xwudao/go-validate"
)

// Even 必须是偶数。
func Even() validate.Rule[int] {
    return func(value int) error {
        if value%2 != 0 {
            return errors.New("必须是偶数")
        }
        return nil
    }
}

// MustEqual 必须等于指定值。
func MustEqual(want string) validate.Rule[string] {
    return func(value string) error {
        if value != want {
            return errors.New("两次输入不一致")
        }
        return nil
    }
}
```

使用方式与内置规则完全一致，并且可以和组合器混用：

```go
validate.Field("count", count, validatex.Even())
validate.Field("confirm", confirm, validatex.MustEqual(password))
validate.Field("nickname", nickname, validate.Optional(validate.MaxLen(16)))
```

一次性规则可以用 `validate.Func`：

```go
validate.Field("code", code, validate.Func(func(value string) error {
    if len(value) != 6 {
        return errors.New("验证码必须是 6 位")
    }
    return nil
}))
```

跨字段校验也只需把别的字段捕获进闭包：

```go
start := params.StartAt
validate.Field("end_at", params.EndAt, validate.Func(func(end time.Time) error {
    if end.Before(start) {
        return errors.New("结束时间不能早于开始时间")
    }
    return nil
}))
```

## 机器可读元数据（Constraint / Spec / Schema）

校验规则和接口文档常常是两份手写清单，容易「代码改了、文档没改」。
`Constraint[T]` 把一条运行时规则和**同一条件**的元数据放在同一个构造函数里；
`Spec` 用同一份声明同时产出**运行时校验**和**机器可读字段元数据**，两者不会漂移。

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
    Role  string `json:"role"`
}

// Spec 是唯一的声明入口。
func (r CreateUserRequest) Spec() validate.Spec {
    return validate.Spec{
        validate.String("name", r.Name,
            validate.NonEmpty(),
            validate.MaxRunes(50),
        ),
        validate.String("email", r.Email,
            validate.WithMessage("邮箱格式不正确", validate.EmailFormat()),
        ),
        validate.Int("age", r.Age,
            validate.MinValue(18),
            validate.MaxValue(120),
        ),
        validate.String("role", r.Role,
            validate.Enum("admin", "member"),
        ),
    }
}

func (r CreateUserRequest) Validate() error { return r.Spec().Validate() }

// Schema 不执行任何规则，可以在零值上调用。
func (r CreateUserRequest) Schema() (validate.Schema, error) {
    return r.Spec().Schema()
}
```

`Schema.Fields` 是 JSON-Schema 中立的字段描述：`Name`、`Type`（`TypeString` /
`TypeInteger` / `TypeNumber`）、`Required`、`Nullable`、`MinLength`、`MaxLength`、
`Minimum`、`Maximum`、`Format`、`Enum`、`Unsupported`、`Invalid`。`Schema.Names()` 返回字段名，
消费方（模板）可以据此和请求结构体的 JSON 标签比对，确保名字与 JSON 形状一致。

### 约束一览（MVP）

| 约束 | 运行时规则 | 元数据 |
| --- | --- | --- |
| `NonEmpty()` | 字符串非空 | `required` + `minLength: 1` |
| `MinRunes(n)` / `MaxRunes(n)` | 按 **Unicode 码点**（rune）计数 | `minLength` / `maxLength` |
| `EmailFormat()` | RFC 5322 邮箱 | `format: email` + `required` |
| `Enum(values...)` | 枚举 | `enum`（零值不在集合内时为 `required`） |
| `MinValue(n)` / `MaxValue(n)` | 数值上下界（`int` / `float64`） | `minimum` / `maximum` |
| `WithMessage(msg, c)` | 替换文案 | **保留**内层全部元数据 |
| `Conditional(cond, c)` | 条件为真才校验 | 标为 unsupported（见下） |
| `Custom(rule)` | 自定义规则 | 标为 unsupported（见下） |

### 必填 / presence 的差别

- **值类型字段**（`String` / `Int` / `Float`）：JSON 绑定后「缺省」和「零值」无法区分，
  因此只要某条约束会拒绝零值（如 `NonEmpty`、`MinValue(1)`、不含 `""` 的 `Enum`），
  schema 就会输出 `required: true`。这正是运行时行为，不会出现文档说可选、运行时必填的错配。
- **可选指针字段**（`OptionalString` / `OptionalInt` / `OptionalFloat`）：`nil` 表示缺省或
  `null`，此时跳过全部约束，因此默认 `required: false`、`nullable: true`；约束只在「存在」时生效。
  需要必填时显式调用 `.Required()`（会拒绝 `nil`，同时 `nullable` 变为 `false`）：

  ```go
  validate.OptionalString("nickname", r.Nickname).Required().MaxRunes(16)
  ```

### 无法表达的约束：明确报错，不静默漏写

`Conditional`（动态条件）和 `Custom`（无元数据的自定义规则）在运行时照常校验，
但**无法无条件写进文档**。`Spec.Schema()` 会返回一个 `validate.UnsupportedError`，
逐字段列出原因；已支持的字段元数据仍会返回，消费方可自行决定是让开发者改用显式约束、
还是手写该字段的文档：

```go
schema, err := req.Spec().Schema()
if err != nil {
    // 例如：unsupported constraints: company: conditional constraint has no unconditional machine-readable form
    var unsupported validate.UnsupportedError
    if errors.As(err, &unsupported) {
        // 交给模板提示开发者补充该字段文档
    }
}
```

注意：`Spec` 方法必须是**无条件**的——字段集合和约束不能依赖接收者的值。
需要条件校验时请使用 `Conditional`，让它以 unsupported 的形式显式暴露，而不是被静默漏掉。

### 非法参数 / 重复字段：同样 fail-closed

构造参数无法构成合法约束时不会 panic，也不会静默降级：`MinRunes`/`MaxRunes` 的负边界、
空的 `Enum()`、非有限（`NaN`/`±Inf`）的 `MinValue`/`MaxValue`，以及相互矛盾的上下界
（如 `MinRunes(5), MaxRunes(3)`）。这类约束的运行时规则**拒绝一切取值**（fail closed），
`Schema()` 返回 `validate.InvalidError` 并逐字段列出原因，绝不输出非法的 JSON Schema：

```go
schema, err := req.Spec().Schema()
if err != nil {
    var invalid validate.InvalidError
    if errors.As(err, &invalid) {
        // 例如：invalid constraints: s: MinRunes requires min >= 0, got -1
    }
}
```

`Spec.Schema()` 还会校验字段名：空字段名和重复字段名都以 `InvalidError` 报错，
避免生成含重复属性的文档。`UnsupportedError` 与 `InvalidError` 可能同时出现，
此时返回的错误可用 `errors.As` 分别取出。

此外，`Constraint.Meta()` 与 `Schema()` 返回的都是**防御性拷贝**（拥有自己的
`*int`、`[]any` 与原因切片），消费方改动导出结果不会回写 `Spec`，
因此文档元数据不会和运行时规则漂移。

### 未来模板集成（可复制）

`go-validate` 不导入任何 HTTP / OpenAPI 依赖；元数据落在 `Schema` 上，
脚手架模板（如 weld / weld-template 的 `add api`）可以直接读取并渲染约束：

```go
// renderOpenAPIConstraints 由模板调用，把 Spec 转成 OpenAPI 字段约束。
func renderOpenAPIConstraints(spec validate.Spec) (map[string]any, error) {
    schema, err := spec.Schema()
    if err != nil {
        return nil, err // 有无法表达的约束：拒绝生成，避免文档与运行时不符
    }

    properties := make(map[string]any, len(schema.Fields))
    for _, field := range schema.Fields {
        prop := map[string]any{"type": field.Type}
        if field.MinLength != nil {
            prop["minLength"] = *field.MinLength
        }
        if field.MaxLength != nil {
            prop["maxLength"] = *field.MaxLength
        }
        if field.Minimum != nil {
            prop["minimum"] = field.Minimum
        }
        if field.Maximum != nil {
            prop["maximum"] = field.Maximum
        }
        if field.Format != "" {
            prop["format"] = field.Format
        }
        if field.Enum != nil {
            prop["enum"] = field.Enum
        }
        properties[field.Name] = prop
    }

    // 消费方可用 schema.Names() 与请求结构体的 JSON 标签核对字段名。
    required := make([]string, 0, len(schema.Fields))
    for _, field := range schema.Fields {
        if field.Required {
            required = append(required, field.Name)
        }
    }
    return map[string]any{"properties": properties, "required": required}, nil
}
```

## 设计约定

- **纯函数**：规则只读取传入的值，不访问外部状态、不发起 IO。业务逻辑放到 service 层。
- **无反射 / 无 tag**：不解析 struct tag，字段名和值都显式传入。
- **首错即停**：每个字段遇到第一条失败规则即停止，避免一个字段返回多条冗余文案。
- **文案可覆盖**：内置文案是中文，用 `Message` 可针对接口定制。
