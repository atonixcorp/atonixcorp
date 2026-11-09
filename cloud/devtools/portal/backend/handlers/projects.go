package handlers

import (
	"net/http"
)

func ListProjects(w http.ResponseWriter, r *http.Request) {
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	projects, err := client.ListProjects(r.Context())
	writeResult(w, projects, err)
}
