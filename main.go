package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

type PodInfo struct {
	Hostname  string `json:"hostname"`
	PodIP     string `json:"pod_ip,omitempty"`
	HostIP    string `json:"host_ip,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type ResourceInfo struct {
	CPURequest string `json:"cpu_request,omitempty"`
	CPULimit   string `json:"cpu_limit,omitempty"`
	MemRequest string `json:"mem_request,omitempty"`
	MemLimit   string `json:"mem_limit,omitempty"`
}

type ConfigMapInfo struct {
	Files map[string]string `json:"files"`
	Count int               `json:"count"`
}

type WhoAmIResponse struct {
	Version     VersionInfo            `json:"version"`
	Pod         PodInfo                `json:"pod"`
	Environment map[string]string      `json:"environment"`
	ConfigMaps  ConfigMapInfo          `json:"configmaps"`
	Resources   ResourceInfo           `json:"resources"`
	Timestamp   string                 `json:"timestamp"`
}

type WhoAmIDetailResponse struct {
	Headers   map[string][]string `json:"headers"`
	ClientIP  string              `json:"client_ip"`
	RemoteIP  string              `json:"remote_ip"`
	PodIP     string              `json:"pod_ip,omitempty"`
	HostIP    string              `json:"host_ip,omitempty"`
	Hostname  string              `json:"hostname"`
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Protocol  string              `json:"protocol"`
	Timestamp string              `json:"timestamp"`
}

func main() {
	port := getEnv("PORT", "8080")

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleWhoAmI)
	mux.HandleFunc("/whoami", handleWhoAmIDetail)
	mux.HandleFunc("/version", handleVersion)
	mux.HandleFunc("/envs", handleEnvs)
	mux.HandleFunc("/cm", handleConfigMaps)
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/readyz", handleReadyz)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	response := WhoAmIResponse{
		Version:     getVersionInfo(),
		Pod:         getPodInfo(),
		Environment: getEnvironment(),
		ConfigMaps:  getConfigMaps(),
		Resources:   getResourceInfo(),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	version := getVersionInfo()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

func handleEnvs(w http.ResponseWriter, r *http.Request) {
	envs := getEnvironment()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(envs)
}

func handleConfigMaps(w http.ResponseWriter, r *http.Request) {
	cms := getConfigMaps()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cms)
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func handleWhoAmIDetail(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()

	// 获取客户端 IP
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = strings.Split(forwarded, ",")[0]
	} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		clientIP = realIP
	}

	response := WhoAmIDetailResponse{
		Headers:   r.Header,
		ClientIP:  strings.TrimSpace(clientIP),
		RemoteIP:  r.RemoteAddr,
		PodIP:     getLocalIP(),
		HostIP:    os.Getenv("HOST_IP"),
		Hostname:  hostname,
		Method:    r.Method,
		Path:      r.URL.Path,
		Protocol:  r.Proto,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func getVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: strings.TrimPrefix(strings.Split(os.Getenv("GOLANG_VERSION"), " ")[0], "go"),
	}
}

func getPodInfo() PodInfo {
	hostname, _ := os.Hostname()
	return PodInfo{
		Hostname:  hostname,
		PodIP:     os.Getenv("POD_IP"),
		HostIP:    os.Getenv("HOST_IP"),
		Namespace: os.Getenv("POD_NAMESPACE"),
	}
}

func getEnvironment() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 && strings.HasPrefix(pair[0], "GAEA") {
			env[pair[0]] = pair[1]
		}
	}
	return env
}

func getConfigMaps() ConfigMapInfo {
	configPath := "/etc/config"
	files := make(map[string]string)

	if _, err := os.Stat(configPath); err == nil {
		filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				content, err := readFileContent(path)
				if err == nil {
					relPath := strings.TrimPrefix(path, configPath+"/")
					files[relPath] = content
				}
			}
			return nil
		})
	}

	return ConfigMapInfo{
		Files: files,
		Count: len(files),
	}
}

func getResourceInfo() ResourceInfo {
	return ResourceInfo{
		CPURequest: os.Getenv("CPU_REQUEST"),
		CPULimit:   os.Getenv("CPU_LIMIT"),
		MemRequest: os.Getenv("MEM_REQUEST"),
		MemLimit:   os.Getenv("MEM_LIMIT"),
	}
}

func readFileContent(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
