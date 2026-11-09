package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"atonixcorp/cloud/hpc/database/model"
	"atonixcorp/cloud/hpc/database/service"
)

var (
	db          *service.DatabaseService
	datasetSvc  *service.DatasetService
	runtime     *service.RuntimeIntegration
	storageBind *service.StorageBinding
)

func init() {
	fmt.Println("Initializing AtonixCorp HPC Database Service...")
	db = service.NewDatabaseService()
	datasetSvc = service.NewDatasetService(db)
	runtime = service.NewRuntimeIntegration(db)
	storageBind = service.NewStorageBinding()
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		runCLI()
		return
	}

	http.HandleFunc("/datasets/register", registerDataset)
	http.HandleFunc("/datasets/list", listDatasets)
	http.HandleFunc("/mount/slurm", mountSlurm)
	http.HandleFunc("/mount/k8s", mountK8s)
	http.HandleFunc("/storage/bind", bindStorage)

	fmt.Println("HPCDB API Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func registerDataset(w http.ResponseWriter, r *http.Request) {
	var ds model.Dataset
	json.NewDecoder(r.Body).Decode(&ds)
	datasetSvc.Create(&ds)
	w.Write([]byte("Dataset registered\n"))
}

func listDatasets(w http.ResponseWriter, r *http.Request) {
	datasets := db.Registry.List()
	json.NewEncoder(w).Encode(datasets)
}

func mountSlurm(w http.ResponseWriter, r *http.Request) {
	datasetID := r.URL.Query().Get("dataset")
	jobID := r.URL.Query().Get("job")
	mount := runtime.InjectForSlurm(datasetID, jobID)
	w.Write([]byte(mount))
}

func mountK8s(w http.ResponseWriter, r *http.Request) {
	datasetID := r.URL.Query().Get("dataset")
	pod := r.URL.Query().Get("pod")
	mount := runtime.InjectForK8s(datasetID, pod)
	w.Write([]byte(mount))
}

func bindStorage(w http.ResponseWriter, r *http.Request) {
	var backend model.StorageBackend
	json.NewDecoder(r.Body).Decode(&backend)
	storageBind.Bind(backend.ID, &backend)
	w.Write([]byte("Storage backend bound\n"))
}
