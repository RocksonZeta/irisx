package irisx

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/microcosm-cc/bluemonday"
)

type Validator struct {
	ctx *Context
	// params interface{}
	key    string
	value  string
	exists bool
	goon   bool
	// errors map[string]string
	// isEmpty bool
}

func NewValidator(ctx *Context, key string, value string, exists bool) *Validator {
	return &Validator{
		ctx: ctx,
		// params: params,
		key:    key,
		value:  value,
		exists: exists,
		goon:   true,
		// errors: make(map[string]string),
	}
}
func NewValidatorValues(ctx *Context, key string, values []string, exists bool) *ValidatorValues {
	return &ValidatorValues{
		ctx: ctx,
		// params: params,
		key:    key,
		values: values,
		exists: exists,
		goon:   true,
		// errors: make(map[string]string),
	}
}
func NewValidatorFile(ctx *Context, key string, file multipart.File, header *multipart.FileHeader, err error) *ValidatorFile {
	return &ValidatorFile{
		ctx: ctx,
		// params: params,
		key:    key,
		file:   file,
		header: header,
		exists: err != nil,
		goon:   true,
		err:    err,
		// errors: make(map[string]string),
	}
}

func (v *Validator) addError(msg string) {
	v.goon = false
	v.ctx.AddParamError(v.key, msg)
	// v.errors[v.key] = msg
}
func (v *Validator) hasError() bool {
	return len(v.ctx.ParamErrors()) != 0
}
func (v *Validator) format(defaultMsg string, msg []string) string {
	if len(msg) > 0 {
		return msg[0]
	}
	return defaultMsg
}
func (v *Validator) Optional() *Validator {
	if !v.exists {
		v.goon = false
	}
	return v
}

