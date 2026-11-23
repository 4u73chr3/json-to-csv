package handlers

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"json-to-csv/services"
	"json-to-csv/templates"
)

const csrfCookieName = "csrf_token"

type Fetcher interface {
	Fetch(context.Context, string) ([]byte, error)
}

type Handler struct {
	api   Fetcher
	index *template.Template
}

func New(api Fetcher) (*Handler, error) {
	page, err := template.New("index.html").Parse(templates.IndexHTML)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &Handler{api: api, index: page}, nil
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	h.renderIndex(w, r, http.StatusOK, "")
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	if !isSameOrigin(r) {
		h.renderIndex(w, r, http.StatusForbidden, "This form submission was rejected.")
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/x-www-form-urlencoded" {
		h.renderIndex(w, r, http.StatusUnsupportedMediaType, "Submit the form using the page controls.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		h.renderIndex(w, r, http.StatusBadRequest, "Unable to read the submitted form.")
		return
	}
	if !validCSRFToken(r) {
		h.renderIndex(w, r, http.StatusForbidden, "Your session expired. Reload the page and try again.")
		return
	}
	parameter := strings.TrimSpace(r.FormValue("parameter"))
	if parameter == "" {
		h.renderIndex(w, r, http.StatusUnprocessableEntity, "Enter an API parameter.")
		return
	}
	if utf8.RuneCountInString(parameter) > 200 {
		h.renderIndex(w, r, http.StatusUnprocessableEntity, "The API parameter must be 200 characters or fewer.")
		return
	}

	payload, err := h.api.Fetch(r.Context(), parameter)
	if err != nil {
		status := http.StatusBadGateway
		message := "The API request failed. Please try again."
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
			message = "The API request timed out. Please try again."
		}
		h.renderIndex(w, r, status, message)
		return
	}

	csvData, err := services.JSONToCSV(payload)
	if err != nil {
		status := http.StatusBadGateway
		message := "The API returned invalid JSON."
		if errors.Is(err, services.ErrNoResults) {
			status = http.StatusUnprocessableEntity
			message = "The API returned no results."
		}
		h.renderIndex(w, r, status, message)
		return
	}

	filename := fmt.Sprintf("export-%s.csv", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvData)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) renderIndex(w http.ResponseWriter, r *http.Request, status int, message string) {
	csrfToken := ensureCSRFToken(w, r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	data := struct {
		Error     string
		CSRFToken string
	}{
		Error:     message,
		CSRFToken: csrfToken,
	}
	if err := h.index.ExecuteTemplate(w, "index.html", data); err != nil {
		return
	}
}

func ensureCSRFToken(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(csrfCookieName); err == nil && isCSRFToken(cookie.Value) {
		return cookie.Value
	}

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("generate CSRF token: " + err.Error())
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   60 * 60,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})
	return token
}

func validCSRFToken(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return false
	}
	formToken := r.FormValue("csrf_token")
	return isCSRFToken(cookie.Value) &&
		isCSRFToken(formToken) &&
		subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formToken)) == 1
}

func isCSRFToken(token string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(decoded) == 32
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func isSameOrigin(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	requestScheme := r.Header.Get("X-Forwarded-Proto")
	if requestScheme == "" {
		requestScheme = "http"
		if r.TLS != nil {
			requestScheme = "https"
		}
	}
	return parsed.Scheme == requestScheme && parsed.Host == r.Host
}
