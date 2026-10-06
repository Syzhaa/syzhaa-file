package main

import (
	"github.com/syzhaa/file-server/internal/middleware"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func setupRoutes() *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.CorsMiddleware)
	r.Use(middleware.SecurityHeadersMiddleware)
	// r.Use(csrfMiddleware) // Temporarily disabled - TODO: Implement proper CSRF token flow in frontend
	r.Use(middleware.RateLimitMiddleware(middleware.GeneralLimiter))

	// Public routes (existing)
	r.HandleFunc("/api/stats", getPublicStatsHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/room/create", createRoomHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/room/pin", accessRoomByPinHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/room/{id}", getRoomInfoHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/room/{id}", handleDeleteRoom).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/room/{id}/settings", updateRoomSettingsHandler).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/upload/{roomId}", uploadChunkHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/file/{id}", deleteFileHandler).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/folder/create/{roomId}", createFolderHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/folders/{roomId}", listFoldersHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/folder/{id}", deleteFolderHandler).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/d/{id}", downloadFileHandler).Methods("GET")

	// Admin email/password auth routes
	r.HandleFunc("/auth/login", handleAdminLogin).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/logout", handleAdminLogout).Methods("POST", "OPTIONS")

	// Admin routes (require session)
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(requireAdminSession)
	adminRouter.HandleFunc("/me", handleAdminMe).Methods("GET")
	adminRouter.HandleFunc("/stats", handleAdminStats).Methods("GET")
	adminRouter.HandleFunc("/api-keys", handleListAPIKeys).Methods("GET")
	adminRouter.HandleFunc("/api-keys", handleCreateAPIKey).Methods("POST")
	adminRouter.HandleFunc("/api-keys/{id}", handleDeleteAPIKey).Methods("DELETE")
	adminRouter.HandleFunc("/api-keys/{id}/toggle", handleToggleAPIKey).Methods("POST")
	
	// Admin user management routes
	adminRouter.HandleFunc("/users", handleAdminListUsers).Methods("GET")
	adminRouter.HandleFunc("/rooms", handleAdminListRooms).Methods("GET")
	adminRouter.HandleFunc("/users/{id}/approve", handleAdminApproveUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}/reject", handleAdminRejectUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}/suspend", handleAdminSuspendUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}/quotas", handleAdminUpdateUserQuotas).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}/api-approve", handleAdminApproveUserAPI).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}/api-revoke", handleAdminRevokeUserAPI).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}/stats", handleAdminGetUserStats).Methods("GET")
	adminRouter.HandleFunc("/settings", handleAdminSystemSettings).Methods("GET", "POST")
	adminRouter.HandleFunc("/account", handleAdminUpdateAccount).Methods("POST", "PUT")

	// User routes (require user session)
	userRouter := r.PathPrefix("/user").Subrouter()
	userRouter.Use(requireUserSession)
	userRouter.HandleFunc("/me", handleUserMe).Methods("GET")
	userRouter.HandleFunc("/rooms", handleUserRooms).Methods("GET")
	userRouter.HandleFunc("/api-keys", handleUserListAPIKeys).Methods("GET")
	userRouter.HandleFunc("/api-keys", handleUserCreateAPIKey).Methods("POST")
	userRouter.HandleFunc("/api-keys/request", handleUserRequestAPIAccess).Methods("POST")
	userRouter.HandleFunc("/api-keys/{id}", handleUserDeleteAPIKey).Methods("DELETE")

	// User auth (public)
	r.HandleFunc("/auth/user/register", handleUserRegister).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/user/login", handleUserLogin).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/user/logout", handleUserLogout).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/user/google", handleUserGoogleLogin).Methods("GET")
	r.HandleFunc("/auth/user/google/callback", handleUserGoogleCallback).Methods("GET")

	// API v1 routes (require API key)
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(requireAPIKey)
	apiRouter.HandleFunc("/room/create", handleAPICreateRoom).Methods("POST")
	apiRouter.HandleFunc("/room/{id}/link", handleAPIGetRoomLink).Methods("GET")
	apiRouter.HandleFunc("/room/{id}/files", handleAPIGetRoomFiles).Methods("GET")
	apiRouter.HandleFunc("/room/{id}/download-all", handleAPIDownloadAll).Methods("GET")

	// Clean URLs: /user -> user dashboard, /admin -> admin dashboard
	r.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/user/dashboard.html", http.StatusTemporaryRedirect)
	}).Methods("GET")
	r.HandleFunc("/user/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/user/dashboard.html", http.StatusTemporaryRedirect)
	}).Methods("GET")
	r.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/dashboard.html", http.StatusTemporaryRedirect)
	}).Methods("GET")
	r.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/dashboard.html", http.StatusTemporaryRedirect)
	}).Methods("GET")

	r.PathPrefix("/").Handler(middleware.CleanURLMiddleware(http.FileServer(http.Dir("./frontend"))))

	log.Printf("🚀 AmbilFile Server (Go) running on port %d", Port)
	return r
}
