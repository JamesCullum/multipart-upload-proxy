package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/h2non/bimg"
)

// Settings / Environment Variables
const IMG_MAX_WIDTH = "IMG_MAX_WIDTH"
const IMG_MAX_HEIGHT = "IMG_MAX_HEIGHT"

var intKeys = []string{IMG_MAX_WIDTH, IMG_MAX_HEIGHT}
var settingsInt map[string]int

const UPLOAD_MAX_SIZE = "UPLOAD_MAX_SIZE"
const IMG_MAX_PIXELS = "IMG_MAX_PIXELS"

var int64Keys = []string{UPLOAD_MAX_SIZE, IMG_MAX_PIXELS}

var settingsInt64 map[string]int64

const FORWARD_DESTINATION = "FORWARD_DESTINATION"
const FILE_UPLOAD_FIELD = "FILE_UPLOAD_FIELD"
const LISTEN_PATH = "LISTEN_PATH"

var stringKeys = []string{FORWARD_DESTINATION, FILE_UPLOAD_FIELD, LISTEN_PATH}
var settingsString map[string]string

var client *http.Client

/*
Test with
curl --header "X-Test: hello" -F "deviceAssetId=web-input.jpg-1672571948584" -F "deviceId=WEB" -F "createdAt=2016-12-02T10:10:20.000Z" -F "modifiedAt=2023-01-01T11:19:08.584Z" -F "isFavorite=false" -F "duration=0:00:00.000000" -F "fileExtension=.jpg" -F "assetData=@example.jpg" http://localhost:6743/upload
*/
func main() {
	// Integer32
	settingsInt = make(map[string]int)
	var defaultSettingsInt = map[string]int{
		IMG_MAX_WIDTH:  1920,
		IMG_MAX_HEIGHT: 1080,
	}
	var intKeys = []string{IMG_MAX_WIDTH, IMG_MAX_HEIGHT}
	for _, intKey := range intKeys {
		settingsInt[intKey] = defaultSettingsInt[intKey]

		envValue := os.Getenv(intKey)
		if len(envValue) > 0 {
			convEnvValue, err := strconv.Atoi(envValue)
			if err == nil {
				settingsInt[intKey] = convEnvValue
			}
		}

		fmt.Println(intKey+": ", settingsInt[intKey])
	}

	// Integer64
	settingsInt64 = make(map[string]int64)
	var defaultSettingsInt64 = map[string]int64{
		UPLOAD_MAX_SIZE: 100 << 20,
		IMG_MAX_PIXELS:  int64(settingsInt["IMG_MAX_WIDTH"]) * int64(settingsInt["IMG_MAX_HEIGHT"]),
	}
	for _, int64Key := range int64Keys {
		settingsInt64[int64Key] = defaultSettingsInt64[int64Key]

		envValue := os.Getenv(int64Key)
		if len(envValue) > 0 {
			convEnvValue, err := strconv.ParseInt(envValue, 10, 64)
			if err == nil {
				settingsInt64[int64Key] = convEnvValue
			}
		}

		fmt.Println(int64Key+": ", settingsInt64[int64Key])
	}

	// Integer64
	settingsString = make(map[string]string)
	var defaultSettingsString = map[string]string{
		FORWARD_DESTINATION: "https://httpbin.org/post",
		FILE_UPLOAD_FIELD:   "assetData",
		LISTEN_PATH:         "/",
	}
	for _, stringKey := range stringKeys {
		settingsString[stringKey] = defaultSettingsString[stringKey]

		envValue := os.Getenv(stringKey)
		if len(envValue) > 0 {
			settingsString[stringKey] = envValue
		}

		fmt.Println(stringKey+": ", settingsString[stringKey])
	}

	// Other
	client = &http.Client{
		Timeout: time.Second * 10,
	}

	http.HandleFunc("/", proxyHandler)
	http.ListenAndServe(":6743", nil)
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == settingsString[LISTEN_PATH] && strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		// Process request with a multipart/form-data
		uploadFile(w, r)
	} else {
		// Forward the rest of requests
		fmt.Println("Forwarding", r.Method, "request on", r.URL.Path)
		forwardRequest(w, r)
	}
}

