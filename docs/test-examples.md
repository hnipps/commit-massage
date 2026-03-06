# LLM Test Examples

Example diffs to send to the LLM for testing commit message generation. Each example covers a different commit type and varies in complexity.

## Example 1 — Dependency update (chore)

```
Files changed:
 go.mod | 2 +-
 go.sum | 4 ++--
 2 files changed, 3 insertions(+), 3 deletions(-)

Diff:
diff --git a/go.mod b/go.mod
index <HASH>..<HASH> 100644
--- a/go.mod
+++ b/go.mod
@@ -3,7 +3,7 @@ module github.com/example/myapp
 go 1.22

 require (
-	github.com/go-chi/chi/v5 v5.0.11
+	github.com/go-chi/chi/v5 v5.1.0
 	github.com/jmoiron/sqlx v1.3.5
 )
diff --git a/go.sum b/go.sum
index <HASH>..<HASH> 100644
--- a/go.sum
+++ b/go.sum
@@ -1,5 +1,5 @@
-github.com/go-chi/chi/v5 v5.0.11 h1:abc123
-github.com/go-chi/chi/v5 v5.0.11/go.mod h1:def456
+github.com/go-chi/chi/v5 v5.1.0 h1:xyz789
+github.com/go-chi/chi/v5 v5.1.0/go.mod h1:ghi012
 github.com/jmoiron/sqlx v1.3.5 h1:aaa111
 github.com/jmoiron/sqlx v1.3.5/go.mod h1:bbb222
```

Expected: something like `chore: bump go-chi/chi to v5.1.0`

## Example 2 — Performance improvement (perf)

```
Files changed:
 internal/search/index.go | 18 +++++++++---------
 1 file changed, 9 insertions(+), 9 deletions(-)

Diff:
diff --git a/internal/search/index.go b/internal/search/index.go
index <HASH>..<HASH> 100644
--- a/internal/search/index.go
+++ b/internal/search/index.go
@@ -15,15 +15,15 @@ func BuildIndex(docs []Document) map[string][]int {
 	index := make(map[string][]int)
 	for i, doc := range docs {
-		words := strings.Split(doc.Body, " ")
-		for _, word := range words {
-			word = strings.ToLower(word)
-			word = strings.TrimSpace(word)
-			if word == "" {
-				continue
-			}
-			index[word] = append(index[word], i)
-		}
+		words := strings.Fields(doc.Body)
+		for _, word := range words {
+			lower := strings.ToLower(word)
+			if existing, ok := index[lower]; ok {
+				index[lower] = append(existing, i)
+			} else {
+				index[lower] = []int{i}
+			}
+		}
 	}
 	return index
 }
```

Expected: something like `perf(search): use strings.Fields and reduce allocations in index build`

## Example 3 — CI pipeline addition (ci)

```
Files changed:
 .github/workflows/lint.yml | 32 ++++++++++++++++++++++++++++++++
 1 file changed, 32 insertions(+), 0 deletions(-)

Diff:
diff --git a/.github/workflows/lint.yml b/.github/workflows/lint.yml
new file mode 100644
index 0000000..<HASH>
--- /dev/null
+++ b/.github/workflows/lint.yml
@@ -0,0 +1,32 @@
+name: Lint
+
+on:
+  pull_request:
+    branches: [main]
+
+jobs:
+  golangci:
+    name: golangci-lint
+    runs-on: ubuntu-latest
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-go@v5
+        with:
+          go-version: '1.22'
+      - name: golangci-lint
+        uses: golangci/golangci-lint-action@v4
+        with:
+          version: latest
+          args: --timeout=5m
+
+  vet:
+    name: go vet
+    runs-on: ubuntu-latest
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-go@v5
+        with:
+          go-version: '1.22'
+      - run: go vet ./...
```

Expected: something like `ci: add lint workflow for PRs`

## Example 4 — Bug fix with multi-file changes (fix)

```
Files changed:
 internal/auth/session.go | 5 +++--
 internal/auth/middleware.go | 3 ++-
 2 files changed, 5 insertions(+), 3 deletions(-)

Diff:
diff --git a/internal/auth/session.go b/internal/auth/session.go
index <HASH>..<HASH> 100644
--- a/internal/auth/session.go
+++ b/internal/auth/session.go
@@ -41,8 +41,9 @@ func (s *Store) Get(token string) (*Session, error) {
 	s.mu.RLock()
 	defer s.mu.RUnlock()
 	sess, ok := s.sessions[token]
-	if !ok {
-		return nil, ErrSessionNotFound
+	if !ok || sess.ExpiresAt.Before(time.Now()) {
+		delete(s.sessions, token)
+		return nil, ErrSessionExpired
 	}
 	return sess, nil
 }
diff --git a/internal/auth/middleware.go b/internal/auth/middleware.go
index <HASH>..<HASH> 100644
--- a/internal/auth/middleware.go
+++ b/internal/auth/middleware.go
@@ -22,7 +22,8 @@ func RequireAuth(store *Store) func(http.Handler) http.Handler {
 			sess, err := store.Get(token)
 			if err != nil {
-				http.Error(w, "unauthorized", http.StatusUnauthorized)
+				http.Error(w, "session expired", http.StatusUnauthorized)
+				http.SetCookie(w, &http.Cookie{Name: "session", MaxAge: -1})
 				return
 			}
```

Expected: something like `fix(auth): expire stale sessions and clear cookie on expiry`

## Example 5 — Refactor extracting a function (refactor)

```
Files changed:
 cmd/server/main.go | 25 +++++++++++--------------
 1 file changed, 11 insertions(+), 14 deletions(-)

Diff:
diff --git a/cmd/server/main.go b/cmd/server/main.go
index <HASH>..<HASH> 100644
--- a/cmd/server/main.go
+++ b/cmd/server/main.go
@@ -18,20 +18,7 @@ func main() {
 	mux := http.NewServeMux()
 	mux.HandleFunc("/health", healthHandler)

-	port := os.Getenv("PORT")
-	if port == "" {
-		port = "8080"
-	}
-	host := os.Getenv("HOST")
-	if host == "" {
-		host = "0.0.0.0"
-	}
-	addr := host + ":" + port
-
-	srv := &http.Server{
-		Addr:    addr,
-		Handler: mux,
-	}
+	srv := newServer(mux)

 	log.Printf("listening on %s", srv.Addr)
 	if err := srv.ListenAndServe(); err != nil {
@@ -39,6 +26,17 @@ func main() {
 	}
 }

+func newServer(handler http.Handler) *http.Server {
+	port := os.Getenv("PORT")
+	if port == "" {
+		port = "8080"
+	}
+	host := os.Getenv("HOST")
+	if host == "" {
+		host = "0.0.0.0"
+	}
+	return &http.Server{Addr: host + ":" + port, Handler: handler}
+}
+
 func healthHandler(w http.ResponseWriter, r *http.Request) {
 	w.WriteHeader(http.StatusOK)
 }
```

Expected: something like `refactor: extract server configuration into newServer function`
