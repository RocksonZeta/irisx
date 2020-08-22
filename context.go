package irisx

import (
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RocksonZeta/wrap/errs"
	"github.com/RocksonZeta/wrap/wraplog"
	"github.com/asaskevich/govalidator"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/host"
)

var log = wraplog.Logger.Fork("github.com/RocksonZeta/irisx", "Context")

const (
	RequestKeyParamErrors = "ParamErrors"
	RequestKeyScripts     = "Scripts"
)

type SessionProvider interface {
	GetSessionId(ctx *Context) string
	SetSessionId(ctx *Context)
	Set(key string, value interface{}, secs int) error
	Get(key string, result interface{}) error
	Refresh(key string, secs int) error
	Remove(key string) error
	UidKey() string
}

// type H map[string]interface{}
type Context struct {
	context.Context
	BeforeView      func(ctx *Context, tplFile string)
	SessionProvider SessionProvider
}

func (ctx *Context) Do(handlers context.Handlers) {
	context.Do(ctx, handlers)
}
func (ctx *Context) Next() {
	context.Next(ctx)
}

func (ctx *Context) SetCookieLocal(key, value string, maxAge int, httpOnly bool, domain string) {
	ctx.Context.SetCookie(&http.Cookie{Name: key, Value: value, MaxAge: maxAge, Path: "/", HttpOnly: httpOnly, Domain: domain})
}
func (ctx *Context) RemoveCookieLocal(key string) {
	ctx.Context.SetCookie(&http.Cookie{Name: key, Value: "", MaxAge: -1, Path: "/"})
}

func (ctx *Context) ParamErrors() map[string]string {
	r := ctx.Values().Get(RequestKeyParamErrors)
	if r == nil {
		return nil
	}
	return r.(map[string]string)
}
func (ctx *Context) Scripts() []string {
	r := ctx.Values().Get(RequestKeyScripts)
	if r == nil {
		return nil
	}
	return r.([]string)
}
func (ctx *Context) AddParamError(key, msg string) {
	if nil == ctx.ParamErrors() {
		ctx.Values().Set(RequestKeyParamErrors, make(map[string]string))
	}
	ctx.ParamErrors()[key] = msg
}
func (ctx *Context) AppendViewData(key string, values ...string) {
	if m, ok := ctx.GetViewData()[key]; ok {
		v := append(m.([]string), values...)
		ctx.ViewData(key, v)
	} else {
		ctx.ViewData(key, values)
	}
}

func (ctx *Context) Js(js ...string) string {
	ctx.AppendViewData("Js", js...)
	return ""
}

func (ctx *Context) Css(css ...string) {
	ctx.AppendViewData("Css", css...)
}
func (ctx *Context) Title(title string) {
	ctx.ViewData("Title", title)
}

func (ctx *Context) Redirect(urlToRedirect string, statusHeader ...int) {
	ctx.Context.Redirect(urlToRedirect, statusHeader...)
}
func (ctx *Context) View(filename string, optionalViewModel ...interface{}) error {
	ctx.ViewData("C", ctx)
	// if ctx.AutoHead {
	// 	headfile := "view/" + filename[:strings.LastIndex(filename, ".")] + ".head"
	// 	var bs []byte
	// 	if old, ok := headers.Load(headfile); ok {
	// 		bs = old.([]byte)
	// 	}
	// 	if s, err := os.Stat(headfile); err == nil && !s.IsDir() {
	// 		var ioerr error
	// 		bs, ioerr = ioutil.ReadFile(headfile)
	// 		if ioerr == nil {
	// 			headers.Store(headfile, bs)
	// 		} else {
	// 			log.Error().Func("View").Err(ioerr).Stack().Str("filename", filename).Msg(err.Error())
	// 		}
	// 	}
	// 	ctx.ViewData("_view_html_head", template.HTML(bs))
	// }
	// if ctx.AutoIncludeCss {
	// 	ctx.Css("/static/css/" + filename[:strings.LastIndex(filename, ".")] + ".css")
	// }
	// if ctx.AutoIncludeJs {
	// 	ctx.Js("/static/js/" + filename[:strings.LastIndex(filename, ".")] + ".js")
	// }
	if nil != ctx.BeforeView {
		ctx.BeforeView(ctx, filename)
	}
	err := ctx.Context.View(filename, optionalViewModel...)
	if nil != err {
		log.Error().Func("View").Err(err).Stack().Str("filename", filename).Msg(err.Error())
	}
	return err
}

func (ctx *Context) Ok(data interface{}) {
	ctx.JSON(errs.Err{State: 0, Data: data}.Result())
}

