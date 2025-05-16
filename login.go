package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	LOGIN_URI = "/korisnik.php"
)

func checkLogin(resp *http.Response) error {
	//TODO kod koji proverava da li je logovanje uspelo
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.New("prazan odgovor")
	}
	if string(body) == "Vas korisnicki nalog je iskljucen sa foruma." {
		return ErrBanned
	}
	return nil
}

/**
* Prijava na sajt
 */
var ErrBanned = errors.New("korisnik je banovan")

func login(c *http.Client, uri string, loginData url.Values) error {
	req, err := http.NewRequest(http.MethodPost, BASE_URL+uri, strings.NewReader(loginData.Encode()))
	if err != nil {
		return err
	}
	addHeaders(&req.Header)

	//Konekcija je potrebna funkciji cookieHack koja se poziva u okviru CustomTransport-a
	ctx := context.WithValue(req.Context(), clientKey, c)
	req = req.WithContext(ctx)

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err = checkLogin(resp); err != nil {
		return err
	}
	return nil
}

func (user *userT) getLoginData() url.Values {
	return url.Values{
		"username": {user.Username},
		"password": {user.Password},
		"Action":   {"Login"},
		"url":      {BASE_URL + "//f2-MadZone"},
		"Submit":   {"Login"},
	}
}
