package handlers

import (
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/product-images/files"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Files is a handler for reading and writing files
type Files struct {
	log   hclog.Logger
	store files.Storage
}

// NewFiles creates a new file handler
func NewFiles(l hclog.Logger, s files.Storage) *Files {
	return &Files{log: l, store: s}
}

// ServeHTTP implements the http.Handler interface
//func (f *Files) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
//	vars := mux.Vars(r)
//	id, idExists := vars["id"]
//	fn, fnExists := vars["filename"]
//
// Validate that id and filename are present
//	if !idExists || !fnExists || id == "" || fn == "" {
//		f.invalidURI(r.URL.Path, rw)
//		return
//	}
//
//	f.log.Info("Handle request", "method", r.Method, "id", id, "filename", fn)
//
// Handle different HTTP methods
//	switch r.Method {
//	case http.MethodGet:
//		f.getFile(id, fn, rw, r)
//	case http.MethodPost:
//		f.saveFile(id, fn, rw, r)
//	default:
//		f.invalidMethod(rw, r)
//	}
//}

func (f *Files) UploadREST(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, idExists := vars["id"]
	fn, fnExists := vars["filename"]

	// Validate that id and filename are present
	if !idExists || !fnExists || id == "" || fn == "" {
		f.invalidURI(r.URL.Path, rw)
		return
	}

	f.log.Info("Handle POST (REST)", "id", id, "filename", fn)

	// Save the file from the request body
	f.saveFile(id, fn, rw, r.Body)
}

func (f *Files) UploadMultipart(rw http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(128 * 1024)
	if err != nil {
		f.log.Error("Unable to parse multipart form", "error", err)
		http.Error(rw, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Retrieve 'id' from form data
	id := r.FormValue("id")
	if id == "" {
		f.log.Error("Missing 'id' in form data")
		http.Error(rw, "Missing 'id' in form data", http.StatusBadRequest)
		return
	}

	// Retrieve the file
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		f.log.Error("Unable to get file from form data", "error", err)
		http.Error(rw, "Unable to get file from form data", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fn := fileHeader.Filename
	f.log.Info("Handle POST (multipart)", "id", id, "filename", fn)

	// Save the file
	f.saveFile(id, fn, rw, file)
}

func (f *Files) GetFile(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	fn := vars["filename"]

	f.log.Info("Handle GET", "id", id, "filename", fn)

	f.getFile(id, fn, rw, r)
}

func (f *Files) invalidURI(uri string, rw http.ResponseWriter) {
	f.log.Error("Invalid path", "path", uri)
	http.Error(rw, "Invalid file path should be in the format: /[id]/[filepath]", http.StatusBadRequest)
}

// saveFile saves the contents of the request to a file
func (f *Files) saveFile(id, path string, rw http.ResponseWriter, r io.Reader) {
	f.log.Info("Save file for products", "id", id, "path", path)

	fp := filepath.Join(id, path)
	err := f.store.Save(fp, r)
	if err != nil {
		f.log.Error("Unable to save the file", "error", err)
		http.Error(rw, "Unable to save the file", http.StatusInternalServerError)
		return
	}

	// Respond with success
	rw.WriteHeader(http.StatusCreated)
	rw.Write([]byte("File uploaded successfully"))
}

func (f *Files) getFile(id, path string, rw http.ResponseWriter, r *http.Request) {
	f.log.Info("Get file for product", "id", id, "path", path)

	// Construct the filepath
	fp := filepath.Join(id, path)

	// Use the storage interface to get the file
	file, err := f.store.Get(fp)
	if err != nil {
		f.log.Error("Unable to get the file", "error", err)
		http.Error(rw, "File not found", http.StatusNotFound)
		return
	}

	defer file.Close()

	// Determine the content type
	contentType, err := getContentType(file)
	if err != nil {
		f.log.Error("Unable to detect content type", "error", err)
		contentType = "application/octet-stream"
	}
	rw.Header().Set("Content-Type", contentType)

	// Write the file content to the response
	_, err = io.Copy(rw, file)
	if err != nil {
		f.log.Error("Unable to write file to response", "error", err)
		http.Error(rw, "Unable to serve the file", http.StatusInternalServerError)
	}
}

// getContentType determines the MIME type of the file based on its content
func getContentType(file *os.File) (string, error) {
	// Read a portion of the file to detect the content type
	buf := make([]byte, 512) // 512 bytes is sufficient
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Reset the file pointer to the beginning
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buf[:n])
	return contentType, nil
}

//func (f *Files) invalidMethod(rw http.ResponseWriter, r *http.Request) {
//	f.log.Error("Invalid method", "method", r.Method)
//	http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
//}