// type Select2 struct {
// 	Id   int    `json:"id"`
// 	Text string `json:"text"`
// }

// func (ctx *Context) Fail() {
// 	ctx.JSON(ctx.Error)
// }
func (ctx *Context) Err(status int, data interface{}) {
	ctx.JSON(errs.Err{State: status, Data: data})
}

// func (ctx *Context) HasError() bool {
// 	return ctx.Error != nil && ctx.Error.State != 0
// }

//
func (ctx *Context) ReadValidate(form interface{}) bool {
	err := ctx.ReadForm(form)
	if nil != err {
		log.Error().Func("ReadValidate").Stack().Err(err).Interface("form", form).Msg(err.Error())
	}
	ok, err := govalidator.ValidateStruct(form)
	if ok {
		return ok
	}
	if nil != err {
		if errs, ok := err.(govalidator.Errors); ok {

			// errorMap := make(map[string]string, len(errs))
			for _, e := range errs {
				s := e.Error()
				i := strings.Index(s, ":")
				// if ctx.Error == nil {
				// 	ctx.Error = &errors.PageError{}
				// 	ctx.Error.State = errorcode.HttpParamError
				// }
				if -1 != i {
					// if nil == ctx.ParamErrors {
					// 	ctx.ParamErrors = make(map[string]string)
					// }
					ctx.AddParamError(strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]))
					// ctx.Error.FieldError[strings.TrimSpace(s[:i])] = strings.TrimSpace(s[i+1:])
				} else {
					// if nil == ctx.ErrorMsgs {
					// 	ctx.FieldError = make(map[string]string)
					// }
					// ctx.Error.Message = s
					break
				}
			}
			// ctx.Err(errorcode.HttpParamError, errorMap)
			// } else {
			// ctx.Err(errorcode.HttpParamError, err)
		}
		// ctx.SetError("/form")
	}
	return false
}

func (ctx *Context) PathParent() string {
	p := ctx.Path()
	if "/" == p {
		return p
	}
	return filepath.Dir(p)
}
func (ctx *Context) PathLeft(count int) string {
	p := ctx.Path()
	if "/" == p {
		return p
	}
	pcount := strings.Count(strings.TrimRight(p, "/"), "/")
	if count >= pcount {
		return p
	}
	cur := p
	for i := 0; i < pcount-count; i++ {
		cur = filepath.Dir(cur)
	}
	return cur
}
func (ctx *Context) PathRight(count int) string {
	p := ctx.Path()
	if "/" == p {
		return p
	}
	trimedPath := strings.Trim(p, "/")
	pcount := strings.Count(trimedPath, "/") + 1
	if count >= pcount {
		return p
	}
	return "/" + strings.Join(strings.Split(trimedPath, "/")[pcount-count:], "/")
}
func (ctx *Context) PathIndex(i int) string {
	p := strings.Split(strings.Trim(ctx.Path(), "/"), "/")
	if len(p) <= i {
		return ""
	}
	return p[i]
}
func (ctx *Context) PathMid(start, length int) string {
	p := ctx.Path()
	if "/" == p {
		return p
	}
	trimedPath := strings.Trim(p, "/")
	pcount := strings.Count(trimedPath, "/") + 1
	if start >= pcount {
		return ""
	}
	end := start + length
	ps := strings.Split(trimedPath, "/")
	if end > len(ps) {
		end = len(ps)
	}
	return "/" + strings.Join(ps[start:start+length], "/")
}
func (ctx *Context) PathMatch(pattern string) bool {
	r, err := regexp.MatchString(pattern, ctx.Path())
	if err != nil {
		log.Error().Func("PathMatch").Err(err).Stack().Str("pattern", pattern).Msg(err.Error())
		return false
	}
	return r
}

func (ctx *Context) RedirectSignin(signinUrl string, needRedirectFrom bool) {
	// signinUrl := "/signin"
	p := ctx.RequestPath(true)
	if needRedirectFrom && signinUrl != p {
		ctx.RedirectWithFrom(signinUrl)
	}
	ctx.Redirect(signinUrl)
}
func (ctx *Context) RedirectWithFrom(uri string) {
	p := ctx.Request().URL.EscapedPath() + "?" + ctx.Request().URL.RawQuery
	r, _ := url.Parse(uri)
	q := r.Query()
	q.Add("redirect_from", url.PathEscape(p))
	r.RawQuery = q.Encode()
	ctx.Redirect(r.String())
}

func (ctx *Context) QueryString() string {
	return ctx.Request().URL.RawQuery
}