func (v *Validator) NotEmpty(msg ...string) *Validator {
	if v.goon && "" == v.value {
		v.addError(v.format(v.key+" can not be empty.", msg))
	}
	return v
}
func (v *Validator) Empty(msg ...string) *Validator {
	if v.goon {
		if !v.exists || "" == v.value {
			v.goon = false
		}
	}
	return v
}
func mustMatch(reg, s string) bool {
	m, _ := regexp.MatchString(reg, s)
	return m
}
func (v *Validator) NotBlank(msg ...string) *Validator {
	if v.goon && ("" == v.value || mustMatch("^\\s*$", v.value)) {
		v.addError(v.format(v.key+" can not be blank.", msg))
	}
	return v
}
func (v *Validator) Exist(msg ...string) *Validator {
	if v.goon && !v.exists {
		v.addError(v.format(v.key+" should exists.", msg))
	}
	return v
}
func (v *Validator) Match(reg string, msg ...string) *Validator {
	if v.goon && !mustMatch(reg, v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) NotMatch(reg string, msg ...string) *Validator {
	if v.goon && mustMatch(reg, v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) Ensure(assertion, shouldBail bool, msg ...string) *Validator {
	if shouldBail {
		v.goon = false
	}
	if v.goon && !assertion {
		v.addError(v.format(v.key+" failed an assertion.", msg))
	}
	return v
}
func (v *Validator) EnsureNot(assertion, shouldBail bool, msg ...string) *Validator {
	if shouldBail {
		v.goon = false
	}
	if v.goon && assertion {
		v.addError(v.format(v.key+" failed an assertion.", msg))
	}
	return v
}
func (v *Validator) IsInt(msg ...string) *Validator {
	if v.goon && !govalidator.IsInt(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsFloat(msg ...string) *Validator {
	if v.goon && !govalidator.IsFloat(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsBool(msg ...string) *Validator {
	_, err := strconv.ParseBool(v.value)
	if v.goon && err != nil {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}

func (v *Validator) Len(min, max int, msg ...string) *Validator {
	if v.goon {
		if len(v.value) < min {
			v.addError(v.format(v.key+"'s length must equal or great than "+strconv.Itoa(min), msg))
			return v
		}
		if max > 0 && len(v.value) > max {
			v.addError(v.format(v.key+"'s length must equal or less than "+strconv.Itoa(max), msg))
			return v
		}
	}
	return v
}
func (v *Validator) ByteLen(min, max int, msg ...string) *Validator {
	if v.goon && !govalidator.IsByteLength(v.value, min, max) {
		v.addError(v.format(v.key+"'s length no ok.", msg))
	}
	return v
}

func (v *Validator) In(values []string, msg ...string) *Validator {
	if v.goon && len(values) > 0 {
		for _, x := range values {
			if v.value == x {
				return v
			}
		}
		v.addError(v.format(v.key+" is bad.", msg))
	}
	return v
}
func (v *Validator) InInts(values []int, msg ...string) *Validator {
	y := v.Int(-1, msg...)
	if v.goon && len(values) > 0 {
		for _, x := range values {
			if y == x {
				return v
			}
		}
		v.addError(v.format(v.key+" is bad.", msg))
	}
	return v
}

func (v *Validator) IsUrl(msg ...string) *Validator {
	if v.goon && !govalidator.IsURL(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsEmail(msg ...string) *Validator {
	if v.goon && !govalidator.IsEmail(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsIP(msg ...string) *Validator {
	if v.goon && !govalidator.IsIP(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsASCII(msg ...string) *Validator {
	if v.goon && !govalidator.IsASCII(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsAlpha(msg ...string) *Validator {
	if v.goon && !govalidator.IsAlpha(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsAlphanumeric(msg ...string) *Validator {
	if v.goon && !govalidator.IsAlphanumeric(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsFilePath(msg ...string) *Validator {
	ok, _ := govalidator.IsFilePath(v.value)
	if v.goon && !ok {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsJSON(msg ...string) *Validator {
	if v.goon && !govalidator.IsJSON(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsNumeric(msg ...string) *Validator {
	if v.goon && !govalidator.IsNumeric(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsTime(format string, msg ...string) *Validator {
	if v.goon && !govalidator.IsTime(format, v.value) {
		v.addError(v.format(v.key+" is bad format", msg))
	}
	return v
}
func (v *Validator) IsLowerCase(msg ...string) *Validator {
	if v.goon && !govalidator.IsLowerCase(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}
func (v *Validator) IsUpperCase(msg ...string) *Validator {
	if v.goon && !govalidator.IsUpperCase(v.value) {
		v.addError(v.format(v.key+" is bad format.", msg))
	}
	return v
}

func (v *Validator) Trim() *Validator {
	if v.goon {
		v.value = strings.TrimSpace(v.value)
	}
	return v
}

//// Get value

//Int to int value
func (v *Validator) String(dv ...string) string {
	if !v.hasError() {
		return v.value
	}
	if len(dv) <= 0 {
		return ""
	}
	return dv[0]
}

var htmlSanitizer = bluemonday.UGCPolicy()

//Int to int value
func (v *Validator) SanitizeHtml() string {
	// if v.goon {
	if len(v.value) == 0 {
		return ""
	}
	return htmlSanitizer.Sanitize(v.value)
	// }
	// return v.value
}
func (v *Validator) Present() bool {
	return v.exists
}
func (v *Validator) Int(dv int, msg ...string) int {
	v.IsInt(msg...)
	if v.goon && v.value != "" {
		x, err := strconv.Atoi(v.value)
		if err != nil {
			v.addError(v.format(v.key+" is not int format.", msg))
			return dv
		}
		return x
	}
	return dv
}
func (v *Validator) Int64(dv int64, msg ...string) int64 {
	v.IsInt(msg...)
	if v.goon && v.value != "" {
		x, err := strconv.ParseInt(v.value, 10, 64)
		if err != nil {
			v.addError(v.format(v.key+" is not int format.", msg))
			return dv
		}
		return x
	}
	return dv
}
func (v *Validator) Float(dv float64, msg ...string) float64 {
	v.IsFloat(msg...)
	if v.goon && v.value != "" {
		x, err := strconv.ParseFloat(v.value, 64)
		if err != nil {
			v.addError(v.format(v.key+" is not int format.", msg))
			return dv
		}
		return x
	}
	return dv
}
func (v *Validator) Float32(dv float32, msg ...string) float32 {
	v.IsFloat(msg...)
	if v.goon && v.value != "" {
		x, err := strconv.ParseFloat(v.value, 32)
		if err != nil {
			v.addError(v.format(v.key+" is not int format.", msg))
			return dv
		}
		return float32(x)
	}
	return dv
}
func (v *Validator) Bool(dv bool, msg ...string) bool {
	v.IsBool(msg...)
	if v.goon && v.value != "" {
		x, err := strconv.ParseBool(v.value)
		if err != nil {
			v.addError(v.format(v.key+" is not bool format.", msg))
			return dv
		}
		return x
	}
	return dv
}
func (v *Validator) Json(r interface{}, msg ...string) {
	if v.goon && v.value != "" {
		err := json.Unmarshal([]byte(v.value), r)
		if err != nil {
			v.addError(v.format(v.key+" is not json format.", msg))
			return
		}
	}
}

func ParseTimeLocal(format, date string) (time.Time, error) {
	loc, _ := time.LoadLocation("Local")
	return time.ParseInLocation(format, date, loc)
}

func (v *Validator) DateFormat(format string, dv time.Time, msg ...string) time.Time {
	v.IsTime(format, msg...)
	if v.goon && v.value != "" {
		x, err := ParseTimeLocal(format, v.value)
		if err != nil {
			v.addError(v.format(v.key+" is not int format.", msg))
			return dv
		}
		return x
	}
	return dv
}

func (v *Validator) DateTime(dv time.Time, msg ...string) time.Time {
	return v.DateFormat("2006-01-02 15:04:05", dv, msg...)
}
func (v *Validator) DateTimeShort(dv time.Time, msg ...string) time.Time {
	return v.DateFormat("2006-01-02 15:04", dv, msg...)
}
func (v *Validator) Date(dv time.Time, msg ...string) time.Time {
	return v.DateFormat("2006-01-02", dv, msg...)
}
func (v *Validator) Time(dv time.Time, msg ...string) time.Time {
	return v.DateFormat("15:04:05", dv, msg...)
}
func (v *Validator) TimeShort(dv time.Time, msg ...string) time.Time {
	return v.DateFormat("15:04", dv, msg...)
}

type ValidatorValues struct {
	ctx *Context
	// params interface{}
	key    string
	values []string
	exists bool
	goon   bool
	// isEmpty bool
	// errors map[string]string
}

func (v *ValidatorValues) addError(msg string) {
	v.goon = false
	v.ctx.AddParamError(v.key, msg)
}
func (v *ValidatorValues) hasError() bool {
	return len(v.ctx.ParamErrors()) != 0
}
func (v *ValidatorValues) format(defaultMsg string, msg []string) string {
	if len(msg) > 0 {
		return msg[0]
	}
	return defaultMsg
}
func (v *ValidatorValues) Optional() *ValidatorValues {
	if !v.exists {
		v.goon = false
	}
	return v
}

func (v *ValidatorValues) NotEmpty(msg ...string) *ValidatorValues {
	if v.goon && len(v.values) == 0 {
		v.addError(v.format(v.key+" can not be empty.", msg))
	}
	return v
}
func (v *ValidatorValues) Empty(msg ...string) *ValidatorValues {
	if v.goon {
		if len(v.values) == 0 {
			v.goon = false
		}
	}
	return v
}

func (v *ValidatorValues) Len(min, max int, msg ...string) *ValidatorValues {
	if v.goon {
		if len(v.values) < min {
			v.addError(v.format(v.key+"'s length must equal or great than "+strconv.Itoa(min), msg))
			return v
		}
		if max > 0 && len(v.values) > max {
			v.addError(v.format(v.key+"'s length must equal or less than "+strconv.Itoa(max), msg))
			return v
		}
	}
	return v
}
func (v *ValidatorValues) Match(reg string, msg ...string) *ValidatorValues {
	if v.goon {
		for _, x := range v.values {
			if !mustMatch(reg, x) {
				v.addError(v.format(v.key+" is bad format.", msg))
				return v
			}
		}
	}
	return v
}
func (v *ValidatorValues) NotMatch(reg string, msg ...string) *ValidatorValues {
	if v.goon {
		for _, x := range v.values {
			if mustMatch(reg, x) {
				v.addError(v.format(v.key+" is bad format.", msg))
				return v
			}
		}
	}
	return v
}
func (v *ValidatorValues) Ensure(assertion, shouldBail bool, msg ...string) *ValidatorValues {
	if shouldBail {
		v.goon = false
	}
	if v.goon && !assertion {
		v.addError(v.format(v.key+" failed an assertion.", msg))
	}
	return v
}
func (v *ValidatorValues) EnsureNot(assertion, shouldBail bool, msg ...string) *ValidatorValues {
	if shouldBail {
		v.goon = false
	}
	if v.goon && assertion {
		v.addError(v.format(v.key+" failed an assertion.", msg))
	}
	return v
}

func (v *ValidatorValues) Strings(dv []string, msg ...string) []string {
	if v.goon {
		return v.values
	}
	return dv
}
func (v *ValidatorValues) Ints(dv []int, msg ...string) []int {
	if v.goon {
		r := make([]int, len(v.values))
		var err error
		for i, x := range v.values {
			r[i], err = strconv.Atoi(x)
			if err != nil {
				v.addError(v.format(v.key+" format error.", msg))
				return dv
			}
		}
		return r
	}
	return dv
}
func (v *ValidatorValues) Floats(dv []float64, msg ...string) []float64 {
	if v.goon {
		r := make([]float64, len(v.values))
		var err error
		for i, x := range v.values {
			r[i], err = strconv.ParseFloat(x, 64)
			if err != nil {
				v.addError(v.format(v.key+" format error.", msg))
				return dv
			}
		}
		return r
	}
	return dv
}
func (v *ValidatorValues) Float32(dv []float32, msg ...string) []float32 {
	if v.goon {
		r := make([]float32, len(v.values))
		for i, x := range v.values {
			x1, err := strconv.ParseFloat(x, 32)
			r[i] = float32(x1)
			if err != nil {
				v.addError(v.format(v.key+" format error.", msg))
				return dv
			}
		}
		return r
	}
	return dv
}

type ValidatorFile struct {
	ctx *Context
	// params interface{}
	key    string
	file   multipart.File
	header *multipart.FileHeader
	exists bool
	goon   bool
	err    error
	// isEmpty bool
	// errors map[string]string
}

func (v *ValidatorFile) addError(msg string) {
	v.goon = false
	v.ctx.AddParamError(v.key, msg)
}
func (v *ValidatorFile) hasError() bool {
	return len(v.ctx.ParamErrors()) != 0
}
func (v *ValidatorFile) format(defaultMsg string, msg []string) string {
	if len(msg) > 0 {
		return msg[0]
	}
	return defaultMsg
}
func (v *ValidatorFile) Optional() *ValidatorFile {
	if !v.exists {
		v.goon = false
	}
	return v
}

func (v *ValidatorFile) NotEmpty(msg ...string) *ValidatorFile {
	if v.goon && (v.header == nil || v.header != nil && v.header.Size <= 0) {
		v.addError(v.format(v.key+" can not be empty.", msg))
	}
	return v
}
func (v *ValidatorFile) Empty(msg ...string) *ValidatorFile {
	if v.goon && v.header.Size <= 0 {
		v.goon = false
	}
	return v
}

func (v *ValidatorFile) Len(min, max int64, msg ...string) *ValidatorFile {
	if v.goon {
		size := v.header.Size
		if size < min {
			v.addError(v.format(v.key+"'s length must equal or great than "+strconv.FormatInt(min, 10), msg))
			return v
		}
		if max > 0 && size > max {
			v.addError(v.format(v.key+"'s length must equal or less than "+strconv.FormatInt(max, 10), msg))
			return v
		}
	}
	return v
}

//ExtIn exts {"jpg","png"}
func (v *ValidatorFile) ExtIn(exts []string, msg ...string) *ValidatorFile {
	if v.goon && len(exts) > 0 {
		ext := filepath.Ext(v.header.Filename)
		for _, x := range exts {
			if ext == "."+x {
				return v
			}
		}
		v.addError(v.format(v.key+" is bad file type.", msg))
	}
	return v
}

func (v *ValidatorFile) IsImage(msg ...string) *ValidatorFile {
	exts := []string{"jpg", "jpeg", "png", "gif", "bmp", "tiff", "tif"}
	if v.goon {
		ext := filepath.Ext(v.header.Filename)
		for _, x := range exts {
			if ext == "."+x {
				return v
			}
		}
		v.addError(v.format(v.key+" is bad file type.", msg))
	}
	return v
}

func (v *ValidatorFile) Copy(dstFile string) {
	// src, err := v.file.Open()
	// if err != nil {
	// 	v.addError(v.key + ":" + err.Error())
	// 	return
	// }
	defer v.file.Close()
	dst, err := os.OpenFile(dstFile, os.O_CREATE, 0644)
	if err != nil {
		v.addError(v.key + ":" + err.Error())
		return
	}
	defer dst.Close()
	_, err = io.Copy(dst, v.file)
	if err != nil {
		v.addError(v.key + ":" + err.Error())
		return
	}
}

type UploadFile struct {
	Header *multipart.FileHeader
	File   multipart.File
	Error  error
	Ext    string
	Size   int64
}

func (v *ValidatorFile) File(dstFile string) UploadFile {
	var r UploadFile
	r.Header = v.header
	r.File = v.file
	r.Error = v.err
	if r.Header != nil {
		r.Ext = filepath.Ext(r.Header.Filename)
		r.Size = r.Header.Size
	}
	return r

}

// type SaveFileResult struct {
// 	Url    string
// 	Header *multipart.FileHeader
// 	Error  error
// }
