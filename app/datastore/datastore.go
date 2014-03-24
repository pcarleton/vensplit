package datastore

import (
  "appengine/datastore"
  "appengine"
  "code.google.com/p/goauth2/oauth"
  "time"
)

type Datastore struct {
  context appengine.Context
}

type Config struct {
  ClientID string
  ClientSecret string
}

// The same as oauth token minus the dict of extra so we can store it.
type Token struct {
  AccessToken string
  RefreshToken string
  Expiry time.Time
}

func storageToken(token *oauth.Token) *Token {
  return &Token{token.AccessToken, token.RefreshToken, token.Expiry}
}

func New(ctx appengine.Context) Datastore {
  return Datastore{ctx}
}

func (d *Datastore) configKey() *datastore.Key {
  return datastore.NewKey(d.context, "config", "singleton", 0, nil)
}

func (d *Datastore) SaveConfig(cfg Config) error {
  _, err := datastore.Put(d.context, d.configKey(), &cfg)
  return err
}

func (d *Datastore) Config() (Config, error) {
  var cfg Config
  err := datastore.Get(d.context, d.configKey(), &cfg)
  return cfg, err
}

func (d *Datastore) tokenKey(userID string) *datastore.Key {
  return datastore.NewKey(d.context, "token", userID, 0, nil)
}

func (d *Datastore) SaveToken(userID string, token *oauth.Token) error {
  _, err := datastore.Put(d.context, d.tokenKey(userID), storageToken(token))
  return err
}

func (d *Datastore) FindToken(userID string) (oauth.Token, error) {
  st := Token{}
  err := datastore.Get(d.context, d.tokenKey(userID), &st)
  return oauth.Token{
    AccessToken: st.AccessToken,
    RefreshToken: st.RefreshToken,
    Expiry: st.Expiry}, err
}
