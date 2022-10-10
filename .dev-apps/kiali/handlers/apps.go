package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kiali/kiali/business"
)

// appParams holds the path and query parameters for appList and appDetails
//
// swagger:parameters appList AppDetails
type appParams struct {
	baseHealthParams
	// The target workload
	//
	// in: path
	Namespace string `json:"namespace"`
	AppName   string `json:"app"`
	// Optional
	IncludeHealth bool `json:"health"`
}

func (p *appParams) extract(r *http.Request) {
	vars := mux.Vars(r)
	query := r.URL.Query()
	p.baseExtract(r, vars)
	p.Namespace = vars["namespace"]
	p.AppName = vars["app"]
	p.IncludeHealth = query.Get("health") != ""
}

// AppList is the API handler to fetch all the apps to be displayed, related to a single namespace
func AppList(w http.ResponseWriter, r *http.Request) {
	p := appParams{}
	p.extract(r)

	criteria := business.AppCriteria{Namespace: p.Namespace, IncludeIstioResources: true, IncludeHealth: p.IncludeHealth, RateInterval: p.RateInterval, QueryTime: p.QueryTime}

	// Get business layer
	business, err := getBusiness(r)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Apps initialization error: "+err.Error())
		return
	}

	if criteria.IncludeHealth {
		rateInterval, err := adjustRateInterval(r.Context(), business, p.Namespace, p.RateInterval, p.QueryTime)
		if err != nil {
			handleErrorResponse(w, err, "Adjust rate interval error: "+err.Error())
			return
		}
		criteria.RateInterval = rateInterval
	}

	// Fetch and build apps
	appList, err := business.App.GetAppList(r.Context(), criteria)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}

	RespondWithJSON(w, http.StatusOK, appList)
}

// AppDetails is the API handler to fetch all details to be displayed, related to a single app
func AppDetails(w http.ResponseWriter, r *http.Request) {
	p := appParams{}
	p.extract(r)

	criteria := business.AppCriteria{Namespace: p.Namespace, AppName: p.AppName, IncludeIstioResources: true, IncludeHealth: p.IncludeHealth, RateInterval: p.RateInterval, QueryTime: p.QueryTime}

	// Get business layer
	business, err := getBusiness(r)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}

	// Fetch and build app
	appDetails, err := business.App.GetAppDetails(r.Context(), criteria)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}

	RespondWithJSON(w, http.StatusOK, appDetails)
}