func forwardRequest(w http.ResponseWriter, r *http.Request) {
	parsedURL, _ := url.Parse(settingsString[FORWARD_DESTINATION])
	baseURL := &url.URL{ // Get only the base URL to forward the whole request there
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
	}
	proxy := httputil.NewSingleHostReverseProxy(baseURL)

	// Modify the request to set the correct host and scheme
	r.URL.Host = baseURL.Host
	r.URL.Scheme = baseURL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host")) // Need to set it as forwarded host, otherwise the host header will be the proxy's host
	r.Host = baseURL.Host

	// Forward the request
	proxy.ServeHTTP(w, r)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Incoming file upload")

	r.ParseMultipartForm(settingsInt64[UPLOAD_MAX_SIZE])
	file, handler, err := r.FormFile(settingsString[FILE_UPLOAD_FIELD])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read and parse image
	byteContainer, _ := io.ReadAll(file)
	oldImage := bimg.NewImage(byteContainer)
	oldImageSize, err := oldImage.Size()
	if err == nil {
		oldImagePX, oldImageAspect := int64(oldImageSize.Width*oldImageSize.Height), float64(oldImageSize.Width)/float64(oldImageSize.Height)
		if oldImagePX > settingsInt64[IMG_MAX_PIXELS] {
			fmt.Println("Conversion needed")

			var newWidth int
			var newHeight int
			if oldImageAspect >= 1 {
				newWidth = settingsInt[IMG_MAX_WIDTH]
				newHeight = int(float64(settingsInt[IMG_MAX_WIDTH]) * oldImageAspect)
			} else {
				newHeight = settingsInt[IMG_MAX_HEIGHT]
				newWidth = int(float64(settingsInt[IMG_MAX_HEIGHT]) * oldImageAspect)
			}

			newByteContainer, err := oldImage.Resize(newWidth, newHeight)
			if err == nil {
				if len(byteContainer) > len(newByteContainer) {
					fmt.Println("Resizing saved space, so we're taking that")
					byteContainer = newByteContainer
				} else {
					fmt.Println("After conversion, original file is smaller - therefore keeping the original")
				}

			} else if err != nil {
				fmt.Println("Resize Error:", err)
			}
		} else {
			fmt.Println("Conversion not needed")
		}
	} else {
		fmt.Println("Size() Error:", err)
	}

	// Copy form values
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for formKey := range r.Form {
		formValue := r.Form.Get(formKey)
		// fmt.Println(formKey, " => ", formValue) // To much bloat in the logs if we print all the form values for each upload

		fw, _ := writer.CreateFormField(formKey)
		io.Copy(fw, strings.NewReader(formValue))
	}
	// Add new file
	mimeType := handler.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		extension := filepath.Ext(handler.Filename)
		mimeType = mime.TypeByExtension(extension)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	fw, _ := CreateFormFileWithMime(writer, settingsString[FILE_UPLOAD_FIELD], handler.Filename, mimeType)
	io.Copy(fw, bytes.NewReader(byteContainer))
	writer.Close()

	// Forward request
	proxyReq, _ := http.NewRequest(r.Method, settingsString[FORWARD_DESTINATION], body)
	copyHeader(proxyReq.Header, r.Header)
	proxyReq.Header.Set("Content-Length", strconv.Itoa(body.Len()))
	proxyReq.Header.Set("Content-Type", writer.FormDataContentType())

	proxyResp, err := client.Do(proxyReq)
	if err != nil {
		fmt.Println("ProxyResp Error:", err)
		http.Error(w, err.Error(), http.StatusFailedDependency)
		return
	}

	// Send result back
	copyHeader(w.Header(), proxyResp.Header)
	w.WriteHeader(proxyResp.StatusCode)
	io.Copy(w, proxyResp.Body)
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func CreateFormFileWithMime(w *multipart.Writer, fieldname, filename, mimeType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(fieldname), escapeQuotes(filename)))
	h.Set("Content-Type", mimeType)
	return w.CreatePart(h)
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