// func formatUploadFileName(filename string) string {
// 	return strconv.FormatInt(time.Now().Unix(), 10) + "-" + filename
// }

func (ctx *Context) ProxyPass(proxy, path string) error {
	target, err := url.Parse(proxy)
	if err != nil {
		log.Error().Func("ProxyPass").Err(err).Stack().Str("proxy", proxy).Str("path", path).Msg(err.Error())
		return err
	}
	p := host.ProxyHandler(target)
	req := ctx.Request()
	req.URL.Path = path
	p.ServeHTTP(ctx.ResponseWriter(), req)
	return nil
}

func (ctx *Context) CheckQuery(field string) *Validator {
	return NewValidator(ctx, field, ctx.URLParam(field), ctx.URLParamExists(field))
}
func (ctx *Context) CheckBody(field string) *Validator {
	_, ok := ctx.FormValues()[field]
	return NewValidator(ctx, field, ctx.FormValue(field), ok)
}
func (ctx *Context) CheckBodyValues(field string) *ValidatorValues {
	values, ok := ctx.FormValues()[field]
	return NewValidatorValues(ctx, field, values, ok)
}
func (ctx *Context) CheckPath(field string) *Validator {
	return NewValidator(ctx, field, ctx.Params().Get(field), ctx.Params().GetEntry(field).Key != "")
}
func (ctx *Context) CheckFile(field string) *ValidatorFile {
	src, header, err := ctx.FormFile(field)
	return NewValidatorFile(ctx, field, src, header, err)
}

func (ctx *Context) AddScript(js string) string {
	ctx.Values().Set(RequestKeyScripts, append(ctx.Scripts(), js))
	return ""
}

func (ctx *Context) SetUid(uid interface{}, secs int) {
	err := ctx.SessionProvider.Set(ctx.Sid()+"/"+ctx.SessionProvider.UidKey(), uid, secs)
	if err != nil {
		log.Error().Func("SetUid").Err(err).Msg(err.Error())
		return
	}
	ctx.Values().Set(ctx.SessionProvider.UidKey(), uid)
}
func (ctx *Context) GetUid(uid interface{}) error {
	err := ctx.SessionProvider.Get(ctx.Sid()+"/"+ctx.SessionProvider.UidKey(), uid)
	if err != nil {
		log.Error().Func("GetUid").Err(err).Msg(err.Error())
		return err
	}
	return nil
}
func (ctx *Context) GetUidInt() int {
	var uid int
	var err error
	uid, err = ctx.Values().GetInt(ctx.SessionProvider.UidKey())
	if err != nil || uid == 0 {
		err = ctx.SessionProvider.Get(ctx.Sid()+"/"+ctx.SessionProvider.UidKey(), &uid)
		if err != nil {
			log.Error().Func("GetUid").Err(err).Msg(err.Error())
			return 0
		}
		ctx.Values().Set(ctx.SessionProvider.UidKey(), uid)
	}
	return uid
}
func (ctx *Context) GetUidInt64() int64 {
	var uid int64
	var err error
	uid, err = ctx.Values().GetInt64(ctx.SessionProvider.UidKey())
	if err != nil || uid == 0 {
		err = ctx.SessionProvider.Get(ctx.Sid()+"/"+ctx.SessionProvider.UidKey(), &uid)
		if err != nil {
			log.Error().Func("GetUidInt64").Err(err).Msg(err.Error())
			return 0
		}
		ctx.Values().Set(ctx.SessionProvider.UidKey(), uid)
	}
	return uid
}
func (ctx *Context) GetUidString() string {
	var uid string
	var err error
	uid = ctx.Values().GetString(ctx.SessionProvider.UidKey())
	if uid == "" {
		err = ctx.SessionProvider.Get(ctx.Sid()+"/"+ctx.SessionProvider.UidKey(), &uid)
		if err != nil {
			log.Error().Func("GetUidString").Err(err).Msg(err.Error())
			return ""
		}
		ctx.Values().Set(ctx.SessionProvider.UidKey(), uid)
	}
	return uid
}
func (ctx *Context) Sid() string {
	return ctx.SessionProvider.GetSessionId(ctx)
}
func (ctx *Context) HasSignin() bool {
	var uid interface{}
	ctx.GetUid(&uid)
	return uid != nil
}

func SidFilter(ctx iris.Context) {
	c := ctx.(*Context)
	sid := c.Sid()
	if sid != "" {
		ctx.Next()
		return
	}
	c.SessionProvider.SetSessionId(c)
	ctx.Next()
	// c.SetCookieSid(GenCookieSid())
}
