package irisx

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/RocksonZeta/wrap/rediswrap"
	"github.com/RocksonZeta/wrap/utils/encryptutil"
	"github.com/RocksonZeta/wrap/utils/hashutil"
	"github.com/RocksonZeta/wrap/utils/mathutil"
	"github.com/RocksonZeta/wrap/utils/timeutil"
	"github.com/kataras/iris/v12"
)

const (
	SidFromCookie = 1
	SidFromHeader = 2
)

type SessionOptions struct {
	DatabaseUrl     string // eg: redis://:password@localhost:1234/1?PoolSize=10
	Prefix          string //session id prefix  prefix/sessionid
	Password        string // password for encrypt sid to token : enc(sid,password)->token,dec(token,password)->sid
	HeaderTokenName string // like X-USER-TOKEN
	CookieSidName   string // like sessionid
	CookieTokenName string //like token
	CookieDomain    string
	SessionTTL      int // session lifttime in seconds
	TokenTTL        int // token lifttime in seconds
	LruMaxLength    int
	KeyUid          string
}

type Sessions struct {
	Options    SessionOptions
	Encypter   SessionTokenEncypter
	Storage    SessionStorage
	Marshaller SessionValueMarshaller
	GetUid     func(sid string, sidForm int) (interface{}, error)
	// lru        *lru.Lru
}

func NewSessions(options SessionOptions) *Sessions {
	s := &Sessions{
		Options:    options,
		Encypter:   new(SessionTokenEncypterAES),
		Marshaller: new(SessionValueMarshallerJson),
	}
	s.Storage = &SessionStorageRedis{
		db: rediswrap.NewFromUrl(s.Options.DatabaseUrl),
	}
	s.defaultOptions()
	// s.lru = lru.New(lru.Options{Ttl: s.Options.SessionTTL, MaxAge: s.Options.SessionTTL * 2, MaxLength: 100000})
	return s
}

func (s *Sessions) defaultOptions() {
	if s.Options.CookieSidName == "" {
		s.Options.CookieSidName = "sessionid"
	}
	if s.Options.Prefix == "" {
		s.Options.Prefix = "session"
	}
	if s.Options.HeaderTokenName == "" {
		s.Options.HeaderTokenName = "X-USER-TOKEN"
	}
	if s.Options.CookieTokenName == "" {
		s.Options.CookieTokenName = "usertoken"
	}
	if s.Options.SessionTTL <= 0 {
		s.Options.SessionTTL = 3600
	}
	if s.Options.TokenTTL <= 0 {
		s.Options.TokenTTL = 3600 * 24 * 14
	}
	if s.Options.LruMaxLength <= 0 {
		s.Options.LruMaxLength = 100000
	}
	if s.Options.KeyUid == "" {
		s.Options.KeyUid = "uid"
	}
}
func (s *Sessions) Filter(ctx iris.Context) {
	c := ctx.(*Context)
	c.sessions = s
	sidFrom := SidFromHeader
	token := c.GetHeader(s.Options.HeaderTokenName)
	if token == "" {
		sidFrom = SidFromCookie
		tokenCookie, _ := c.Request().Cookie(s.Options.CookieTokenName)
		if nil != tokenCookie {
			token = tokenCookie.Value
		}
	}
	if token != "" {
		sid, err := s.ParseToken(token, SidFromHeader)
		if err != nil {
			log.Error().Func("Filter").Msg(err.Error())
			c.StatusCode(http.StatusForbidden)
			c.EndRequest()
			return
		}
		uid, err := s.GetUid(sid, sidFrom)
		if err != nil {
			log.Error().Func("Filter").Msg(err.Error())
			c.StatusCode(http.StatusForbidden)
			c.RemoveCookie(s.Options.CookieTokenName)
			c.EndRequest()
			return
		}
		c.session = s.newSession(sid, c)
		c.session.SetUid(uid)
		c.Next()
		return
	}

	var sessionToken string
	// sessionToken := c.GetCookie(s.Options.CookieSidName)
	sessionTokenCookie, _ := c.Request().Cookie(s.Options.CookieSidName)
	var sid string
	var err error
	if sessionTokenCookie != nil {
		sessionToken = sessionTokenCookie.Value
		sid, err = s.ParseToken(sessionToken, SidFromCookie)
		if err != nil {
			log.Error().Func("Filter").Str("sessionToken", sessionToken).Msg(err.Error())
			c.StatusCode(http.StatusBadRequest)
			c.RemoveCookie(s.Options.CookieSidName)
			c.EndRequest()
			return
		}
		sessionToken = s.MakeToken(sid)
	} else {
		sid, sessionToken = s.NewToken()
	}
	c.session = s.newSession(sid, c)
	c.session.ResetExpire()
	c.SetCookieLocal(s.Options.CookieSidName, sessionToken, s.Options.SessionTTL, true, s.Options.CookieDomain)
	c.Next()

}

