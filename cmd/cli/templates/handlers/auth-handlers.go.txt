package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"myapp/data"
	"net/http"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/namnguyen191/goravel/mailer"

	"github.com/namnguyen191/goravel/urlsigner"
)

func (h *Handlers) UserLogin(rw http.ResponseWriter, r *http.Request) {
	err := h.App.Render.Page(rw, r, "login", nil, nil)
	if err != nil {
		h.App.ErrorLog.Println(err)
	}
}

func (h *Handlers) PostUserLogin(rw http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		rw.Write([]byte(err.Error()))
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	user, err := h.Models.Users.GetByEmail(email)
	if err != nil {
		rw.Write([]byte(err.Error()))
		return
	}

	matches, err := user.PasswordMatches(password)
	if err != nil {
		rw.Write([]byte("Error validating password"))
		return
	}

	if !matches {
		rw.Write([]byte("Invalid password"))
		return
	}

	// did the user check "remember me"
	if r.Form.Get("remember") == "remember" {
		randomString := h.randomString(12)
		hasher := sha256.New()
		_, err := hasher.Write([]byte(randomString))
		if err != nil {
			h.App.ErrorStatus(rw, http.StatusBadRequest)
			return
		}

		sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
		rm := data.RememberToken{}
		err = rm.InsertToken(user.ID, sha)
		if err != nil {
			h.App.ErrorStatus(rw, http.StatusBadRequest)
			return
		}

		// set a cookie
		expire := time.Now().Add(365 * 24 * 60 * time.Second)
		cookie := http.Cookie{
			Name:     fmt.Sprintf("_%s_remember", h.App.AppName),
			Value:    fmt.Sprintf("%d|%s", user.ID, sha),
			Path:     "/",
			Expires:  expire,
			HttpOnly: true,
			Domain:   h.App.Session.Cookie.Domain,
			MaxAge:   315350000,
			Secure:   h.App.Session.Cookie.Secure,
			SameSite: http.SameSiteStrictMode,
		}

		http.SetCookie(rw, &cookie)
		// save hash in session
		h.App.Session.Put(r.Context(), "remember_token", sha)
	}

	h.App.Session.Put(r.Context(), "userID", user.ID)

	http.Redirect(rw, r, "/", http.StatusSeeOther)
}

func (h *Handlers) Logout(rw http.ResponseWriter, r *http.Request) {
	// delete the remember token if exist
	if h.App.Session.Exists(r.Context(), "remember_token") {
		rt := data.RememberToken{}
		_ = rt.Delete(h.App.Session.GetString(r.Context(), "remember_token"))
	}

	// delete the cookie
	newCookie := http.Cookie{
		Name:     fmt.Sprintf("_%s_remember", h.App.AppName),
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-100 * time.Hour),
		HttpOnly: true,
		Domain:   h.App.Session.Cookie.Domain,
		MaxAge:   -1,
		Secure:   h.App.Session.Cookie.Secure,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(rw, &newCookie)

	h.App.Session.RenewToken(r.Context())
	h.App.Session.Remove(r.Context(), "userID")
	h.App.Session.Remove(r.Context(), "remember_token")
	h.App.Session.Destroy(r.Context())
	h.App.Session.RenewToken(r.Context())

	http.Redirect(rw, r, "/users/login", http.StatusSeeOther)
}

func (h *Handlers) Forgot(rw http.ResponseWriter, r *http.Request) {
	err := h.render(rw, r, "forgot", nil, nil)

	if err != nil {
		h.App.ErrorLog.Println("error rendering with error: ", err)
		h.App.Error500(rw, r)
	}
}

func (h *Handlers) PostForgot(rw http.ResponseWriter, r *http.Request) {
	// parse form
	err := r.ParseForm()
	if err != nil {
		h.App.ErrorStatus(rw, http.StatusBadRequest)
		return
	}

	// verify that supplied email exists
	var u *data.User
	email := r.Form.Get("email")
	u, err = u.GetByEmail(email)
	if err != nil {
		h.App.ErrorStatus(rw, http.StatusBadRequest)
		return
	}

	// create a link to password reset form
	link := fmt.Sprintf("%s/users/reset-password?email=%s", h.App.Server.URL, email)

	// sign the link
	sign := urlsigner.Signer{
		Secret: []byte(h.App.EncryptionKey),
	}

	signedLink := sign.GenerateTokenFromString(link)

	h.App.InfoLog.Println("Signed Link is", signedLink)

	// email the message
	var data struct {
		Link string
	}

	data.Link = signedLink

	msg := mailer.Message{
		To:       u.Email,
		Subject:  "Password reset",
		Template: "password-reset",
		Data:     data,
		From:     "admin@example.com",
	}

	h.App.Mail.Jobs <- msg
	res := <-h.App.Mail.Results
	if res.Error != nil {
		h.App.ErrorStatus(rw, http.StatusBadRequest)
		return
	}

	// redirect the user
	http.Redirect(rw, r, "/users/login", http.StatusSeeOther)
}

func (h *Handlers) ResetPasswordForm(rw http.ResponseWriter, r *http.Request) {
	// get form value
	email := r.URL.Query().Get("email")
	theURL := r.RequestURI
	testURL := fmt.Sprintf("%s%s", h.App.Server.URL, theURL)

	// validate the url
	signer := urlsigner.Signer{
		Secret: []byte(h.App.EncryptionKey),
	}

	valid := signer.VerifyToken(testURL)
	if !valid {
		h.App.ErrorLog.Println("Invalid url: ", testURL)
		h.App.ErrorUnauthorized(rw, r)
		return
	}

	// make sure it's not expired
	expired := signer.Expired(testURL, 60)
	if expired {
		h.App.ErrorLog.Println("Link expired")
		h.App.ErrorUnauthorized(rw, r)
		return
	}

	// display form
	encryptedEmail, _ := h.encrypt(email)

	vars := make(jet.VarMap)
	vars.Set("email", encryptedEmail)

	err := h.render(rw, r, "reset-password", vars, nil)
	if err != nil {
		h.App.ErrorLog.Println(err)
		return
	}
}

func (h *Handlers) PostResetPassword(rw http.ResponseWriter, r *http.Request) {
	// parse the form
	err := r.ParseForm()
	if err != nil {
		h.App.Error500(rw, r)
		return
	}

	// get and decrypt the email
	email, err := h.decrypt(r.Form.Get("email"))
	if err != nil {
		h.App.Error500(rw, r)
		return
	}

	// get the user
	var u data.User
	user, err := u.GetByEmail(email)
	if err != nil {
		h.App.Error500(rw, r)
		return
	}

	// reset the password
	err = user.ResetPassword(user.ID, r.Form.Get("password"))
	if err != nil {
		h.App.Error500(rw, r)
		return
	}

	// redirect
	h.App.Session.Put(r.Context(), "flash", "Password reset. You can now log in.")
	http.Redirect(rw, r, "/users/login", http.StatusSeeOther)
}
