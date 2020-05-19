package irisx

import (
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/RocksonZeta/wrap/errs"
	"github.com/RocksonZeta/wrap/wraplog"
	"github.com/asaskevich/govalidator"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/host"
)

var pkg = reflect.TypeOf(BreadCrumb{}).PkgPath()
var log = wraplog.Logger.Fork(pkg, "Context")

const (
	RequestKeyParamErrors = "ParamErrors"
	RequestKeyScripts     = "Scripts"
	RequestKeyBreadCrumbs = "BreadCrumbs"
)

// type H map[string]interface{}
type Context struct {
	context.Context
	sessions *Sessions
	// session         *sessions.Session
	PageSize int
	// AutoIncludeCss bool
	// AutoIncludeJs  bool
	// AutoHead       bool
	// Owner          table.User
	// Error           *errors.PageError
	// ParamErrors map[string]string
	// BreadCrumbs []BreadCrumb
	// Scripts     []string
	// sid         string //sessionId
	// values sync.Map
	// uid         int
	session    *Session
	BeforeView func(ctx *Context, tplFile string)
	// uid        interface{}
	// sid        string
}

var headers sync.Map

type BreadCrumb struct {
	Title string
	Url   string
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

// func (ctx *Context) HasSignin() bool {
// 	return ctx.Uid() > 0
// }

func (ctx *Context) Sessions() *Sessions {
	return ctx.sessions
}
func (ctx *Context) Session() *Session {
	return ctx.session
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
func (ctx *Context) BreadCrumbs() []BreadCrumb {
	r := ctx.Values().Get(RequestKeyBreadCrumbs)
	if r == nil {
		return nil
	}
	return r.([]BreadCrumb)
}

// func (ctx *Context) GetUidInt() int {
// 	return ctx.Uid.(int)
// }
// func (ctx *Context) GetUidString() string {
// 	return ctx.Uid.(string)
// }
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

type PageData struct {
	Items     interface{}
	Total     int64 //共多少条
	PageIndex int   //当前页
	PageCount int   //共多少页
	PageSize  int   //一页显示多少条
}

func (ctx *Context) OkPageDefault(data interface{}, total int64) {
	pi := ctx.CheckQuery("PageIndex").Empty().Int(0)
	ctx.OkPage(data, total, pi, 0)
}
func (ctx *Context) OkPage(data interface{}, total int64, pageIndex, pageSize int) {
	if pageSize == 0 {
		pageSize = ctx.PageSize
	}
	var pc int
	if pageSize != 0 {
		pc = int(total) / pageSize
		if pc*pageSize != int(total) {
			pc++
		}
	}
	ctx.Ok(PageData{Items: data, Total: total, PageIndex: pageIndex, PageSize: pageSize, PageCount: pc})
}

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

func (ctx *Context) RedirectSignin(needRedirectFrom bool) {
	signinUrl := "/signin"
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

// func (ctx *Context) Signout() {
// 	ctx.RemoveCookieLocal(SessionCookieId)
// 	ctx.RemoveCookieLocal(SessionCookieTokenId)
// 	if ctx.Session != nil {
// 		localSessionUids.Delete(ctx.Session.Sid)
// 	}
// }

// func (ctx *Context) CookieSet(name, value string, maxAge time.Duration) {
// 	ctx.SetCookieKV(name, value, context.CookiePath("/"), context.CookieHTTPOnly(false), context.CookieExpires(maxAge))
// }

// func (ctx *Context) IsJson() bool {
// 	return ctx.URLParamExists("json") || ctx.GetHeader("return") == "json" || strings.HasPrefix(ctx.Path(), "/json")
// }

// func saveUploadedFile(src io.Reader, fname string) (int64, error) {
// 	// src, err := fh.Open()
// 	// if err != nil {
// 	// 	return 0, err
// 	// }
// 	// defer src.Close()

// 	dst, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, os.FileMode(0666))

// 	if err != nil {
// 		return 0, err
// 	}
// 	defer dst.Close()

// 	return io.Copy(dst, src)
// }

// type FileHeader struct {
// 	*multipart.FileHeader
// 	SavedFile string
// }

// func (ctx *Context) saveFileTmp(field string) (*FileHeader, error) {
// 	file, info, err := ctx.FormFile(field)
// 	if file == nil || info.Size <= 0 {
// 		return nil, nil
// 	}
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()
// 	fh := &FileHeader{FileHeader: info}
// 	fh.SavedFile = filepath.Join(config.Config.Http.UploadTmpDir, formatUploadFileName(info.Filename))
// 	saveUploadedFile(file, fh.SavedFile)
// 	return fh, nil
// }

func formatUploadFileName(filename string) string {
	return strconv.FormatInt(time.Now().Unix(), 10) + "-" + filename
}

// func (ctx *Context) saveFilesTmp() ([]FileHeader, int, error) {
// 	//app.Run(iris.Addr(":8080") /* 0.*/, iris.WithPostMaxMemory(maxSize))
// 	maxSize := ctx.Application().ConfigurationReadOnly().GetPostMaxMemory()

// 	err := ctx.Request().ParseMultipartForm(maxSize)
// 	if err != nil {
// 		log.Log.Error(ctx.Path() + " Context.SaveFiles - parse multipart form error")
// 		return nil, 1, err
// 	}

// 	form := ctx.Request().MultipartForm

// 	var fhs []FileHeader
// 	files := form.File["files[]"]
// 	failures := 0
// 	for _, file := range files {
// 		src, err := file.Open()
// 		if err != nil {
// 			failures++
// 			log.Log.Error(ctx.Path() + " Context.SaveFiles - open file error ")
// 			continue
// 		}
// 		defer src.Close()
// 		// fname := formatUploadFileName(file.Filename)
// 		fname := filepath.Join(config.Config.Http.UploadTmpDir, formatUploadFileName(file.Filename))
// 		_, err = saveUploadedFile(src, fname)
// 		if err != nil {
// 			failures++
// 			log.Log.Error(ctx.Path() + " Context.SaveFiles - save file error " + file.Filename)
// 			continue
// 		}
// 		fh := FileHeader{FileHeader: file, SavedFile: fname}
// 		fhs = append(fhs, fh)
// 	}
// 	return fhs, failures, nil
// }

// func (ctx *Context) SaveFile(field, collection string) (httpfsclient.HfLink, *FileHeader, error) {
// 	fh, err := ctx.saveFileTmp(field)
// 	if fh == nil {
// 		return "", nil, nil
// 	}
// 	if err != nil {
// 		return "", nil, err
// 	}
// 	defer os.Remove(fh.SavedFile)
// 	reader, err := fh.Open()
// 	if err != nil {
// 		return "", nil, err
// 	}
// 	defer reader.Close()
// 	link, err := httpfsclient.Write(reader, config.Config.Clusters.Static, fh.Filename, collection)
// 	return link, fh, err
// }

//SaveImage return [原图，裁剪图，压缩后的图]
// func (ctx *Context) SaveImage(field string, crop []int, sizes [][]int) ([]httpfsclient.HfLink, *FileHeader, error) {
// 	return ctx.SaveImageCollection(field, crop, sizes, httpfsclient.CollectionImage)
// }
// func (ctx *Context) SaveImageCollection(field string, crop []int, sizes [][]int, collection string) ([]httpfsclient.HfLink, *FileHeader, error) {
// 	fh, err := ctx.saveFileTmp(field)
// 	if nil == fh {
// 		return nil, nil, nil
// 	}
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	defer os.Remove(fh.SavedFile)

// 	reader, err := fh.Open()
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	defer reader.Close()
// 	hflink, err := httpfsclient.Write(reader, config.Config.Clusters.Static, fh.Filename, collection)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	r := []httpfsclient.HfLink{hflink}
// 	if len(crop) > 0 || len(sizes) > 0 {
// 		images, err := hflink.ImageResize(crop, sizes)
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 		return append(r, images...), fh, nil
// 	}
// 	return r, fh, nil
// }
// func (ctx *Context) SaveVideo(field string) (httpfsclient.HfLink, *FileHeader, error) {
// 	fh, err := ctx.saveFileTmp(field)
// 	if fh == nil {
// 		return "", nil, nil
// 	}
// 	if err != nil {
// 		return "", nil, err
// 	}
// 	defer os.Remove(fh.SavedFile)
// 	reader, err := fh.Open()
// 	if err != nil {
// 		return "", nil, err
// 	}
// 	defer reader.Close()
// 	link, err := httpfsclient.Write(reader, config.Config.Clusters.Static, fh.Filename, httpfsclient.CollectionVideo)
// 	return link, fh, err
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

// func (ctx *Context) RedirectToReferer() {
// 	ctx.Redirect(ctx.GetHeader("Referer"))
// }

// func (ctx *Context) FormBool(name string, dv bool) bool {
// 	form := ctx.FormValues()
// 	if vs, ok := form[name]; ok {
// 		if len(vs) > 0 {
// 			r, err := strconv.ParseBool(vs[0])
// 			if err != nil {
// 				log.Log.Error(err)
// 			}
// 			return r
// 		} else {
// 			return dv
// 		}
// 	}
// 	return dv
// }
// func (ctx *Context) FormInt(name string, dv int) int {
// 	form := ctx.FormValues()
// 	if vs, ok := form[name]; ok {
// 		if len(vs) > 0 {
// 			r, err := strconv.Atoi(vs[0])
// 			if err != nil {
// 				log.Log.Error(err)
// 			}
// 			return r
// 		} else {
// 			return dv
// 		}
// 	}
// 	return dv
// }
// func (ctx *Context) FormInt64(name string, dv int64) int64 {
// 	form := ctx.FormValues()
// 	if vs, ok := form[name]; ok {
// 		if len(vs) > 0 {
// 			r, err := strconv.ParseInt(vs[0], 10, 64)
// 			if err != nil {
// 				log.Log.Error(err)
// 			}
// 			return r
// 		} else {
// 			return dv
// 		}
// 	}
// 	return dv
// }
// func (ctx *Context) FormNullInt(name string) null.Int {
// 	form := ctx.FormValues()
// 	if vs, ok := form[name]; ok {
// 		if len(vs) > 0 {
// 			r, _ := strconv.ParseInt(vs[0], 10, 64)
// 			return null.IntFrom(r)
// 		}
// 	}
// 	return null.Int{}
// }

// func (ctx *Context) FormFloat32(name string, dv float32) float32 {
// 	form := ctx.FormValues()
// 	if vs, ok := form[name]; ok {
// 		if len(vs) > 0 {
// 			r, err := strconv.ParseFloat(vs[0], 32)
// 			if err != nil {
// 				log.Log.Error(err)
// 			}
// 			return float32(r)
// 		} else {
// 			return dv
// 		}
// 	}
// 	return dv
// }
// func (ctx *Context) FormFloat64(name string, dv float64) float64 {
// 	form := ctx.FormValues()
// 	if vs, ok := form[name]; ok {
// 		if len(vs) > 0 {
// 			r, err := strconv.ParseFloat(vs[0], 64)
// 			if err != nil {
// 				log.Log.Error(err)
// 			}
// 			return r
// 		} else {
// 			return dv
// 		}
// 	}
// 	return dv
// }

// func (ctx *Context) SaveFile(field, collection string) (fsutil.File, *multipart.FileHeader, error) {
// 	f, h, err := ctx.FormFile(field)
// 	defer f.Close()
// 	if err != nil {
// 		return fsutil.File{}, h, err
// 	}
// 	if h.Size <= 0 {
// 		return fsutil.File{}, h, err
// 	}
// 	nf := fsutil.NewFileRandom(config.Config.Dfs.Static, collection, filepath.Ext(h.Filename))
// 	err = nf.Write(f)
// 	return nf, h, err
// }

// type SaveImageResult struct {
// 	Raw     fsutil.File
// 	Crop    fsutil.File
// 	Resizes []fsutil.File
// 	Header  *multipart.FileHeader
// 	Error   error
// }

//SaveImage 保存图片 crop :[x,y,w,h] , sizes:[[100,100],[200,300]]
// func (ctx *Context) SaveImage(field, collection string, crop []int, sizes [][]int) SaveImageResult {
// 	var r SaveImageResult
// 	r.Raw, r.Header, r.Error = ctx.SaveFile(field, collection)
// 	if r.Error != nil || r.Header.Size <= 0 {
// 		return r
// 	}
// 	if len(crop) >= 4 {
// 		r.Crop, r.Error = r.Raw.ImgCrop(crop[0], crop[1], crop[2], crop[3])
// 		if r.Error != nil {
// 			return r
// 		}
// 	}
// 	if len(sizes) > 0 {
// 		if r.Crop.RPath != "" {
// 			r.Resizes, r.Error = r.Crop.ImgResizeKeepRatio(sizes)
// 		} else {
// 			r.Resizes, r.Error = r.Raw.ImgResizeKeepRatio(sizes)
// 		}
// 	}
// 	return r
// }

// func (ctx *Context) Check() {
// 	if len(ctx.ParamErrors()) > 0 {

// 		// panic(errors.PageError{Err: errors.Err{State: errorcode.HttpParamError}, FieldError: ctx.ParamErrors})
// 	}
// }

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
	return NewValidatorFile(ctx, field, src, header, err != nil)
}

func (ctx *Context) PushBreadCrumb(title, url string) {
	ctx.Values().Set(RequestKeyBreadCrumbs, append(ctx.BreadCrumbs(), BreadCrumb{Title: title, Url: url}))
}

func (ctx *Context) AddScript(js string) string {
	ctx.Values().Set(RequestKeyScripts, append(ctx.Scripts(), js))
	return ""
}
