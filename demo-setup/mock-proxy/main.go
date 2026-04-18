package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

type User struct {
	Label      string
	Sub        string
	Email      string
	Name       string
	Groups     string
	BadgeColor string
}

var users = []User{
	{
		Label:      "Admin + PII",
		Sub:        "user-admin-001",
		Email:      "anna.admin@example.com",
		Name:       "Anna Lindqvist",
		Groups:     "qe-admins,pii-approved,qe-adhoc",
		BadgeColor: "#dc2626",
	},
	{
		Label:      "Editor",
		Sub:        "user-editor-001",
		Email:      "bo.editor@example.com",
		Name:       "Bo Eriksson",
		Groups:     "qe-editors",
		BadgeColor: "#2563eb",
	},
	{
		Label:      "Viewer (no PII)",
		Sub:        "user-viewer-001",
		Email:      "carin.viewer@example.com",
		Name:       "Carin Viewer",
		Groups:     "qe-viewers",
		BadgeColor: "#16a34a",
	},
}

func currentUser(r *http.Request) (int, User) {
	if c, err := r.Cookie("demo_user"); err == nil {
		for i, u := range users {
			if u.Sub == c.Value {
				return i, u
			}
		}
	}
	return 0, users[0]
}

func main() {
	upstream := os.Getenv("UPSTREAM")
	if upstream == "" {
		upstream = "http://app:8080"
	}
	secret := os.Getenv("PROXY_SECRET")
	if secret == "" {
		secret = "dev-proxy-secret-change-me"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	target, err := url.Parse(upstream)
	if err != nil {
		log.Fatalf("invalid upstream: %v", err)
	}

	mux := http.NewServeMux()

	// User switcher endpoint.
	mux.HandleFunc("/__demo/switch", func(w http.ResponseWriter, r *http.Request) {
		sub := r.URL.Query().Get("user")
		rd := r.URL.Query().Get("rd")
		if rd == "" {
			rd = "/"
		}
		http.SetCookie(w, &http.Cookie{
			Name:  "demo_user",
			Value: sub,
			Path:  "/",
		})
		http.Redirect(w, r, rd, http.StatusFound)
	})

	// Proxy everything else.
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(resp *http.Response) error {
		log.Println("proxy.ModifyResponse")
    // Only inject into non-API HTML responses.
    if strings.HasPrefix(resp.Request.URL.Path, "/api/") {
				log.Println("got api request returning null")
        return nil
    }
    ct := resp.Header.Get("Content-Type")
    if !strings.Contains(ct, "text/html") {
        return nil
    }
		// Read body.
		var bodyReader io.Reader = resp.Body
		if resp.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			defer gr.Close()
			bodyReader = gr
			resp.Header.Del("Content-Encoding")
		}
		body, err := io.ReadAll(bodyReader)
		if err != nil {
			return err
		}
		resp.Body.Close()

		// Inject banner before </body>.
		_, activeUser := currentUser(resp.Request)
		banner := buildBanner(activeUser, resp.Request.URL.Path)
		injected := bytes.Replace(body, []byte("</body>"), []byte(banner+"</body>"), 1)

		resp.Body = io.NopCloser(bytes.NewReader(injected))
		resp.ContentLength = int64(len(injected))
		resp.Header.Del("Content-Length")
		return nil
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, u := currentUser(r)

		// Strip inbound identity headers.
		r.Header.Del("X-Auth-Proxy-Secret")
		r.Header.Del("X-Auth-Request-User")
		r.Header.Del("X-Auth-Request-Email")
		r.Header.Del("X-Auth-Request-Preferred-Username")
		r.Header.Del("X-Auth-Request-Groups")

		// Inject identity.
		r.Header.Set("X-Auth-Proxy-Secret", secret)
		r.Header.Set("X-Auth-Request-User", u.Sub)
		r.Header.Set("X-Auth-Request-Email", u.Email)
		r.Header.Set("X-Auth-Request-Preferred-Username", u.Name)
		r.Header.Set("X-Auth-Request-Groups", u.Groups)

		proxy.ServeHTTP(w, r)
	})

	log.Printf("demo proxy on :%s → %s", port, upstream)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func buildBanner(active User, currentPath string) string {
	var btns strings.Builder
	for _, u := range users {
		isActive := u.Sub == active.Sub
		style := fmt.Sprintf("background:%s;color:#fff;", u.BadgeColor)
		if !isActive {
			style = fmt.Sprintf("background:#f3f4f6;color:%s;border:1px solid %s;", u.BadgeColor, u.BadgeColor)
		}
		btns.WriteString(fmt.Sprintf(
			`<a href="/__demo/switch?user=%s&rd=%s" style="%s padding:5px 14px;border-radius:20px;font-size:12px;font-weight:600;text-decoration:none;cursor:pointer;">%s</a>`,
			u.Sub, currentPath, style, u.Label,
		))
	}
	return fmt.Sprintf(`
<div style="position:fixed;bottom:0;left:0;right:0;z-index:99999;background:#1e1e2e;border-top:2px solid #313244;padding:8px 16px;display:flex;align-items:center;gap:10px;font-family:system-ui,sans-serif;">
  <span style="font-size:11px;color:#6c7086;margin-right:4px;">👤 Demo user:</span>
  %s
  <span style="flex:1"></span>
  <span style="font-size:11px;color:#45475a;">QueryExplorer Demo</span>
</div>
<div style="height:44px"></div>`, btns.String())
}