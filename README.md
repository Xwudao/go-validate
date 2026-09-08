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

## 设计约定

- **纯函数**：规则只读取传入的值，不访问外部状态、不发起 IO。业务逻辑放到 service 层。
- **无反射 / 无 tag**：不解析 struct tag，字段名和值都显式传入。
- **首错即停**：每个字段遇到第一条失败规则即停止，避免一个字段返回多条冗余文案。
- **文案可覆盖**：内置文案是中文，用 `Message` 可针对接口定制。
