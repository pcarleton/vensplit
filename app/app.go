package app

import (
  "fmt"
  "net/http"
  "appengine"
  "appengine/urlfetch"
  "appengine/user"
  "encoding/json"
  "github.com/pcarleton/vengo/api"
  "github.com/pcarleton/vensplit/app/datastore"
  "code.google.com/p/goauth2/oauth"
)

var (
  cfg = oauth.Config {
    Scope: "make_payments,access_friends,access_profile",
    AuthURL: "https://api.venmo.com/v1/oauth/authorize",
    TokenURL: "https://api.venmo.com/v1/oauth/access_token",
  }
)

func init() {
  http.Handle("/data", appHandler(handler))
  http.Handle("/", appHandler(oauthHandler))
  http.Handle("/savetoken", appHandler(saveToken))
  http.Handle("/saveconfig", appHandler(saveConfigHandler))
}

func handler(w http.ResponseWriter, r *http.Request) *appError {
  c := appengine.NewContext(r)
  d := datastore.New(c)
  token, err := d.FindToken(user.Current(c).Email)
  if err != nil {
    return &appError{err, "Error fetching token.", 500}
  }

  transport := oauth.Transport{
    Token: &token,
    Config: &cfg,
    Transport: urlfetch.Client(c).Transport,
  }

  svc := api.NewFromClient(transport.Client(), token.AccessToken)
  
  meInfo, err := svc.Me()
  if err != nil {
    return &appError{err, "Error getting me info.", 500}
  }
  res, err := svc.ListFriends(meInfo.Data.User.ID, &api.ListFriendsRequest{
  Limit: "200"})
  if err != nil {
    return &appError{err, "Error getting friends info.", 500}
  }
  penc, _ := json.Marshal(res.Data)
  fmt.Fprintf(w, "%s", penc)
  return nil
}

func oauthHandler(w http.ResponseWriter, r *http.Request) *appError {
  if appErr := loadConfig(r); appErr != nil {
    return appErr
  }
  destination := cfg.AuthCodeURL("savetoken") + "&response_type=code"
  http.Redirect(w, r, destination, http.StatusFound)
  return nil
}

func saveToken(w http.ResponseWriter, r *http.Request) *appError {
  c := appengine.NewContext(r)
  if appErr := loadConfig(r); appErr != nil {
    return appErr
  }
  if user.Current(c) == nil {
    return &appError{nil, "Must be signed in to save token.", 400}
  }
  code := r.FormValue("code")
  if code == "" {
    return &appError{nil, "No 'code' parameter found", 500}
  }
  t := &oauth.Transport{
    Config: &cfg,
    Transport: urlfetch.Client(c).Transport,
  }

  if _, err := t.Exchange(code); err != nil {
    return &appError{err, "Error exchanging code for token.", 500}
  }

  d := datastore.New(c)
  if err := d.SaveToken(user.Current(c).Email, t.Token); err != nil {
    return &appError{err, "Error saving token.", 500}
  }
  http.Redirect(w, r, "/app", http.StatusFound)
  return nil
}

func saveConfigHandler(w http.ResponseWriter, r *http.Request) *appError {
  d := datastore.New(appengine.NewContext(r))
  cfg := datastore.Config{}
  cfg.ClientID = r.FormValue("ClientID")
  cfg.ClientSecret = r.FormValue("ClientSecret")
  if err := d.SaveConfig(cfg); err != nil {
    return &appError{err, "Error saving config", 500}
  }
  fmt.Fprintf(w, "Saved config!")
  return nil
}

func loadConfig(r *http.Request) *appError {
  d := datastore.New(appengine.NewContext(r))
  params, err := d.Config()
  if err != nil {
    return &appError{err, "Error loading config from datastore.", 500}
  }
  cfg.ClientId = params.ClientID
  cfg.ClientSecret = params.ClientSecret
  return nil
}

type appHandler func(http.ResponseWriter, *http.Request) *appError

type appError struct {
  Error error
  Message string
  Code int
}

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  if e := fn(w, r); e != nil {
    c := appengine.NewContext(r)
    c.Errorf("%v", e)
    http.Error(w, e.Message, e.Code)
  }
}
