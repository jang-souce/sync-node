package util

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

var (
	uni      *ut.UniversalTranslator
	validate *validator.Validate
	trans    ut.Translator
)

func init() {
	// 初始化翻译器
	zhT := zh.New()
	enT := en.New()
	uni = ut.New(enT, zhT, enT)

	// 获取验证器
	var ok bool
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		validate = v
	} else {
		validate = validator.New()
	}

	// 注册 JSON tag 作为字段名
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// 设置默认语言为中文
	trans, ok = uni.GetTranslator("zh")
	if !ok {
		panic("found no translator for 'zh'")
	}

	// 注册翻译
	// 此时验证器可能已经被 Gin 初始化过，这里尝试注册中文翻译
	// 如果是独立使用的 validate 实例，直接注册
	// 如果是 Gin 的 binding.Validator，它是一个全局单例，我们需要确保注册进去
	switch v := binding.Validator.Engine().(type) {
	case *validator.Validate:
		_ = zh_translations.RegisterDefaultTranslations(v, trans)
	}
	_ = zh_translations.RegisterDefaultTranslations(validate, trans)
	enTrans, _ := uni.GetTranslator("en")
	_ = en_translations.RegisterDefaultTranslations(validate, enTrans)
}

// Translate 翻译校验错误
func Translate(err error) string {
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 非校验错误，直接返回
		return err.Error()
	}

	// 拼接错误信息
	var msgList []string
	for _, e := range errs {
		msgList = append(msgList, e.Translate(trans))
	}
	return strings.Join(msgList, "; ")
}

// ValidateStruct 手动校验结构体
func ValidateStruct(obj interface{}) error {
	if err := validate.Struct(obj); err != nil {
		return fmt.Errorf("%s", Translate(err))
	}
	return nil
}
