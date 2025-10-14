package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	EXIF_ORIENTATION_NORMAL = 1
)

const (
	DEFAULT_MIME_TYPE = "application/octet-stream"
	JPEG_MIME_TYPE    = "image/jpeg"
	WEBP_MIME_TYPE    = "image/webp"
)

const (
	DEFAULT_IMG_MAX_WIDTH         = 1920
	DEFAULT_IMG_MAX_HEIGHT        = 1080
	DEFAULT_IMG_MAX_NARROW_SIDE   = 0
	DEFAULT_JPEG_QUALITY          = 90
	DEFAULT_WEBP_QUALITY          = 90
	DEFAULT_NORMALIZE_EXTENSIONS  = 1
	DEFAULT_CONVERT_TO_FORMAT     = ""
)

const IMG_MAX_WIDTH = "IMG_MAX_WIDTH"
const IMG_MAX_HEIGHT = "IMG_MAX_HEIGHT"
const IMG_MAX_NARROW_SIDE = "IMG_MAX_NARROW_SIDE"
const JPEG_QUALITY = "JPEG_QUALITY"
const WEBP_QUALITY = "WEBP_QUALITY"
const NORMALIZE_EXTENSIONS = "NORMALIZE_EXTENSIONS"


const UPLOAD_MAX_SIZE = "UPLOAD_MAX_SIZE"
const IMG_MAX_PIXELS = "IMG_MAX_PIXELS"


const FORWARD_DESTINATION = "FORWARD_DESTINATION"
const FILE_UPLOAD_FIELD = "FILE_UPLOAD_FIELD"
const LISTEN_PATH = "LISTEN_PATH"
const CONVERT_TO_FORMAT = "CONVERT_TO_FORMAT"


var client *http.Client


/*
Test with
curl --header "X-Test: hello" -F "deviceAssetId=web-input.jpg-1672571948584" -F "deviceId=WEB" -F "createdAt=2016-12-02T10:10:20.000Z" -F "modifiedAt=2023-01-01T11:19:08.584Z" -F "isFavorite=false" -F "duration=0:00:00.000000" -F "fileExtension=.jpg" -F "assetData=@example.jpg" http://localhost:6743/upload
*/
func main() {
	cfg := NewConfigFromEnv()
	
	log.Println(IMG_MAX_WIDTH+": ", cfg.ImgMaxWidth)
	log.Println(IMG_MAX_HEIGHT+": ", cfg.ImgMaxHeight) 
	log.Println(IMG_MAX_NARROW_SIDE+": ", cfg.ImgMaxNarrowSide)
	log.Println(JPEG_QUALITY+": ", cfg.JpegQuality)
	log.Println(WEBP_QUALITY+": ", cfg.WebpQuality)
	if cfg.NormalizeExt {
		log.Println(NORMALIZE_EXTENSIONS+": ", 1)
	} else {
		log.Println(NORMALIZE_EXTENSIONS+": ", 0)
	}
	log.Println(UPLOAD_MAX_SIZE+": ", cfg.UploadMaxSize)
	log.Println(IMG_MAX_PIXELS+": ", cfg.ImgMaxPixels)
	log.Println(FORWARD_DESTINATION+": ", cfg.ForwardDestination)
	log.Println(FILE_UPLOAD_FIELD+": ", cfg.FileUploadField)
	log.Println(LISTEN_PATH+": ", cfg.ListenPath)
	log.Println(CONVERT_TO_FORMAT+": ", cfg.ConvertToFormat)

	client = &http.Client{
		Timeout: time.Second * 60,
	}

	handlerWithConfig := func(w http.ResponseWriter, r *http.Request) {
		proxyHandler(w, r, cfg)
	}

	http.HandleFunc(cfg.ListenPath, handlerWithConfig)
	if cfg.ListenPath != "/" {
		http.HandleFunc("/", handlerWithConfig)
	}
	http.ListenAndServe(":6743", nil)
}

func proxyHandler(w http.ResponseWriter, r *http.Request, cfg *Config) {
	body := &bytes.Buffer{}
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		log.Println("Incoming file upload")

		var err error
		contentType, body, err = reformatMultipart(w, r, cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		byteBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body = bytes.NewBuffer(byteBody)
	}

	// Forward request
	proxyReq, _ := http.NewRequest(r.Method, cfg.ForwardDestination, body)
	copyHeader(proxyReq.Header, r.Header)
	proxyReq.Header.Set("Content-Length", strconv.Itoa(body.Len()))
	proxyReq.Header.Set("Content-Type", contentType)

	if r.URL.Path != cfg.ListenPath {
		log.Println("Request hit proxy but not the intended path, proxying to copied path")
		proxyReq.URL.Path = r.URL.Path
		proxyReq.URL.RawQuery = r.URL.RawQuery
	}

	proxyResp, err := client.Do(proxyReq)
	if err != nil {
		log.Println("ProxyResp Error:", err)
		http.Error(w, err.Error(), http.StatusFailedDependency)
		return
	}

	copyHeader(w.Header(), proxyResp.Header)
	w.WriteHeader(proxyResp.StatusCode)
	io.Copy(w, proxyResp.Body)
}

var ignoreHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
	"Accept-Encoding",
	"Host",
	"Cf-Ipcountry",
	"Cf-Connecting-Ip",
	"X-Forwarded-Proto",
	"X-Forwarded-For",
	"Cf-Ray",
	"Cf-Visitor",
	"Cf-Warp-Tag-Id",
	"Content-Type",
	"Origin",
	"X-Amzn-Trace-Id",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		ignoreThisHeader := false
		for _, ignoreHeader := range ignoreHeaders {
			if strings.ToLower(k) == strings.ToLower(ignoreHeader) {
				ignoreThisHeader = true
				break
			}
		}
		if ignoreThisHeader {
			continue
		}

		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

