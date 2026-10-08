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

// ExampleSpec shows how one DTO method feeds both runtime validation and static
// field metadata. The template that renders OpenAPI consumes the same Spec, so
// the document cannot drift from the rules.
func ExampleSpec() {
	type createUserRequest struct {
		Name  string
		Email string
		Age   int
	}

	spec := func(r createUserRequest) validate.Spec {
		return validate.Spec{
			validate.String("name", r.Name, validate.NonEmpty(), validate.MaxRunes(20)),
			validate.String("email", r.Email,
				validate.WithMessage("邮箱格式不正确", validate.EmailFormat()),
			),
			validate.Int("age", r.Age, validate.MinValue(18), validate.MaxValue(120)),
		}
	}

	fmt.Println(spec(createUserRequest{Name: "", Email: "bad", Age: 10}).Validate())

	schema, _ := spec(createUserRequest{}).Schema()
	for _, field := range schema.Fields {
		fmt.Printf("%s type=%s required=%v\n", field.Name, field.Type, field.Required)
	}

	// Output:
	// name: 不能为空; email: 邮箱格式不正确; age: 不能小于 18
	// name type=string required=true
	// email type=string required=true
	// age type=integer required=true
}
