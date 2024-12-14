package main

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/oklog/ulid/v2"
)

const (
	initialFare     = 500
	farePerDistance = 100
)

type ownerPostOwnersRequest struct {
	Name string `json:"name"`
}

type ownerPostOwnersResponse struct {
	ID                 string `json:"id"`
	ChairRegisterToken string `json:"chair_register_token"`
}

func ownerPostOwners(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &ownerPostOwnersRequest{}
	if err := bindJSON(r, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("some of required fields(name) are empty"))
		return
	}

	ownerID := ulid.Make().String()
	accessToken := secureRandomStr(32)
	chairRegisterToken := secureRandomStr(32)

	_, err := db.ExecContext(
		ctx,
		"INSERT INTO owners (id, name, access_token, chair_register_token) VALUES (?, ?, ?, ?)",
		ownerID, req.Name, accessToken, chairRegisterToken,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Path:  "/",
		Name:  "owner_session",
		Value: accessToken,
	})

	writeJSON(w, http.StatusCreated, &ownerPostOwnersResponse{
		ID:                 ownerID,
		ChairRegisterToken: chairRegisterToken,
	})
}

type chairSales struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Sales int    `json:"sales"`
}

type modelSales struct {
	Model string `json:"model"`
	Sales int    `json:"sales"`
}

type ownerGetSalesResponse struct {
	TotalSales int          `json:"total_sales"`
	Chairs     []chairSales `json:"chairs"`
	Models     []modelSales `json:"models"`
}

type OwnerRide struct {
	ChairID              string         `db:"chair_id"`
	ChairModel           string         `db:"chair_model"`
	ChairName            string         `db:"chair_name"`
	Status               sql.NullString `db:"status"`
	PickupLatitude       sql.NullInt32  `db:"pickup_latitude"`
	PickupLongitude      sql.NullInt32  `db:"pickup_longitude"`
	DestinationLatitude  sql.NullInt32  `db:"destination_latitude"`
	DestinationLongitude sql.NullInt32  `db:"destination_longitude"`
}

func ownerGetSales(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	since := time.Unix(0, 0)
	until := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if r.URL.Query().Get("since") != "" {
		parsed, err := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		since = time.UnixMilli(parsed)
	}
	if r.URL.Query().Get("until") != "" {
		parsed, err := strconv.ParseInt(r.URL.Query().Get("until"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		until = time.UnixMilli(parsed)
	}

	owner := r.Context().Value("owner").(*Owner)

	chairs := make([]string, 0, 100)
	if err := db.SelectContext(ctx, &chairs, "SELECT id FROM chairs WHERE owner_id = ?", owner.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	res := ownerGetSalesResponse{
		TotalSales: 0,
	}

	modelSalesByModel := map[string]int{}

	ownerRides := make([]OwnerRide, 0, 100)

	query := `SELECT rides.pickup_latitude AS pickup_latitude, rides.pickup_longitude AS pickup_longitude, rides.destination_latitude AS destination_latitude, rides.destination_longitude AS destination_longitude, chairs.id AS chair_id, chairs.model AS chair_model, chairs.name AS chair_name, ride_statuses.status AS status
	FROM chairs LEFT JOIN rides ON chairs.id = rides.chair_id AND (rides.updated_at BETWEEN ? AND ? + INTERVAL 999 MICROSECOND)
	LEFT JOIN ride_statuses ON rides.id = ride_statuses.ride_id AND status = 'COMPLETED'
	WHERE chairs.id IN (?)`

	query, args, err := sqlx.In(query, since, until, chairs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	err = db.SelectContext(ctx, &ownerRides, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	rideModels := make(map[string]chairSales)

	for _, ride := range ownerRides {
		sales := 0
		if ride.Status.Valid {
			sales = calculateSale(Ride{
				PickupLatitude:       int(ride.PickupLatitude.Int32),
				PickupLongitude:      int(ride.PickupLongitude.Int32),
				DestinationLatitude:  int(ride.DestinationLatitude.Int32),
				DestinationLongitude: int(ride.DestinationLongitude.Int32),
			})
		}
		res.TotalSales += sales

		r, ok := rideModels[ride.ChairID]
		if !ok {
			r = chairSales{
				ID:    ride.ChairID,
				Name:  ride.ChairName,
				Sales: sales,
			}
		} else {
			r.Sales += sales
		}
		rideModels[ride.ChairID] = r

		modelSalesByModel[ride.ChairModel] += sales
	}

	for _, ride := range rideModels {
		res.Chairs = append(res.Chairs, ride)
	}

	sort.Slice(res.Chairs, func(i, j int) bool {
		return res.Chairs[i].ID < res.Chairs[j].ID
	})

	models := []modelSales{}
	for model, sales := range modelSalesByModel {
		models = append(models, modelSales{
			Model: model,
			Sales: sales,
		})
	}
	res.Models = models

	writeJSON(w, http.StatusOK, res)
}

func sumSales(rides []Ride) int {
	sale := 0
	for _, ride := range rides {
		sale += calculateSale(ride)
	}
	return sale
}

func calculateSale(ride Ride) int {
	return calculateFare(ride.PickupLatitude, ride.PickupLongitude, ride.DestinationLatitude, ride.DestinationLongitude)
}

type chairWithDetail struct {
	ID                     string       `db:"id"`
	OwnerID                string       `db:"owner_id"`
	Name                   string       `db:"name"`
	AccessToken            string       `db:"access_token"`
	Model                  string       `db:"model"`
	IsActive               bool         `db:"is_active"`
	CreatedAt              time.Time    `db:"created_at"`
	UpdatedAt              time.Time    `db:"updated_at"`
	TotalDistance          int          `db:"total_distance"`
	TotalDistanceUpdatedAt sql.NullTime `db:"total_distance_updated_at"`
}

type ownerGetChairResponse struct {
	Chairs []ownerGetChairResponseChair `json:"chairs"`
}

type ownerGetChairResponseChair struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Model                  string `json:"model"`
	Active                 bool   `json:"active"`
	RegisteredAt           int64  `json:"registered_at"`
	TotalDistance          int    `json:"total_distance"`
	TotalDistanceUpdatedAt *int64 `json:"total_distance_updated_at,omitempty"`
}

func ownerGetChairs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	owner := ctx.Value("owner").(*Owner)

	chairs := []chairWithDetail{}
	if err := db.SelectContext(ctx, &chairs, `SELECT id,
       owner_id,
       name,
       access_token,
       model,
       is_active,
       created_at,
       updated_at,
       IFNULL(total_distance, 0) AS total_distance,
       total_distance_updated_at
FROM chairs
       LEFT JOIN distance_table ON distance_table.chair_id = chairs.id
WHERE owner_id = ?
`, owner.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	res := ownerGetChairResponse{}
	for _, chair := range chairs {
		c := ownerGetChairResponseChair{
			ID:            chair.ID,
			Name:          chair.Name,
			Model:         chair.Model,
			Active:        chair.IsActive,
			RegisteredAt:  chair.CreatedAt.UnixMilli(),
			TotalDistance: chair.TotalDistance,
		}
		if chair.TotalDistanceUpdatedAt.Valid {
			t := chair.TotalDistanceUpdatedAt.Time.UnixMilli()
			c.TotalDistanceUpdatedAt = &t
		}
		res.Chairs = append(res.Chairs, c)
	}
	writeJSON(w, http.StatusOK, res)
}
