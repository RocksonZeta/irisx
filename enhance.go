package irisx

import (
	"encoding/json"
	"fmt"
	"html/template"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/RocksonZeta/wrap/errs"
	"github.com/RocksonZeta/wrap/utils/nutil"
	"github.com/RocksonZeta/wrap/utils/stringutil"
	"github.com/RocksonZeta/wrap/utils/sutil"
	"github.com/RocksonZeta/wrap/utils/timeutil"
	"github.com/kataras/iris/v12/view"
)

const (
	TypeCastError = 1 + iota
)

func newError(state int, msg string) *errs.Err {
	return &errs.Err{
		State:   state,
		Message: msg,
		Pkg:     "github.com/RocksonZeta/wrap/iriswrap",
		Module:  "Enhance",
	}
}

//Enhance Template
func Enhance(app *view.HTMLEngine) {
	app.AddFunc("json", func(obj interface{}) (template.HTML, error) {
		bs, err := json.Marshal(obj)
		return template.HTML(string(bs)), err

	})
	app.AddFunc("has", func(kv map[string]interface{}, k string) bool {
		_, ok := kv[k]
		return ok
	})
	app.AddFunc("list", func(vs ...interface{}) []interface{} {
		return vs
	})
	app.AddFunc("dict", func(kvs ...interface{}) map[string]interface{} {
		r := make(map[string]interface{})
		for i := 0; i < len(kvs); i += 2 {
			r[kvs[i].(string)] = kvs[i+1]
		}
		return r
	})
	app.AddFunc("type", func(v interface{}) string {
		return reflect.TypeOf(v).String()
	})
	app.AddFunc("indexAt", func(arr interface{}, v interface{}) (interface{}, error) {
		if a, ok := arr.([]string); ok {
			if i, ok := v.(int); ok {
				return a[i], nil
			}
		}
		if a, ok := arr.(map[int]string); ok {
			if i, ok := v.(int); ok {
				return a[i], nil
			}
		}
		return nil, newError(TypeCastError, "indexAt")
	})
	// app.AddFunc("unescape", func(s string) template.HTML {
	// 	return template.HTML(s)
	// })
	app.AddFunc("rawCss", func(s string) template.CSS {
		return template.CSS(s)
	})
	app.AddFunc("rawJs", func(s string) template.JS {
		return template.JS(s)
	})
	app.AddFunc("rawHtmlAttr", func(s string) template.HTMLAttr {
		return template.HTMLAttr(s)
	})
	app.AddFunc("rawHtml", func(s string) template.HTML {
		return template.HTML(s)
	})
	app.AddFunc("rawUrl", func(s string) template.URL {
		return template.URL(s)
	})
	app.AddFunc("add", func(a, b interface{}) interface{} {
		return nutil.Add(a, b)
	})
	app.AddFunc("sub", func(a, b interface{}) interface{} {
		return nutil.Sub(a, b)
	})
	app.AddFunc("mul", func(a, b interface{}) interface{} {
		return nutil.Mul(a, b)
	})
	app.AddFunc("div", func(a, b interface{}) interface{} {
		return nutil.Div(a, b)
	})
	app.AddFunc("divf", func(a, b interface{}) float64 {
		return nutil.DivFloat(a, b)
	})
	app.AddFunc("mod", func(a, b interface{}) interface{} {
		return nutil.Mod(a, b)
	})
	app.AddFunc("int", func(a interface{}) int {
		return nutil.IntMust(a)
	})
	app.AddFunc("int64", func(a interface{}) int64 {
		return nutil.Int64Must(a)
	})
	app.AddFunc("float", func(a interface{}) float64 {
		return nutil.Float64Must(a)
	})
	app.AddFunc("float32", func(a interface{}) float32 {
		return nutil.Float32Must(a)
	})
	app.AddFunc("string", func(a interface{}) string {
		return nutil.String(a)
	})
	app.AddFunc("set", func(v interface{}, r interface{}) string {
		v = r
		return ""
	})
	app.AddFunc("css", func(files []string) template.HTML {
		s := ""
		for _, f := range files {
			s += `<link rel="stylesheet" href="` + f + `"/>`
		}
		return template.HTML(s)
	})

	app.AddFunc("js", func(files []string) template.HTML {
		s := ""
		for _, f := range files {
			s += `<script src="` + f + `"></script>
`
		}
		return template.HTML(s)
	})
	app.AddFunc("ajs", func(ctx *Context, files ...string) template.HTML {
		ctx.Js(files...)
		return ""
	})
	app.AddFunc("int", func(o interface{}) (int, error) {
		return nutil.Int(o)
	})
	app.AddFunc("if3", func(condition interface{}, v1, v2 string) template.HTML {
		if nutil.Bool(condition) {
			return template.HTML(v1)
		}
		return template.HTML(v2)
	})
	app.AddFunc("ifeq3", func(c1, c2 interface{}, v1, v2 string) template.HTML {
		if c1 == c2 {
			return template.HTML(v1)
		}
		return template.HTML(v2)
	})
	app.AddFunc("if2", func(condition interface{}, v string) template.HTML {
		if nutil.Bool(condition) {
			return template.HTML(v)
		}
		return template.HTML("")
	})
	app.AddFunc("ifeq2", func(v1, v2 interface{}, v string) template.HTML {
		if v1 == v2 {
			return template.HTML(v)
		}
		return template.HTML("")
	})
	app.AddFunc("page", func(ctx *Context, total int64) template.HTML {
		return template.HTML(Page(ctx, total, true, "pagination pagination-warning"))
	})
	app.AddFunc("pagex", func(ctx *Context, total int64, showTotal bool, classes string) template.HTML {
		return template.HTML(Page(ctx, total, showTotal, classes))
	})
	app.AddFunc("select", func(options map[int]string, defaultV interface{}, needEmptyOption bool, props string) template.HTML {
		return selector(options, defaultV, needEmptyOption, props)
	})
	app.AddFunc("date", func(obj interface{}) (string, error) {
		if t, err := obj.(int); err {
			return timeutil.FormatDate(int64(t)), nil
		}
		if t, err := obj.(int64); err {
			return timeutil.FormatDate(t), nil
		}
		return "", nil
	})
	app.AddFunc("time", func(obj interface{}) (string, error) {
		if t, err := obj.(int); err {
			return timeutil.FormatTime(int64(t)), nil
		}
		if t, err := obj.(int64); err {
			return timeutil.FormatTime(t), nil
		}
		return "", nil
	})
	app.AddFunc("datetime", func(obj interface{}) (string, error) {
		if t, err := obj.(int); err {
			return timeutil.FormatDatetimeShort(int64(t)), nil
		}
		if t, err := obj.(int64); err {
			return timeutil.FormatDatetimeShort(t), nil
		}
		return "", nil
	})
	app.AddFunc("dateformat", func(obj ...interface{}) (string, error) {
		if 2 > len(obj) {
			return "", nil
		}
		t, ok := obj[0].(int64)
		t1, ok1 := obj[0].(int64)
		f, ok2 := obj[1].(string)
		if ok && ok2 {
			return timeutil.FormatTimeWith(t, f), nil
		}
		if ok1 && ok2 {
			return timeutil.FormatTimeWith(t1, f), nil
		}
		return "", nil
	})
	app.AddFunc("duration", func(obj interface{}) (string, error) {
		if t, err := obj.(int); err {
			return timeutil.FormatDuration(int64(t)), nil
		}
		if t, err := obj.(int64); err {
			return timeutil.FormatDuration(t), nil
		}
		return "", nil
	})
	app.AddFunc("format", func(str string, params ...interface{}) string {
		return fmt.Sprintf(str, params...)
	})
	app.AddFunc("match", func(pattern, str string) (bool, error) {
		return regexp.MatchString(pattern, str)
	})
	app.AddFunc("replace", func(src, pattern, repl string) (string, error) {
		m, err := regexp.Compile(pattern)
		if err != nil {
			return "", err
		}
		return m.ReplaceAllString(src, repl), nil
	})
	app.AddFunc("concat", func(a ...interface{}) string {
		format := ""
		for _ = range a {
			format += "%v"
		}
		return fmt.Sprintf(format, a...)
	})
	//case,condition,case1,v1,case2,v2,....,defaultValue
	app.AddFunc("case", func(c interface{}, a ...interface{}) string {

		n := len(a)
		n = n - n%2
		m := make(map[interface{}]interface{}, n/2)
		for i := 0; i < n; i += 2 {
			m[a[i]] = a[i+1]
		}
		if v, ok := m[c]; ok {
			fmt.Println("match ok:", v)
			return fmt.Sprintf("%v", v)
		}
		if n != len(a) {
			return fmt.Sprintf("%v", a[len(a)-1])
		}
		return ""

	})
	// app.AddFunc("dd", func(ddlink string) string {
	// 	// if httpfsclient.(ddlink) {
	// 	httpfsclient.HfLink(ddlink).Url()
	// 	// }
	// 	return ddlink
	// })

	//map get map value by key :return m[key]
	app.AddFunc("m", func(m interface{}, key interface{}) interface{} {
		if m == nil || key == nil {
			return nil
		}
		mV := reflect.ValueOf(m)
		r := mV.MapIndex(reflect.ValueOf(key))
		if !r.IsValid() {
			return nil
		}
		return r.Interface()
	})
	app.AddFunc("a", func(arr interface{}, index int) interface{} {
		if arr == nil {
			return nil
		}
		arrV := reflect.ValueOf(arr)
		r := arrV.Index(index)
		if !r.IsValid() {
			return nil
		}
		return r.Interface()
	})

	app.AddFunc("in", func(arr, value interface{}) bool {
		return Index(arr, value) != -1
	})
	app.AddFunc("notIn", func(arr, value interface{}) bool {
		return Index(arr, value) == -1
	})
	app.AddFunc("index", func(arr, value interface{}) int {
		return Index(arr, value)
	})
	app.AddFunc("alphabet", func(i int) string {
		return string('A' + i)
	})
	// app.AddFunc("Dict", func(module string) interface{} {
	// 	return constant.Dicts[module]
	// })
	// app.AddFunc("DictValue", func(module string, key interface{}) interface{} {
	// 	return structutil.MapGet(constant.Dicts[module], key)
	// })
	app.AddFunc("pathJoin", func(parts ...string) string {
		return filepath.Join(parts...)
	})
	//list ,vf,sf,dv,empty,attr
	app.AddFunc("select", func(list interface{}, valueField, showField string, defaultValue interface{}, needEmptyOption bool, attr string) interface{} {
		r := `<select ` + attr + `>`
		if needEmptyOption {
			r += `<option></option>`
		}
		if list != nil {
			values := sutil.NewSlice(list)
			for _, v := range values {
				value := v.Field(valueField).Value()
				var show string
				if -1 == strings.Index(showField, "{") { //非模板
					show = fmt.Sprintf("%v", v.Field(showField).Value())
				} else {
					show = stringutil.Template(showField, v.Map())
				}
				r += `<option value="` + fmt.Sprintf("%v", value) + `"`
				if value == defaultValue {
					r += ` selected="selected" `
				}
				r += ">"
				r += fmt.Sprintf("%v", show) + `</option>`
			}
		}
		r += `</select>`
		return template.HTML(r)
	})

	app.AddFunc("fsize", func(size interface{}) string {
		return ""
	})
}

func Index(arr, value interface{}) int {
	switch value.(type) {
	case int:
		if a, ok := arr.([]int); ok {
			for i, v := range a {
				if v == value {
					return i
				}
				return -1
			}
		}
	case int64:
		if a, ok := arr.([]int64); ok {
			for i, v := range a {
				if v == value {
					return i
				}
				return -1
			}
		}
	case string:
		if a, ok := arr.([]string); ok {
			for i, v := range a {
				fmt.Println(i, v)
				if v == value {
					return i
				}
				return -1
			}
		}
	case interface{}:
		if a, ok := arr.([]interface{}); ok {
			for i, v := range a {
				if v == value {
					return i
				}
				return -1
			}
		}
	}
	arrV := reflect.ValueOf(arr)
	arrLen := arrV.Len()
	for i := 0; i < arrLen; i++ {
		if value == arrV.Index(i).Interface() {
			return i
		}
	}
	return -1
}
