package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	bolt "go.etcd.io/bbolt"
)

type Resource struct {
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Spec      map[string]any    `json:"spec"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type Server struct {
	db     *bolt.DB
	logger *slog.Logger
}

func New(dbPath string, logger *slog.Logger) (*Server, error) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return &Server{db: db, logger: logger}, nil
}

func (s *Server) Close() { s.db.Close() }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	mux.HandleFunc("/api/v1/alerts", s.handleAlerts)
	mux.HandleFunc("/api/v1/resources", s.handleResources)
	mux.HandleFunc("/api/v1/resources/", s.handleResource)
	return mux
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Alerts []struct {
			Status      string            `json:"status"`
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
		} `json:"alerts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, a := range payload.Alerts {
		name := a.Labels["alertname"]
		res := Resource{Kind: "Alert", Name: name, Spec: map[string]any{"status": a.Status, "labels": a.Labels, "summary": a.Annotations["summary"]}}
		data, _ := json.Marshal(res)
		s.db.Update(func(tx *bolt.Tx) error {
			b, _ := tx.CreateBucketIfNotExists([]byte("Alert"))
			return b.Put([]byte(name), data)
		})
		s.logger.Info("alert received", "name", name, "status", a.Status)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	switch r.Method {
	case "GET":
		var result []Resource
		s.db.View(func(tx *bolt.Tx) error {
			if kind != "" {
				b := tx.Bucket([]byte(kind))
				if b == nil {
					return nil
				}
				return b.ForEach(func(k, v []byte) error {
					var res Resource
					json.Unmarshal(v, &res)
					result = append(result, res)
					return nil
				})
			}
			return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
				return b.ForEach(func(k, v []byte) error {
					var res Resource
					json.Unmarshal(v, &res)
					result = append(result, res)
					return nil
				})
			})
		})
		json.NewEncoder(w).Encode(result)

	case "POST":
		var res Resource
		if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data, _ := json.Marshal(res)
		s.db.Update(func(tx *bolt.Tx) error {
			b, _ := tx.CreateBucketIfNotExists([]byte(res.Kind))
			return b.Put([]byte(res.Name), data)
		})
		s.logger.Info("resource created", "kind", res.Kind, "name", res.Name)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(res)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleResource(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/v1/resources/"):]
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "need /kind/name", http.StatusBadRequest)
		return
	}
	kind, name := parts[0], parts[1]

	switch r.Method {
	case "GET":
		var res Resource
		var found bool
		s.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(kind))
			if b == nil {
				return nil
			}
			v := b.Get([]byte(name))
			if v != nil {
				json.Unmarshal(v, &res)
				found = true
			}
			return nil
		})
		if !found {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "DELETE":
		s.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(kind))
			if b == nil {
				return nil
			}
			return b.Delete([]byte(name))
		})
		s.logger.Info("resource deleted", "kind", kind, "name", name)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