func (s *Sessions) newSession(sid string, ctx *Context) *Session {
	newSession := &Session{
		id:       sid,
		sessions: s,
		ctx:      ctx,
		// prefix:   s.Options.Prefix,
		// // createTime: timeutil.NowInt(),
		// ttl:        s.Options.SessionTTL,
		// mashaller:  s.Marshaller,
		// storage:    s.Storage,
		// keyUid:     s.Options.KeyUid,
	}

	return newSession
}
func (s Sessions) ParseToken(token string, src int) (string, error) {
	sid, err := s.Encypter.Decrypt(token, s.Options.Password)
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(sid, "/")
	timeStr := sid[i+1:]
	t, err := strconv.ParseInt(timeStr, 10, 64)
	ttl := int64(s.Options.SessionTTL)
	if src == SidFromHeader {
		ttl = int64(s.Options.TokenTTL)
	}
	if time.Now().Unix()-t > ttl {
		return "", errors.New("user token has expired")
	}
	return sid[0:i], err
}
func (s Sessions) NewToken() (string, string) {
	id := mathutil.RandomStr(32, false)
	ct := timeutil.NowStr()
	str, _ := s.Encypter.Encrypt(id+"/"+ct, s.Options.Password)
	return id, str
}
func (s Sessions) MakeToken(sid string) string {
	ct := timeutil.NowStr()
	str, _ := s.Encypter.Encrypt(sid+"/"+ct, s.Options.Password)
	return str
}

// type Session interface {
// 	ID() string
// 	Destroy()
// 	Set(key string, value interface{})
// 	Get(key string, valueAddr interface{})
// 	SetLru(key string, value interface{})
// 	GetLru(key string, valueAddr interface{})
// 	Delete(key string)
// 	ResetExpire()
// }
type Session struct {
	id       string // session id , can be parse from token
	sessions *Sessions
	ctx      *Context
	// prefix string
	// createTime int // seconds
	// updateTime int // seconds ,Get Set SesetExpire will update this field to current time
	// ttl        int //seconds
	// storage    SessionStorage
	// mashaller  SessionValueMarshaller
	// lru *lru.Lru
	// keyUid     string
}

func (s Session) skey() string {
	return s.sessions.Options.Prefix + "/" + s.id
}
func (s Session) SID() string {
	return s.id
}
func (s Session) Destroy() {
	s.sessions.Storage.Destroy(s.skey())
}
func (s Session) ResetExpire() {
	s.sessions.Storage.SetExpire(s.skey(), s.sessions.Options.SessionTTL)
}
func (s Session) Set(key string, value interface{}) error {
	str, err := s.sessions.Marshaller.Marshal(value)
	if err != nil {
		return err
	}
	s.sessions.Storage.Set(s.skey(), key, str)
	// s.ResetExpire()
	return nil
}
func (s Session) GetAll(key string) (map[string]string, error) {
	m := s.sessions.Storage.GetAll(s.skey())
	s.ResetExpire()
	return m, nil
}
func (s Session) Get(key string, result interface{}) error {
	str := s.sessions.Storage.Get(s.skey(), key)
	if str == "" {
		return nil
	}
	err := s.sessions.Marshaller.Ummarshal(str, result)
	// s.ResetExpire()
	return err
}

func (s Session) GetInt(key string) (int, error) {
	var r int
	err := s.Get(key, &r)
	return r, err
}
func (s Session) GetInt64(key string) (int64, error) {
	var r int64
	err := s.Get(key, &r)
	return r, err
}
func (s Session) GetString(key string) (string, error) {
	var r string
	err := s.Get(key, &r)
	return r, err
}

// func (s Session) lruKey(key string) string {
// 	return s.skey() + "/" + key
// }

