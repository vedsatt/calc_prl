package orchestrator

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"sync"

	"github.com/vedsatt/calc_prl/pkg/database"
)

const port = ":8080"

type (
	Orchestrator struct {
	}

	Request struct {
		Expression string `json:"expression"`
	}

	RespID struct {
		Id int `json:"id"`
	}

	Error struct {
		Res string `json:"error"`
	}

	ExprReq struct {
		exp string
		id  int
	}

	contextKey string
)

func New() *Orchestrator {
	return &Orchestrator{}
}

var (
	base   = database.New()
	mu     sync.Mutex // Мьютекс для синхронизации доступа к результатам
	ctxKey contextKey = "expression id"
)

func errorResponse(w http.ResponseWriter, err string, statusCode int) {
	w.WriteHeader(statusCode)
	e := Error{Res: err}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(e)
}

func checkId(id string) bool {
	pattern := "^[0-9]+$"
	r := regexp.MustCompile(pattern)
	return r.MatchString(id)
}

func (o *Orchestrator) Run() {
	StartManager()
	// запуск сервера для общения с агентом
	go runGRPC()

	mux := http.NewServeMux()

	expr := http.HandlerFunc(ExpressionHandler)
	GetData := http.HandlerFunc(GetDataHandler)

	// хендлеры
	mux.Handle("/api/v1/calculate", logsMiddleware(databaseMiddleware(expr)))
	mux.Handle("/api/v1/expressions/", logsMiddleware(GetData))

	log.Printf("Starting sevrer on port %s", port)
	log.Fatal(http.ListenAndServe(port, mux))

}
