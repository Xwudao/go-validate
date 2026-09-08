package validate_test

import (
	"errors"
	"fmt"

	"github.com/Xwudao/go-validate"
)

// even is a custom rule defined entirely outside the library, proving that a
// dependent project can extend validation without changing go-validate.
func even() validate.Rule[int] {
	return func(value int) error {
		if value%2 != 0 {
			return errors.New("必须是偶数")
		}
		return nil
	}
}

func Example() {
	err := validate.Validate(
		validate.Field("name", "",
			validate.Message("名称不能为空", validate.Required()),
		),
		validate.Field("count", 3, even()),
	)
	fmt.Println(err)
	// Output: name: 名称不能为空; count: 必须是偶数
}

func ExampleErrors_First() {
	err := validate.Validate(
		validate.Field("age", 1, validate.Min(18)),
	)
	fmt.Println(err.(validate.Errors).First())
	// Output: 不能小于 18
}
