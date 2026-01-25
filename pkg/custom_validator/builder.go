// Package customvalidator provides a builder for setting up custom validation rules with translation support.
package customvalidator

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/pkg/utils"
	"fmt"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

type ValidatorBuilder struct {
	Validate *validator.Validate
	Trans    ut.Translator
}

func NewValidatorBuilder() *ValidatorBuilder {
	// 1. Setup Translator
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator("en")

	// 2. Setup Validator
	v := validator.New()

	// 3. Register Default Translations (e.g., required, min, max)
	_ = en_translations.RegisterDefaultTranslations(v, trans)

	// 4. (Optional) Use JSON tag names instead of Struct Field Names in errors
	// v.RegisterTagNameFunc(func(fld reflect.StructField) string {
	// 	name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
	// 	if name == "-" {
	// 		return ""
	// 	}
	// 	return name
	// })

	return &ValidatorBuilder{
		Validate: v,
		Trans:    trans,
	}
}

func (vb *ValidatorBuilder) AddCustomRule(tag string, fn validator.Func, msg string) *ValidatorBuilder {
	// 1. Register the validation logic
	err := vb.Validate.RegisterValidation(tag, fn)
	if err != nil {
		panic(fmt.Sprintf("failed to register validation tag '%s': %s", tag, err))
	}

	// 2. Register the translation logic
	// We use a generic translation function that handles {0} (field) and {1} (param)
	err = vb.Validate.RegisterTranslation(tag, vb.Trans, func(ut ut.Translator) error {
		return ut.Add(tag, msg, true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		// This generic function applies the values to the template
		t, _ := ut.T(tag, fe.Field(), fe.Param())
		return t
	})

	if err != nil {
		panic(fmt.Sprintf("failed to register translation for '%s': %s", tag, err))
	}

	return vb
}

// AddStructValidation registers a StructLevel validation function for specific types.
func (vb *ValidatorBuilder) AddStructValidation(fn validator.StructLevelFunc, types ...any) *ValidatorBuilder {
	vb.Validate.RegisterStructValidation(fn, types...)
	return vb
}

// AddTranslation manually registers a custom error message for a specific tag.
// You use this when your validation logic (like StructLevel) is separate from the tag definition.
func (vb *ValidatorBuilder) AddTranslation(tag string, msg string) *ValidatorBuilder {
	err := vb.Validate.RegisterTranslation(tag, vb.Trans, func(ut ut.Translator) error {
		return ut.Add(tag, msg, true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(tag, fe.Field(), fe.Param())
		return t
	})

	if err != nil {
		panic(fmt.Sprintf("failed to register translation for '%s': %s", tag, err))
	}

	return vb
}

// Check is a helper to run validation and return a simple map of error messages
// instead of the complex validator.ValidationErrors struct.
func (vb *ValidatorBuilder) Check(s any) []responses.ValidationErrorDetail {
	err := vb.Validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return []responses.ValidationErrorDetail{{Message: "Invalid input", Value: err.Error()}}
	}
	details := make([]responses.ValidationErrorDetail, len(validationErrors))
	for i, e := range validationErrors {
		details[i] = responses.ValidationErrorDetail{
			JSONField:   e.Field(),
			StructField: e.StructField(),
			Value:       utils.ToString(e.Value()),
			Message:     e.Translate(vb.Trans),
		}
	}
	return details
}