// func (s Session) SetLru(key string, value interface{}) error {
// 	s.lru.Add(s.lruKey(key), value)
// 	return s.Set(key, value)
// }
// func (s Session) GetLru(key string, resultAddr interface{}) error {
// 	result := s.lru.Get(s.lruKey(key))
// 	if result != nil {
// 		reflect.ValueOf(resultAddr).Elem().Set(reflect.ValueOf(result))
// 		s.ResetExpire()
// 		return nil
// 	} else {
// 		err := s.Get(key, resultAddr)
// 		s.lru.Add(s.lruKey(key), reflect.ValueOf(resultAddr).Elem().Interface())
// 		return err
// 	}
// }
// func (s Session) GetLruInt(key string) (int, error) {
// 	result := s.lru.Get(s.lruKey(key))
// 	if result != nil {
// 		return result.(int), nil
// 	} else {
// 		return s.GetInt(key)
// 	}
// }
// func (s Session) GetLruInt64(key string) (int64, error) {
// 	result := s.lru.Get(s.lruKey(key))
// 	if result != nil {
// 		return result.(int64), nil
// 	} else {
// 		return s.GetInt64(key)
// 	}
// }
// func (s Session) GetLruString(key string) (string, error) {
// 	result := s.lru.Get(s.lruKey(key))
// 	if result != nil {
// 		return result.(string), nil
// 	} else {
// 		return s.GetString(key)
// 	}
// }
func (s Session) SetUid(uid interface{}) error {
	// s.ctx.uid = uid
	s.ctx.Values().Set(s.sessions.Options.KeyUid, uid)
	return s.Set(s.sessions.Options.KeyUid, uid)
}
func (s Session) GetUid(uidAddr interface{}) error {
	uid := s.ctx.Values().Get(s.sessions.Options.KeyUid)
	// if s.ctx.uid != nil {
	if uid != nil {
		reflect.ValueOf(uidAddr).Elem().Set(reflect.ValueOf(uid))
		return nil
	}
	// err := s.GetLru(s.sessions.Options.KeyUid, uidAddr)
	err := s.Get(s.sessions.Options.KeyUid, uidAddr)
	s.ctx.Values().Set(s.sessions.Options.KeyUid, reflect.ValueOf(uidAddr).Elem().Interface())
	return err
}
func (s Session) GetUidInt() int {
	uid, _ := s.ctx.Values().GetInt(s.sessions.Options.KeyUid)
	if uid != 0 {
		return uid
	}
	uid, err := s.GetInt(s.sessions.Options.KeyUid)
	if err != nil {
		log.Error().Func("GetUidInt").Msg(err.Error())
	}
	s.ctx.Values().Set(s.sessions.Options.KeyUid, uid)
	return uid
}
func (s Session) GetUidInt64() int64 {
	uid, _ := s.ctx.Values().GetInt64(s.sessions.Options.KeyUid)
	if uid != 0 {
		return uid
	}
	uid, err := s.GetInt64(s.sessions.Options.KeyUid)
	if err != nil {
		log.Error().Func("GetUidInt64").Msg(err.Error())
	}
	s.ctx.Values().Set(s.sessions.Options.KeyUid, uid)
	return uid
}
func (s Session) GetUidString() string {
	uid := s.ctx.Values().GetString(s.sessions.Options.KeyUid)
	if uid != "" {
		return uid
	}
	uid, err := s.GetString(s.sessions.Options.KeyUid)
	if err != nil {
		log.Error().Func("GetUidString").Msg(err.Error())
	}
	s.ctx.Values().Set(s.sessions.Options.KeyUid, uid)
	return uid
}

type SessionTokenEncypter interface {
	Encrypt(str, password string) (string, error)
	Decrypt(str, password string) (string, error)
}

type SessionTokenEncypterAES struct {
}

func (s *SessionTokenEncypterAES) Encrypt(str, password string) (string, error) {
	return encryptutil.AESCBCPKCS5EncryptBase64(hashutil.Md5(password), str)
}
func (s *SessionTokenEncypterAES) Decrypt(str, password string) (string, error) {
	return encryptutil.AESCBCPKCS5DecryptBase64(hashutil.Md5(password), str)
}

type SessionValueMarshaller interface {
	Marshal(value interface{}) (string, error)
	Ummarshal(str string, valueAddr interface{}) error
}
type SessionValueMarshallerJson struct {
}

func (s *SessionValueMarshallerJson) Marshal(value interface{}) (string, error) {
	bs, err := json.Marshal(value)
	return string(bs), err
}
func (s *SessionValueMarshallerJson) Ummarshal(str string, valueAddr interface{}) error {
	return json.Unmarshal([]byte(str), valueAddr)
}

type SessionStorage interface {
	Set(sid, key, value string)
	Get(sid, key string) string
	GetAll(sid string) map[string]string
	SetExpire(sid string, seconds int)
	// GetExpire(sid string) int
	Delete(sid string, key string)
	Destroy(sid string)
}

type SessionStorageRedis struct {
	db *rediswrap.Redis
}

func (s *SessionStorageRedis) Set(sid, key, value string) {
	s.db.HSet(sid, key, value)
}
func (s *SessionStorageRedis) Get(sid, key string) string {
	return s.db.HGet(sid, key).Val()
}
func (s *SessionStorageRedis) GetAll(sid string) map[string]string {
	return s.db.HGetAll(sid).Val()
}
func (s *SessionStorageRedis) SetExpire(sid string, seconds int) {
	s.db.Expire(sid, time.Duration(seconds)*time.Second)
}
func (s *SessionStorageRedis) Delete(sid string, key string) {
	s.db.HDel(sid, key)
}
func (s *SessionStorageRedis) Destroy(sid string) {
	s.db.Del(sid)
}
