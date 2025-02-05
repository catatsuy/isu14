package main

import (
	"context"
	crand "crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-sql-driver/mysql"
	"github.com/goccy/go-json"
	"github.com/jmoiron/sqlx"
	proxy "github.com/shogo82148/go-sql-proxy"
)

var db *sqlx.DB

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	mux := setup()
	go func() {
		ch := time.Tick(500 * time.Millisecond)
		for {
			<-ch
			internalMatching(context.Background())
		}
	}()
	slog.Info("Listening on :8080")
	http.ListenAndServe(":8080", mux)
}

func RegisterTracer() {
	sql.Register("mysql:mytrace", proxy.NewProxyContext(&mysql.MySQLDriver{}, proxy.NewTraceHooks(proxy.TracerOptions{
		Filter: proxy.PackageFilter{
			"database/sql":                       struct{}{},
			"github.com/shogo82148/go-sql-proxy": struct{}{},
			"github.com/jmoiron/sqlx":            struct{}{},
		},
	})))
}

func setup() http.Handler {
	host := os.Getenv("ISUCON_DB_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("ISUCON_DB_PORT")
	if port == "" {
		port = "3306"
	}
	_, err := strconv.Atoi(port)
	if err != nil {
		panic(fmt.Sprintf("failed to convert DB port number from ISUCON_DB_PORT environment variable into int: %v", err))
	}
	user := os.Getenv("ISUCON_DB_USER")
	if user == "" {
		user = "isucon"
	}
	password := os.Getenv("ISUCON_DB_PASSWORD")
	if password == "" {
		password = "isucon"
	}
	dbname := os.Getenv("ISUCON_DB_NAME")
	if dbname == "" {
		dbname = "isuride"
	}

	dbConfig := mysql.NewConfig()
	dbConfig.User = user
	dbConfig.Passwd = password
	dbConfig.Addr = net.JoinHostPort(host, port)
	dbConfig.Net = "tcp"
	dbConfig.DBName = dbname
	dbConfig.ParseTime = true
	dbConfig.InterpolateParams = true

	var isDev bool
	if os.Getenv("DEV") == "1" {
		isDev = true
	}

	driverName := "mysql"
	if isDev {
		RegisterTracer()

		driverName = "mysql:mytrace"
	}

	db, err = sqlx.Connect(driverName, dbConfig.FormatDSN())
	if err != nil {
		panic(err)
	}

	maxConns := os.Getenv("DB_MAXOPENCONNS")
	maxConnsInt := 25
	if maxConns != "" {
		maxConnsInt, err = strconv.Atoi(maxConns)
		if err != nil {
			panic(err)
		}
	}
	db.SetMaxOpenConns(maxConnsInt)
	db.SetMaxIdleConns(maxConnsInt * 2)
	db.SetConnMaxLifetime(3 * time.Minute)
	// db.SetConnMaxIdleTime(2 * time.Minute)

	for {
		err := db.Ping()
		// _, err := db.Exec("SELECT 42")
		if err == nil {
			break
		}
		slog.Info("failed to ping DB", err)
		time.Sleep(time.Second * 2)
	}

	// devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	// if err != nil {
	// 	panic(err)
	// }
	// defer devNull.Close()
	// logger := slog.New(slog.NewTextHandler(devNull, &slog.HandlerOptions{}))
	// slog.SetDefault(logger)

	mux := chi.NewRouter()
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.HandleFunc("POST /api/initialize", postInitialize)

	// app handlers
	{
		mux.HandleFunc("POST /api/app/users", appPostUsers)

		authedMux := mux.With(appAuthMiddleware)
		authedMux.HandleFunc("POST /api/app/payment-methods", appPostPaymentMethods)
		authedMux.HandleFunc("GET /api/app/rides", appGetRides)
		authedMux.HandleFunc("POST /api/app/rides", appPostRides)
		authedMux.HandleFunc("POST /api/app/rides/estimated-fare", appPostRidesEstimatedFare)
		authedMux.HandleFunc("POST /api/app/rides/{ride_id}/evaluation", appPostRideEvaluatation)
		authedMux.HandleFunc("GET /api/app/notification", appGetNotification)
		authedMux.HandleFunc("GET /api/app/nearby-chairs", appGetNearbyChairs)
	}

	// owner handlers
	{
		mux.HandleFunc("POST /api/owner/owners", ownerPostOwners)

		authedMux := mux.With(ownerAuthMiddleware)
		authedMux.HandleFunc("GET /api/owner/sales", ownerGetSales)
		authedMux.HandleFunc("GET /api/owner/chairs", ownerGetChairs)
	}

	// chair handlers
	{
		c := time.Tick(1 * time.Second)
		go func() {
			for {
				cls := mCacheChairLocation.Rotate()
				err := BulkUpdateChairLocations(context.Background(), cls)
				if err != nil {
					log.Printf("[WARN] logger send failed. err:%s", err)
				}
				<-c
			}
		}()

		mux.HandleFunc("POST /api/chair/chairs", chairPostChairs)

		authedMux := mux.With(chairAuthMiddleware)
		authedMux.HandleFunc("POST /api/chair/activity", chairPostActivity)
		authedMux.HandleFunc("POST /api/chair/coordinate", chairPostCoordinate)
		authedMux.HandleFunc("GET /api/chair/notification", chairGetNotification)
		authedMux.HandleFunc("POST /api/chair/rides/{ride_id}/status", chairPostRideStatus)
	}

	// internal handlers
	{
		mux.HandleFunc("GET /api/internal/matching", internalGetMatching)
	}

	return mux
}

type postInitializeRequest struct {
	PaymentServer string `json:"payment_server"`
}

type postInitializeResponse struct {
	Language string `json:"language"`
}

func postInitialize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &postInitializeRequest{}
	if err := bindJSON(r, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if out, err := exec.Command("../sql/init.sh").CombinedOutput(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to initialize: %s: %w", string(out), err))
		return
	}

	if _, err := db.ExecContext(ctx, "UPDATE settings SET value = ? WHERE name = 'payment_gateway_url'", req.PaymentServer); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, postInitializeResponse{Language: "go"})
}

type Coordinate struct {
	Latitude  int `json:"latitude"`
	Longitude int `json:"longitude"`
}

func bindJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	buf, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(statusCode)
	w.Write(buf)
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.WriteHeader(statusCode)
	buf, marshalError := json.Marshal(map[string]string{"message": err.Error()})
	if marshalError != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"marshaling error failed"}`))
		return
	}
	w.Write(buf)

	_, file, line, _ := runtime.Caller(1)

	slog.Error("error response wrote", err, "file", file, "line", line)
}

func secureRandomStr(b int) string {
	k := make([]byte, b)
	if _, err := crand.Read(k); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", k)
}
